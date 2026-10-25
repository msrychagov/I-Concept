package usecase

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"vucdiagnostic/internal/domain/model"
)

// The autobiography for the personal file: the paper form written in the
// first person from the answers of a submitted autobiography, in the order of
// the form (Вики/Автобиография/Шаблон автобиографии.md, approved 25 October
// 2026). Only what the paper form asks: sport, hobbies and the fields added
// over the form stay out. The text is made anew on every request and is never
// stored; no language model takes part.

// AutobiographyText is the printed autobiography.
type AutobiographyText struct {
	Student    string   `json:"student"`
	Paragraphs []string `json:"paragraphs"`
	// Date is the Moscow day the autobiography was submitted, «15 сентября 2024 г.».
	Date    string `json:"date"`
	Surname string `json:"surname"`
}

// textReader reads the answers of one section or group record of the form.
type textReader struct {
	form   model.Form
	fields map[string]model.FormField
	a      formAnswers
}

func sectionReader(form model.Form, id string, answers formAnswers) textReader {
	r := textReader{form: form, fields: map[string]model.FormField{}, a: answers}
	for _, section := range form.Sections {
		if section.ID == id {
			for _, field := range section.Fields {
				r.fields[field.ID] = field
			}
		}
	}
	return r
}

// records returns a reader for every record of a group field.
func (r textReader) records(id string) []textReader {
	list := []textReader{}
	for _, record := range r.a.records(id) {
		inner := textReader{form: r.form, fields: map[string]model.FormField{}, a: record}
		for _, field := range r.fields[id].Fields {
			inner.fields[field.ID] = field
		}
		list = append(list, inner)
	}
	return list
}

func (r textReader) code(id string) string { return r.a.text(id) }

func (r textReader) is(id string, values ...string) bool { return r.a.is(id, values...) }

// typed is what the student typed, on one line and without the full stop at
// the end: the template puts its own punctuation after it.
func (r textReader) typed(id string) string {
	return strings.TrimRight(strings.Join(strings.Fields(r.a.text(id)), " "), ". ;,")
}

// option is the text of the chosen option, "" when nothing is chosen.
func (r textReader) option(id string) string {
	value := r.a.text(id)
	for _, option := range r.form.Options(r.fields[id]) {
		if option.ID == value {
			return option.Text
		}
	}
	return ""
}

var (
	monthsOf = []string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}
	monthsIn = []string{"январе", "феврале", "марте", "апреле", "мае", "июне", "июле", "августе", "сентябре", "октябре", "ноябре", "декабре"}
	monthsTo = []string{"январь", "февраль", "март", "апрель", "май", "июнь", "июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь"}
)

// dayText is «14 мая 2001 года» for 2001-05-14.
func dayText(value string) string {
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return fmt.Sprintf("%d %s %d года", date.Day(), monthsOf[date.Month()-1], date.Year())
}

// monthOf splits 2024-09 into the month (1–12) and the year.
func monthOf(value string) (int, int, bool) {
	date, err := time.Parse("2006-01", value)
	if err != nil {
		return 0, 0, false
	}
	return int(date.Month()), date.Year(), true
}

// inMonth is «сентябре 2024 года»: «поступил в сентябре 2024 года».
func inMonth(value string) string {
	m, y, ok := monthOf(value)
	if !ok {
		return value
	}
	return fmt.Sprintf("%s %d года", monthsIn[m-1], y)
}

// monthName is «июнь 2028 года»: «окончание обучения — июнь 2028 года».
func monthName(value string) string {
	m, y, ok := monthOf(value)
	if !ok {
		return value
	}
	return fmt.Sprintf("%s %d года", monthsTo[m-1], y)
}

