package usecase

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strconv"
	"time"
	"vucdiagnostic/internal/domain/model"
)

// The input of the language model for the autobiography portrait. Only fields
// marked llm in the form pass, with option codes replaced by their texts. Two
// counted values carry no personal data and come from other fields: the age
// (from the birth date) and the number of children (from the records). The
// levels of potential come ready: the model only writes the text. The format
// follows the made-up questionnaires in Вики/Автобиография/Сравнение моделей/анкеты
// and is the same for every model.

// payloadScope reads one section or group record for the model.
type payloadScope struct {
	form   model.Form
	path   string
	fields map[string]model.FormField
	// asked holds every field the builder asked for and whether it may be sent.
	asked map[string]bool
}

func newPayloadScope(form model.Form, asked map[string]bool, path string, fields []model.FormField) payloadScope {
	scope := payloadScope{form: form, path: path, fields: map[string]model.FormField{}, asked: asked}
	for _, field := range fields {
		scope.fields[field.ID] = field
	}
	return scope
}

func (s payloadScope) group(id string) payloadScope {
	return newPayloadScope(s.form, s.asked, s.path+"."+id, s.fields[id].Fields)
}

func (s payloadScope) allowed(id string) bool {
	field, ok := s.fields[id]
	allowed := ok && field.LLM && field.Type != "group"
	s.asked[s.path+"."+id] = allowed
	return allowed
}

// code returns the stored value of a field marked llm and "" for any other field.
func (s payloadScope) code(a formAnswers, id string) string {
	if !s.allowed(id) {
		return ""
	}
	return a.text(id)
}

func (s payloadScope) option(id, value string) string {
	for _, option := range s.form.Options(s.fields[id]) {
		if option.ID == value {
			return option.Text
		}
	}
	return value
}

// text returns the value as the student saw it: a choice gives its option text.
func (s payloadScope) text(a formAnswers, id string) string {
	value := s.code(a, id)
	if value == "" || s.fields[id].Type != "choice" {
		return value
	}
	return s.option(id, value)
}

// named returns the text of a choice or what the student typed for "other".
func (s payloadScope) named(a formAnswers, id, other string) string {
	typed := s.text(a, other)
	if s.code(a, id) == "other" && typed != "" {
		return typed
	}
	return s.text(a, id)
}

// codes returns the chosen options of a multi field in the order of the form,
// so the same choice gives the same input whatever order it was clicked in.
func (s payloadScope) codes(a formAnswers, id string) []string {
	result := []string{}
	if !s.allowed(id) {
		return result
	}
	chosen := a.list(id)
	for _, option := range s.form.Options(s.fields[id]) {
		if slices.Contains(chosen, option.ID) {
			result = append(result, option.ID)
		}
	}
	return result
}

// texts returns the option texts of a multi field; "other" is replaced by what
// the student typed in the field other, when there is one.
func (s payloadScope) texts(a formAnswers, id, other string) []string {
	typed := ""
	if other != "" {
		typed = s.text(a, other)
	}
	result := []string{}
	for _, value := range s.codes(a, id) {
		if value == "other" && typed != "" {
			result = append(result, typed)
		} else {
			result = append(result, s.option(id, value))
		}
	}
	return result
}

// year returns a year as a number, or nil when it is blank.
func (s payloadScope) year(a formAnswers, id string) any {
	value := s.code(a, id)
	if value == "" {
		return nil
	}
	if number, err := strconv.Atoi(value); err == nil {
		return number
	}
	return value
}

// period is "с 2020-03 по 2021-08" or, while still there, "с 2023-05 по настоящее время".
func (s payloadScope) period(a formAnswers, from, to, still string) string {
	start, end := s.code(a, from), s.code(a, to)
	current := still != "" && s.code(a, still) == "yes"
	switch {
	case start == "":
		return ""
	case current:
		return "с " + start + " по настоящее время"
	case end == "":
		return "с " + start
	}
	return "с " + start + " по " + end
}