// periodText is «с июня 2020 по апрель 2021 года», «с июня по август 2024 года»
// or, while still there, «с июля 2022 года по настоящее время».
func periodText(from, to string, still bool) string {
	m1, y1, ok := monthOf(from)
	if !ok {
		return ""
	}
	if still {
		return fmt.Sprintf("с %s %d года по настоящее время", monthsOf[m1-1], y1)
	}
	m2, y2, ok := monthOf(to)
	switch {
	case !ok:
		return fmt.Sprintf("с %s %d года", monthsOf[m1-1], y1)
	case y1 == y2:
		return fmt.Sprintf("с %s по %s %d года", monthsOf[m1-1], monthsTo[m2-1], y2)
	}
	return fmt.Sprintf("с %s %d по %s %d года", monthsOf[m1-1], y1, monthsTo[m2-1], y2)
}

// midSentence writes a rank, a language or a degree in the middle of a sentence.
func midSentence(text string) string {
	letters := []rune(text)
	if len(letters) == 0 {
		return text
	}
	// An abbreviation («ВДВ») keeps its capitals.
	if len(letters) > 1 && unicode.IsUpper(letters[1]) {
		return text
	}
	letters[0] = unicode.ToLower(letters[0])
	return string(letters)
}

// quoted puts a typed name in «», unless the student already did.
func quoted(text string) string {
	for _, mark := range []string{"«", "\"", "'", "“", "„"} {
		if text == "" || strings.HasPrefix(text, mark) {
			return text
		}
	}
	return "«" + text + "»"
}

// joinNonEmpty joins the parts that are not empty.
func joinNonEmpty(sep string, parts ...string) string {
	kept := []string{}
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, sep)
}

// containsFold says whether a typed place already names the region.
func containsFold(text, part string) bool {
	return part != "" && strings.Contains(strings.ToLower(text), strings.ToLower(part))
}

// autobiographyText writes the autobiography of a submitted questionnaire.
func autobiographyText(form model.Form, account model.Account, submission model.FormSubmission) AutobiographyText {
	answers := formAnswers(submission.Answers)
	name := strings.Join(strings.Fields(account.Name), " ")
	surname := ""
	if words := strings.Fields(name); len(words) > 0 {
		surname = words[0]
	}
	paragraphs := []string{aboutParagraph(sectionReader(form, "about", answers), name)}
	paragraphs = append(paragraphs, educationParagraph(sectionReader(form, "education", answers)))
	paragraphs = append(paragraphs, workParagraph(sectionReader(form, "work", answers)))
	paragraphs = append(paragraphs, serviceParagraph(sectionReader(form, "military", answers)))
	paragraphs = append(paragraphs, universityParagraph(sectionReader(form, "hse", answers)))
	paragraphs = append(paragraphs, languagesParagraph(sectionReader(form, "languages", answers)))
	paragraphs = append(paragraphs, familyParagraph(sectionReader(form, "family", answers)))
	relatives := sectionReader(form, "relatives", answers)
	paragraphs = append(paragraphs, relativesParagraphs(relatives)...)
	paragraphs = append(paragraphs, specialParagraph(sectionReader(form, "special", answers), relatives))
	paragraphs = append(paragraphs, awardsParagraph(sectionReader(form, "awards", answers)))
	paragraphs = append(paragraphs, addressParagraph(sectionReader(form, "address", answers)))
	paragraphs = append(paragraphs, registrationParagraph(sectionReader(form, "military", answers)))
	kept := []string{}
	for _, paragraph := range paragraphs {
		if paragraph != "" {
			kept = append(kept, paragraph)
		}
	}
	return AutobiographyText{Student: name, Paragraphs: kept, Date: submittedDay(submission), Surname: surname}
}

// submittedDay is the Moscow day of the submission, «15 сентября 2024 г.».
func submittedDay(submission model.FormSubmission) string {
	at, err := time.Parse(time.RFC3339, cmp.Or(submission.CompletedAt, submission.Updated))
	if err != nil {
		return ""
	}
	at = at.In(moscow)
	return fmt.Sprintf("%d %s %d г.", at.Day(), monthsOf[at.Month()-1], at.Year())
}

var nameChangeReasons = map[string]string{
	"marriage": "в связи со вступлением в брак",
	"own":      "по собственному желанию",
	"adoption": "в связи с усыновлением (установлением отцовства)",
	"other":    "по иной причине",
}