// put sets a value unless it is blank: the model gets no empty fields.
func put(target map[string]any, key string, value any) {
	switch v := value.(type) {
	case nil:
		return
	case string:
		if v == "" {
			return
		}
	case []string:
		if len(v) == 0 {
			return
		}
	}
	target[key] = value
}

// ageOn returns the full years on the Moscow date of asOf, or nil without a birth date.
func ageOn(birth string, asOf time.Time) any {
	date, err := time.Parse("2006-01-02", birth)
	if err != nil {
		return nil
	}
	year, month, day := asOf.In(moscow).Date()
	age := year - date.Year()
	if month < date.Month() || month == date.Month() && day < date.Day() {
		age--
	}
	return age
}

// autobiographyPayload is the model input of an autobiography. The age is
// counted on the Moscow date of asOf: the caller passes the submission time.
func autobiographyPayload(form model.Form, answers map[string]json.RawMessage, asOf time.Time) map[string]any {
	payload, _ := buildPayload(form, answers, asOf)
	return payload
}

// buildPayload also returns the fields it asked for, to check them against the llm marks.
func buildPayload(form model.Form, answers map[string]json.RawMessage, asOf time.Time) (map[string]any, map[string]bool) {
	asked := map[string]bool{}
	sections := map[string]payloadScope{}
	for _, section := range form.Sections {
		sections[section.ID] = newPayloadScope(form, asked, section.ID, section.Fields)
	}
	a := formAnswers(answers)
	payload := map[string]any{}

	// Section 1: only the age.
	put(payload, "возраст", ageOn(a.text("birth_date"), asOf))

	edu := sections["education"].group("education_list")
	schools := []any{}
	for _, r := range a.records("education_list") {
		record := map[string]any{}
		put(record, "тип", edu.text(r, "edu_type"))
		put(record, "профиль", edu.text(r, "edu_profile"))
		put(record, "год_окончания", edu.year(r, "edu_year"))
		put(record, "специальность", edu.text(r, "edu_speciality"))
		put(record, "окончил", edu.text(r, "edu_finished"))
		put(record, "медаль", edu.text(r, "edu_medal"))
		schools = append(schools, record)
	}
	payload["учёба_до_вуза"] = schools

	hse := sections["hse"]
	study := map[string]any{}
	put(study, "кампус", hse.text(a, "hse_campus"))
	put(study, "уровень", hse.text(a, "hse_level"))
	for _, id := range []string{"hse_faculty_msk", "hse_faculty_spb", "hse_faculty_nnov", "hse_faculty_perm"} {
		put(study, "факультет", hse.text(a, id))
	}
	put(study, "направление", hse.text(a, "hse_program"))
	put(study, "поступление", hse.text(a, "hse_start"))
	put(study, "окончание", hse.text(a, "hse_end"))
	payload["вшэ"] = study

	work := sections["work"].group("work_list")
	jobs := []any{}
	for _, r := range a.records("work_list") {
		record := map[string]any{}
		put(record, "сфера", work.text(r, "sphere"))
		put(record, "должность", work.text(r, "position"))
		put(record, "период", work.period(r, "from", "to", "still"))
		put(record, "причина_ухода", work.text(r, "leave_reason"))
		jobs = append(jobs, record)
	}
	payload["работа_до_вуза"] = jobs

	military := sections["military"]
	service := map[string]any{}
	put(service, "служил", military.text(a, "served"))
	put(service, "род_войск", military.text(a, "service_branch"))
	put(service, "период", military.period(a, "service_from", "service_to", ""))
	put(service, "должность", military.text(a, "service_position"))
	put(service, "звание", military.text(a, "service_rank"))
	payload["служба"] = service

	languages := sections["languages"]
	lang := languages.group("lang_list")
	known := []any{}
	if languages.code(a, "lang_has") == "yes" {
		for _, r := range a.records("lang_list") {
			record := map[string]any{}
			put(record, "язык", lang.named(r, "language", "language_other"))
			put(record, "степень", lang.text(r, "degree"))
			// Every detail is read whatever the practice, so that each field is asked for.
			details := map[string]map[string]any{
				"school":  {"программа": lang.text(r, "school_kind")},
				"courses": {"курсы": lang.text(r, "courses_what"), "длительность": lang.text(r, "courses_time")},
				"exam":    {"экзамен": lang.named(r, "exam", "exam_other"), "результат": lang.text(r, "exam_result"), "год": lang.year(r, "exam_year")},
				"abroad":  {"страна": lang.text(r, "abroad_country"), "срок": lang.text(r, "abroad_time")},
				"env":     {"страна": lang.text(r, "env_country"), "сколько_лет": lang.text(r, "env_years")},
				"work":    {"применение": lang.text(r, "work_use")},
			}
			practice := []any{}
			for _, kind := range lang.codes(r, "practice") {
				item := map[string]any{"вид": lang.option("practice", kind)}
				for key, value := range details[kind] {
					put(item, key, value)
				}
				practice = append(practice, item)
			}
			record["практика"] = practice
			known = append(known, record)
		}
	}
	payload["языки"] = known

	// Section 7: the status and the number of children, nothing about the spouse.
	family := sections["family"]
	children := 0
	if family.code(a, "children_has") == "yes" {
		children = len(a.records("children_list"))
	}
	marital := map[string]any{"детей": children}
	put(marital, "статус", family.text(a, "marital"))
	payload["семейное_положение"] = marital

	rel := sections["relatives"].group("rel_list")
	relatives := []any{}
	for _, r := range a.records("rel_list") {
		record := map[string]any{}
		put(record, "кем_приходится", rel.text(r, "kind"))
		put(record, "сведения", rel.text(r, "known"))
		put(record, "жив", rel.text(r, "alive"))
		put(record, "занятость", rel.text(r, "work"))
		put(record, "сфера", rel.text(r, "sphere"))
		put(record, "должность", rel.text(r, "position"))
		put(record, "иностранное_гражданство", rel.text(r, "foreign"))
		put(record, "страна_гражданства", rel.text(r, "foreign_country"))
		put(record, "судимость", rel.text(r, "crim"))
		put(record, "статья", rel.text(r, "crim_what"))
		put(record, "тяжкое", rel.text(r, "crim_grave"))
		relatives = append(relatives, record)
	}
	circle := map[string]any{"родственники": relatives}
	put(circle, "кто_воспитывал", sections["relatives"].text(a, "upbringing"))
	payload["семья"] = circle

	special := sections["special"]
	trip := special.group("trips_list")
	trips := []any{}
	if special.code(a, "trips_has") == "yes" {
		for _, r := range a.records("trips_list") {
			record := map[string]any{}
			put(record, "страна", trip.text(r, "country"))
			put(record, "год", trip.year(r, "year"))
			put(record, "цель", trip.text(r, "purpose"))
			trips = append(trips, record)
		}
	}
	payload["поездки_за_границу"] = trips
	// The student's own criminal record: only the fact.
	switch special.code(a, "crim_has") {
	case "yes":
		payload["уголовная_ответственность"] = "привлекался"
	case "no":
		payload["уголовная_ответственность"] = "не привлекался"
	}

	awards := sections["awards"]
	award := awards.group("awards_list")
	given := []any{}
	if awards.code(a, "awards_has") == "yes" {
		for _, r := range a.records("awards_list") {
			record := map[string]any{}
			put(record, "название", award.text(r, "award_name"))
			put(record, "год", award.year(r, "award_year"))
			given = append(given, record)
		}
	}
	payload["награды"] = given

	sport := sections["sport"]
	kind := sport.group("sport_list")
	sports := []any{}
	if sport.code(a, "sport_has") == "yes" {
		for _, r := range a.records("sport_list") {
			record := map[string]any{}
			put(record, "вид", kind.named(r, "sport_kind", "sport_kind_other"))
			put(record, "сейчас", kind.text(r, "sport_now"))
			put(record, "стаж", kind.text(r, "sport_years"))
			put(record, "разряд", kind.text(r, "sport_rank"))
			put(record, "соревнования", kind.text(r, "sport_level"))
			put(record, "достижения", kind.text(r, "sport_ach"))
			sports = append(sports, record)
		}
	}
	athletics := map[string]any{"занятия": sports}
	put(athletics, "гто", sport.text(a, "gto"))
	payload["спорт"] = athletics

	creative := sections["creative"]
	direction := creative.group("creative_list")
	arts := []any{}
	if creative.code(a, "creative_has") == "yes" {
		for _, r := range a.records("creative_list") {
			record := map[string]any{}
			put(record, "направление", direction.text(r, "direction"))
			put(record, "уточнение", direction.text(r, "detail"))
			put(record, "обучение", direction.text(r, "learning"))
			put(record, "конкурсы", direction.text(r, "level"))
			put(record, "достижения", direction.text(r, "ach"))
			arts = append(arts, record)
		}
	}
	payload["творчество"] = arts

	patriotic := sections["patriotic"]
	patriot := map[string]any{}
	put(patriot, "участвовал", patriotic.text(a, "patriot_has"))
	put(patriot, "формы", patriotic.texts(a, "patriot_what", "patriot_other"))
	put(patriot, "роль", patriotic.text(a, "patriot_role"))
	put(patriot, "длительность", patriotic.text(a, "patriot_term"))
	put(patriot, "уровень", patriotic.text(a, "patriot_level"))
	put(patriot, "навыки", patriotic.texts(a, "patriot_skills", ""))
	put(patriot, "достижения", patriotic.text(a, "patriot_ach"))
	payload["военно_патриотическое"] = patriot

	science := sections["science"]
	research := map[string]any{}
	put(research, "занимался", science.text(a, "science_has"))
	put(research, "формы", science.texts(a, "science_forms", ""))
	put(research, "область", science.text(a, "science_area"))
	put(research, "уровень", science.text(a, "science_level"))
	put(research, "результат", science.text(a, "science_result"))
	put(research, "достижения", science.text(a, "science_ach"))
	payload["наука"] = research

	volunteering := sections["volunteering"]
	help := map[string]any{}
	put(help, "занимался", volunteering.text(a, "vol_has"))
	put(help, "направления", volunteering.texts(a, "vol_dirs", "vol_other"))
	put(help, "длительность", volunteering.text(a, "vol_term"))
	put(help, "роль", volunteering.text(a, "vol_role"))
	payload["волонтёрство"] = help

	hobbies := sections["hobbies"]
	interests := hobbies.texts(a, "hobbies", "")
	if own := hobbies.text(a, "hobby_own"); own != "" {
		interests = append(interests, own)
	}
	payload["увлечения"] = interests

	levels := map[string]any{}
	for _, area := range autobiographyPotential(form, answers).Areas {
		levels[lowerFirst(area.Title)] = area.LevelText
	}
	payload["уровни"] = levels
	return payload, asked
}

// canonicalJSON writes a value with sorted keys, without spaces and without
// escaping: the same input always gives the same bytes.
func canonicalJSON(value any) []byte {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n"))
}

// payloadFingerprint is the hex sha256 of the canonical input. The portrait
// is to be stored under it, so equal anonymised answers get the same portrait.
func payloadFingerprint(payload map[string]any) string {
	sum := sha256.Sum256(canonicalJSON(payload))
	return hex.EncodeToString(sum[:])
}

// submissionPayload is the model input of a submitted autobiography and its
// fingerprint. The age is counted on the day of submission, so the input does
// not change after the student's birthday.
func submissionPayload(form model.Form, submission model.FormSubmission) (map[string]any, string, error) {
	if submission.FormID != autobiographyForm || !submission.Completed {
		return nil, "", problem(Invalid, "Сведения для модели собираются только по сданной автобиографии.")
	}
	submitted, err := time.Parse(time.RFC3339, submission.CompletedAt)
	if err != nil {
		return nil, "", problem(Invalid, "У анкеты нет даты сдачи.")
	}
	payload := autobiographyPayload(form, submission.Answers, submitted)
	return payload, payloadFingerprint(payload), nil
}