func aboutParagraph(r textReader, name string) string {
	text := "Я, " + name
	if r.is("name_changed", "yes") {
		text += " (прежние ФИО — " + r.typed("prev_names") + ", изменены " + nameChangeReasons[r.code("name_change_reason")] + ")"
	}
	text += ", родился " + dayText(r.code("birth_date"))
	place := r.typed("birth_place")
	where := r.option("birth_country")
	if r.is("birth_country", "RU") {
		where = r.option("birth_region")
	}
	if containsFold(place, where) {
		where = ""
	}
	if place = joinNonEmpty(", ", place, where); place != "" {
		text += "; место рождения — " + place
	}
	text += "."
	nationality := maleNationality[r.code("nationality")]
	if r.is("nationality", "other") {
		nationality = r.typed("nationality_other")
	}
	if nationality != "" {
		text += " По национальности — " + nationality + "."
	}
	return text
}

// The kinds of school as they stand after «окончил», when the name the
// student typed does not say it, and after «учился в».
var (
	schoolKinds   = map[string]string{"school": "школу", "lyceum": "лицей", "gymnasium": "гимназию", "cadet": "кадетскую школу", "suvorov": "училище", "college": "колледж", "university": "вуз", "other": "учебное заведение"}
	schoolWords   = []string{"школ", "лице", "гимнази", "корпус", "училищ", "колледж", "техникум", "университет", "институт", "академи", "вуз"}
	studiedPlaces = []struct{ stem, place string }{{"техникум", "в техникуме"}, {"колледж", "в колледже"}, {"университет", "в университете"}, {"институт", "в институте"}, {"академи", "в академии"}}
)

// schoolName is the name as typed when it says what the school is, and
// «школу № 15» or «школу МБОУ СОШ № 15» when it does not.
func schoolName(kind, name string) string {
	lower := strings.ToLower(name)
	for _, word := range schoolWords {
		if strings.Contains(lower, word) {
			return name
		}
	}
	if strings.IndexFunc(name, func(r rune) bool { return !unicode.IsDigit(r) }) < 0 {
		return schoolKinds[kind] + " № " + name
	}
	return joinNonEmpty(" ", schoolKinds[kind], name)
}

func educationParagraph(r textReader) string {
	schools := r.records("education_list")
	slices.SortStableFunc(schools, func(x, y textReader) int { return cmp.Compare(x.code("edu_year"), y.code("edu_year")) })
	sentences := []string{}
	for _, s := range schools {
		region := s.option("edu_region")
		if s.is("edu_region", "abroad") || containsFold(s.typed("school_city"), region) {
			region = ""
		}
		place := joinNonEmpty(", ", s.typed("school_city"), region)
		if place != "" {
			place = " (" + place + ")"
		}
		speciality := ""
		if s.is("edu_type", "college", "university") && s.typed("edu_speciality") != "" {
			speciality = ", специальность " + quoted(s.typed("edu_speciality"))
		}
		name := s.typed("school_name")
		if s.is("edu_type", "college", "university") && s.is("edu_finished", "no") {
			where := map[string]string{"college": "в колледже", "university": "в вузе"}[s.code("edu_type")]
			for _, p := range studiedPlaces {
				if strings.Contains(strings.ToLower(name), p.stem) {
					where = p.place
					break
				}
			}
			sentences = append(sentences, fmt.Sprintf("До %s года учился %s — %s%s%s; обучение не окончил.", s.code("edu_year"), where, name, place, speciality))
			continue
		}
		sentences = append(sentences, fmt.Sprintf("В %s году окончил %s%s%s.", s.code("edu_year"), schoolName(s.code("edu_type"), name), place, speciality))
	}
	return strings.Join(sentences, " ")
}

var leaveReasons = map[string]string{
	"study":    "уволился в связи с поступлением в вуз",
	"contract": "уволился по окончании договора",
	"own":      "уволился по собственному желанию",
	"move":     "уволился в связи с переездом",
	"cut":      "уволен по сокращению",
}

func workParagraph(r textReader) string {
	if !r.is("work_has", "yes") {
		return "До поступления в университет не работал."
	}
	jobs := r.records("work_list")
	slices.SortStableFunc(jobs, func(x, y textReader) int { return cmp.Compare(x.code("from"), y.code("from")) })
	parts := []string{}
	for _, job := range jobs {
		still := job.is("still", "yes")
		text := joinNonEmpty(" — ", periodText(job.code("from"), job.code("to"), still), joinNonEmpty(", ", job.typed("org"), job.typed("position")))
		if !still {
			text = joinNonEmpty(", ", text, leaveReasons[job.code("leave_reason")])
		}
		parts = append(parts, text)
	}
	return "До поступления в университет работал: " + strings.Join(parts, "; ") + "."
}

func serviceParagraph(r textReader) string {
	if !r.is("served", "yes") {
		return "В Вооружённых силах не служил."
	}
	branch := r.option("service_branch")
	if r.is("service_branch", "other") {
		branch = ""
	}
	details := []string{branch}
	for _, item := range []struct{ label, value string }{
		{"место службы", r.typed("service_place")},
		{"должность", r.typed("service_position")},
		{"ВУС", r.typed("service_vus")},
		{"воинское звание", midSentence(r.option("service_rank"))},
	} {
		if item.value != "" {
			details = append(details, item.label+" — "+item.value)
		}
	}
	return upperFirst(joinNonEmpty(" ", periodText(r.code("service_from"), r.code("service_to"), false), "проходил военную службу в Вооружённых силах: "+joinNonEmpty("; ", details...)+"."))
}

var (
	hseCampuses = map[string]string{"msk": "НИУ ВШЭ", "spb": "НИУ ВШЭ — Санкт-Петербург", "nnov": "НИУ ВШЭ — Нижний Новгород", "perm": "НИУ ВШЭ — Пермь"}
	hseLevels   = map[string]string{"bakalavriat": "бакалавриата", "spetsialitet": "специалитета", "magistratura": "магистратуры"}
)

func universityParagraph(r textReader) string {
	campus := r.code("hse_campus")
	text := joinNonEmpty(" ", "В "+inMonth(r.code("hse_start")), "поступил в "+cmp.Or(hseCampuses[campus], "НИУ ВШЭ"))
	if r.code("hse_start") == "" {
		text = "Поступил в " + cmp.Or(hseCampuses[campus], "НИУ ВШЭ")
	}
	program := r.typed("hse_program")
	if program != "" {
		text += " на программу " + joinNonEmpty(" ", hseLevels[r.code("hse_level")], quoted(program))
	}
	faculty := "hse_faculty_" + campus
	if r.code(faculty) != "" && !r.is(faculty, "other") {
		text += " (" + r.option(faculty) + ")"
	}
	text += "."
	if r.code("hse_end") != "" {
		text += " Окончание обучения — " + monthName(r.code("hse_end")) + "."
	}
	return text
}

func languagesParagraph(r textReader) string {
	if !r.is("lang_has", "yes") {
		return "Иностранными языками не владею."
	}
	parts := []string{}
	for _, l := range r.records("lang_list") {
		language := midSentence(l.option("language"))
		if l.is("language", "other") {
			language = l.typed("language_other")
		}
		parts = append(parts, joinNonEmpty(" — ", language, midSentence(l.option("degree"))))
	}
	if len(parts) == 1 {
		return "Владею иностранным языком: " + parts[0] + "."
	}
	return "Владею иностранными языками: " + strings.Join(parts, "; ") + "."
}

var maritalStatus = map[string]string{"single": "холост", "married": "женат", "divorced": "разведён"}

func familyParagraph(r textReader) string {
	sentences := []string{}
	if status := maritalStatus[r.code("marital")]; status != "" {
		sentences = append(sentences, "Семейное положение — "+status+".")
	}
	if r.is("marital", "married") {
		last, maiden := r.typed("spouse_last"), r.typed("spouse_maiden")
		spouse := last
		if maiden != "" && !strings.EqualFold(maiden, last) {
			spouse += " (девичья фамилия — " + maiden + ")"
		}
		spouse = joinNonEmpty(" ", spouse, r.typed("spouse_first"), r.typed("spouse_middle"))
		details := []string{spouse + ", " + dayText(r.code("spouse_birth")) + " рождения"}
		if place := r.typed("spouse_birthplace"); place != "" {
			details[0] += ", место рождения — " + place
		}
		if job := r.typed("spouse_job"); job != "" {
			details = append(details, "место работы и должность — "+job)
		}
		if r.is("spouse_with_me", "yes") {
			details = append(details, "проживает со мной")
		} else if address := r.typed("spouse_address"); address != "" {
			details = append(details, "место жительства — "+address)
		}
		sentences = append(sentences, "Супруга — "+strings.Join(details, "; ")+".")
		certificate := []string{}
		if issuer := r.typed("cert_issued_by"); issuer != "" {
			certificate = append(certificate, "кем выдано — "+issuer)
		}
		if r.code("cert_date") != "" {
			certificate = append(certificate, "дата выдачи — "+dayText(r.code("cert_date")))
		}
		if number := r.typed("cert_number"); number != "" || len(certificate) > 0 {
			sentences = append(sentences, joinNonEmpty("; ", "Свидетельство о браке "+number, strings.Join(certificate, "; "))+".")
		}
	}
	if !r.is("children_has", "yes") {
		return strings.Join(append(sentences, "Детей не имею."), " ")
	}
	children := []string{}
	for _, child := range r.records("children_list") {
		children = append(children, joinNonEmpty(", ", child.typed("child_name"), dayText(child.code("child_birth"))+" рождения"))
	}
	return strings.Join(append(sentences, "Дети: "+strings.Join(children, "; ")+"."), " ")
}

// relativeKind is how a relative is named: at the start of the paragraph,
// inside a list, after «сведений о» and whether «умер» or «умерла» fits.
type relativeKind struct {
	title, inList, about, died string
}

var relativeKinds = map[string]relativeKind{
	"father":      {"Отец", "отец", "об отце", "умер"},
	"mother":      {"Мать", "мать", "о матери", "умерла"},
	"brother":     {"Брат", "брат", "о брате", "умер"},
	"sister":      {"Сестра", "сестра", "о сестре", "умерла"},
	"guardian":    {"Опекун (приёмный родитель)", "опекун (приёмный родитель)", "об опекуне (приёмном родителе)", ""},
	"grandmother": {"Бабушка", "бабушка", "о бабушке", "умерла"},
	"grandfather": {"Дедушка", "дедушка", "о дедушке", "умер"},
	"other":       {"Близкий родственник", "близкий родственник", "о близком родственнике", ""},
}

// relativeOrder is the order of the form: father, mother, brothers, sisters…
var relativeOrder = []string{"father", "mother", "brother", "sister", "guardian", "grandmother", "grandfather", "other"}

// sortedRelatives returns the relatives in the order of the form.
func sortedRelatives(r textReader) []textReader {
	if !r.is("rel_has", "yes") {
		return nil
	}
	list := r.records("rel_list")
	slices.SortStableFunc(list, func(x, y textReader) int {
		return cmp.Compare(slices.Index(relativeOrder, x.code("kind")), slices.Index(relativeOrder, y.code("kind")))
	})
	return list
}

func kindOf(r textReader) relativeKind {
	kind, ok := relativeKinds[r.code("kind")]
	if !ok {
		return relativeKinds["other"]
	}
	return kind
}

// living says whether a relative is known and alive: only then the form asks
// about the work, the citizenship and a criminal record.
func living(r textReader) bool { return r.is("known", "yes") && r.is("alive", "yes") }

func relativesParagraphs(r textReader) []string {
	if !r.is("rel_has", "yes") {
		return []string{"Близких родственников не имею."}
	}
	paragraphs := []string{"Близкие родственники:"}
	for _, relative := range sortedRelatives(r) {
		kind := kindOf(relative)
		if !relative.is("known", "yes") {
			paragraphs = append(paragraphs, "Сведений "+kind.about+" не имею.")
			continue
		}
		text := kind.title + " — " + joinNonEmpty(", ", relative.typed("name"), dayText(relative.code("birth"))+" рождения")
		if !relative.is("alive", "yes") {
			if kind.died != "" {
				text += ", " + kind.died + " в " + relative.code("death_year") + " году."
			} else {
				text += ", год смерти — " + relative.code("death_year") + "."
			}
			paragraphs = append(paragraphs, text)
			continue
		}
		switch relative.code("work") {
		case "works":
			text += ", место работы и должность — " + joinNonEmpty(", ", relative.typed("org"), relative.typed("position"))
		case "pensioner":
			text += ", на пенсии"
		case "student":
			text += ", учится"
		case "none":
			text += ", не работает"
		}
		if relative.is("with_me", "yes") {
			text += "; проживает со мной"
		} else if address := relative.typed("address"); address != "" {
			text += "; проживает по адресу: " + address
		}
		paragraphs = append(paragraphs, text+".")
	}
	return paragraphs
}

var tripPurposes = map[string]string{"tourism": "туризм", "study": "учёба или стажировка", "work": "работа", "relatives": "к родственникам", "sport": "соревнования", "lived": "жил постоянно"}

func specialParagraph(r, relatives textReader) string {
	sentences := []string{}
	if r.is("trips_has", "yes") {
		trips := r.records("trips_list")
		slices.SortStableFunc(trips, func(x, y textReader) int { return cmp.Compare(x.code("year"), y.code("year")) })
		parts := []string{}
		for _, trip := range trips {
			year := ""
			if trip.code("year") != "" {
				year = trip.code("year") + " год"
			}
			parts = append(parts, joinNonEmpty(" — ", trip.option("country"), joinNonEmpty(", ", year, tripPurposes[trip.code("purpose")])))
		}
		sentences = append(sentences, "За границей был: "+strings.Join(parts, "; ")+".")
	} else {
		sentences = append(sentences, "За границей не был.")
	}
	foreign, criminal := []string{}, []string{}
	for _, relative := range sortedRelatives(relatives) {
		if !living(relative) {
			continue
		}
		who := joinNonEmpty(" ", kindOf(relative).inList, relative.typed("name"))
		if relative.is("foreign", "yes") {
			foreign = append(foreign, joinNonEmpty(" — ", who, relative.option("foreign_country")))
		}
		if relative.is("crim", "yes") {
			when := ""
			if relative.code("crim_year") != "" {
				when = "в " + relative.code("crim_year") + " году"
			}
			measure := ""
			if relative.typed("crim_measure") != "" {
				measure = "мера наказания — " + relative.typed("crim_measure")
			}
			criminal = append(criminal, joinNonEmpty(" — ", who, joinNonEmpty(", ", when, relative.typed("crim_what"), measure)))
		}
	}
	switch len(foreign) {
	case 0:
		sentences = append(sentences, "Близких родственников, имеющих иностранное гражданство или подданство, не имею.")
	case 1:
		sentences = append(sentences, "Из близких родственников иностранное гражданство имеет: "+foreign[0]+".")
	default:
		sentences = append(sentences, "Из близких родственников иностранное гражданство имеют: "+strings.Join(foreign, "; ")+".")
	}
	own := r.is("crim_has", "yes")
	switch {
	case !own && len(criminal) == 0:
		sentences = append(sentences, "Ни я, ни мои близкие родственники к уголовной ответственности не привлекались.")
	case own:
		text := "привлекался к уголовной ответственности"
		if r.code("crim_year") != "" {
			text = "В " + r.code("crim_year") + " году " + text
		}
		text = upperFirst(text)
		if what := r.typed("crim_what"); what != "" {
			text += ": " + what
		}
		if measure := r.typed("crim_measure"); measure != "" {
			text += "; мера наказания — " + measure
		}
		sentences = append(sentences, text+".")
		if len(criminal) == 0 {
			sentences = append(sentences, "Близкие родственники к уголовной ответственности не привлекались.")
		}
	default:
		sentences = append(sentences, "Я к уголовной ответственности не привлекался.")
	}
	if len(criminal) > 0 {
		sentences = append(sentences, "Из близких родственников к уголовной ответственности привлекались: "+strings.Join(criminal, "; ")+".")
	}
	return strings.Join(sentences, " ")
}

func awardsParagraph(r textReader) string {
	if !r.is("awards_has", "yes") {
		return "Государственных наград не имею."
	}
	parts := []string{}
	for _, award := range r.records("awards_list") {
		year := ""
		if award.code("award_year") != "" {
			year = award.code("award_year") + " год"
		}
		parts = append(parts, joinNonEmpty(" — ", award.typed("award_name"), year))
	}
	if len(parts) == 1 {
		return "Награждён государственной наградой: " + parts[0] + "."
	}
	return "Награждён государственными наградами: " + strings.Join(parts, "; ") + "."
}

var livingPlaces = map[string]string{"dorm": "в общежитии", "rent": "в съёмном жилье", "relatives": "у родственников"}

func addressParagraph(r textReader) string {
	address := r.typed("reg_address")
	region := r.option("reg_region")
	if containsFold(address, region) {
		region = ""
	}
	registered := joinNonEmpty(", ", region, address)
	if !r.is("lives_at_reg", "no") {
		return "Постоянно зарегистрирован и проживаю по адресу: " + registered + "."
	}
	return "Постоянно зарегистрирован по адресу: " + registered + ". " + joinNonEmpty(" ", "Проживаю", livingPlaces[r.code("live_kind")]) + " по адресу: " + r.typed("live_address") + "."
}

// commissariat is «военном комиссариате Ленинского района г. Воронежа
// (Воронежская область)»: the student types the district after the hint
// «Военный комиссариат …»; a name typed in full loses its first words.
func commissariat(district, region string) string {
	lower := strings.ToLower(district)
	for _, prefix := range []string{"военный комиссариат", "военного комиссариата", "военном комиссариате", "военкомат"} {
		if strings.HasPrefix(lower, prefix) {
			district = strings.TrimSpace(string([]rune(district)[len([]rune(prefix)):]))
			break
		}
	}
	if containsFold(district, region) {
		region = ""
	}
	text := joinNonEmpty(" ", "военном комиссариате", district)
	if region != "" {
		text += " (" + region + ")"
	}
	return text
}

func registrationParagraph(r textReader) string {
	text := "На воинском учёте состою в " + commissariat(r.typed("vk_district"), r.option("vk_region")) + "."
	if r.is("vk_same", "no") {
		return text + " До поступления в университет состоял на учёте в " + commissariat(r.typed("vk_prev_district"), r.option("vk_prev_region")) + "."
	}
	return text + " До поступления в университет состоял на учёте там же."
}

// maleNationality is the nationality of the form, «Русские», as one man
// writes it of himself: «русский».
var maleNationality = map[string]string{
	"abaziny": "абазин", "abkhazy": "абхаз", "avartsy": "аварец", "aguly": "агул", "adygeytsy": "адыгеец",
	"azerbaydzhantsy": "азербайджанец", "altaytsy": "алтаец", "amerikantsy": "американец", "anglichane": "англичанин",
	"araby": "араб", "armyane": "армянин", "assiriytsy": "ассириец", "afgantsy": "афганец",
	"afroamerikantsy": "афроамериканец", "balkartsy": "балкарец", "bashkiry": "башкир", "belorusy": "белорус",
	"bolgary": "болгарин", "buryaty": "бурят", "vengry": "венгр", "vepsy": "вепс", "vetnamtsy": "вьетнамец",
	"gorskie_evrei": "горский еврей", "greki": "грек", "gruziny": "грузин", "dargintsy": "даргинец",
	"dolgany": "долганин", "evrei": "еврей", "ezidy": "езид", "ingushi": "ингуш", "indiytsy": "индиец",
	"ispantsy": "испанец", "italyantsy": "итальянец", "kabardintsy": "кабардинец", "kazakhi": "казах",
	"kalmyki": "калмык", "karaimy": "караим", "karachaevtsy": "карачаевец", "karely": "карел", "kirgizy": "киргиз",
	"kitaytsy": "китаец", "komi": "коми", "koreytsy": "кореец", "koryaki": "коряк", "krymskie_tatary": "крымский татарин",
	"krymchaki": "крымчак", "kumyki": "кумык", "kurdy": "курд", "laktsy": "лакец", "latyshi": "латыш",
	"lezginy": "лезгин", "litovtsy": "литовец", "mansi": "манси", "mariytsy": "мариец", "moldavane": "молдаванин",
	"mongoly": "монгол", "mordva": "мордвин", "nanaytsy": "нанаец", "nemtsy": "немец", "nentsy": "ненец",
	"nogaytsy": "ногаец", "osetiny": "осетин", "persy": "перс", "polyaki": "поляк", "rumyny": "румын",
	"russkie": "русский", "rutultsy": "рутулец", "saamy": "саам", "serby": "серб", "slovaki": "словак",
	"tabasarany": "табасаран", "tadzhiki": "таджик", "tatary": "татарин", "tati": "тат", "teleuty": "телеут",
	"tofalary": "тофалар", "tuvintsy": "тувинец", "turki": "турок", "turki_meskhetintsy": "турок-месхетинец",
	"turkmeny": "туркмен", "udmurty": "удмурт", "uzbeki": "узбек", "ukraintsy": "украинец", "finny": "финн",
	"frantsuzy": "француз", "khakasy": "хакас", "khanty": "ханты", "khorvaty": "хорват", "tsakhury": "цахур",
	"tsygane": "цыган", "cherkesy": "черкес", "chekhi": "чех", "chechentsy": "чеченец", "chuvashi": "чуваш",
	"chukchi": "чукча", "shortsy": "шорец", "evenki": "эвенк", "eveny": "эвен", "estontsy": "эстонец",
	"yakuty": "якут", "yapontsy": "японец",
}

// AutobiographyTextFor returns the printed autobiography of a submitted
// autobiography of the teacher's assignments.
func (a *Service) AutobiographyTextFor(teacherID, id string) (AutobiographyText, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return AutobiographyText{}, err
	}
	for _, submission := range db.Submissions {
		if submission.ID != id || submission.FormID != autobiographyForm || !submission.Completed || !teacherOwnsSubmission(db, teacherID, submission) {
			continue
		}
		return a.autobiographyOf(db, submission)
	}
	return AutobiographyText{}, missingAutobiography()
}

// OwnAutobiographyText returns the student's own submitted autobiography.
func (a *Service) OwnAutobiographyText(loginToken, id string) (AutobiographyText, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return AutobiographyText{}, err
	}
	student, ok := accountIn(db, loginToken, "student")
	if !ok {
		return AutobiographyText{}, problem(Unauthenticated, "Войдите в кабинет.")
	}
	for _, submission := range db.Submissions {
		if submission.ID == id && submission.StudentID == student.ID && submission.FormID == autobiographyForm && submission.Completed {
			return a.autobiographyOf(db, submission)
		}
	}
	return AutobiographyText{}, missingAutobiography()
}

func missingAutobiography() error {
	return problem(NotFound, "Сданная автобиография не найдена.")
}

func (a *Service) autobiographyOf(db model.Database, submission model.FormSubmission) (AutobiographyText, error) {
	form, known := a.forms[submission.FormID]
	if !known {
		return AutobiographyText{}, missingAutobiography()
	}
	for _, account := range db.Students {
		if account.ID == submission.StudentID {
			return autobiographyText(form, account, submission), nil
		}
	}
	return AutobiographyText{}, missingAutobiography()
}
