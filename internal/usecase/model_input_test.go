package usecase

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	"vucdiagnostic/internal/domain/model"
)

// answersOf stores test values the way the form saves them: an unknown option
// fails the test, a hidden field is dropped.
func answersOf(t *testing.T, form model.Form, values answerSet) map[string]json.RawMessage {
	t.Helper()
	answers := map[string]json.RawMessage{}
	used := 0
	for _, s := range form.Sections {
		input := map[string]json.RawMessage{}
		for _, field := range s.Fields {
			if value, ok := values[field.ID]; ok {
				input[field.ID] = raw(t, value)
				used++
			}
		}
		cleaned, err := cleanSection(form, s, input, today)
		if err != nil {
			t.Fatalf("section %s: %v", s.ID, err)
		}
		maps.Copy(answers, cleaned)
	}
	if used != len(values) {
		t.Fatalf("%d values are not fields of the form", len(values)-used)
	}
	return answers
}

func everySection(form model.Form) map[string]bool {
	detailed := map[string]bool{}
	for _, s := range form.Sections {
		detailed[s.ID] = true
	}
	return detailed
}

// flatten lists the paths of a JSON value with their types, like
// "семья.родственники[].жив string", and the numbers with their values.
// Arrays merge their elements: the order and number of records do not matter.
func flatten(value any) (paths []string, numbers []string) {
	var data any
	_ = json.Unmarshal(canonicalJSON(value), &data)
	seen := map[string]bool{}
	var walk func(any, string)
	walk = func(v any, path string) {
		switch v := v.(type) {
		case map[string]any:
			seen[path+" object"] = true
			for _, key := range sortedKeys(v) {
				walk(v[key], strings.TrimPrefix(path+"."+key, "."))
			}
		case []any:
			seen[path+" array"] = true
			for _, item := range v {
				walk(item, path+"[]")
			}
		case string:
			seen[path+" string"] = true
		case float64:
			seen[path+" number"] = true
			numbers = append(numbers, path+" = "+strings.TrimSpace(string(canonicalJSON(v))))
		default:
			seen[path+" other"] = true
		}
	}
	walk(data, "")
	return slices.Sorted(maps.Keys(seen)), numbers
}

func without(list, other []string) []string {
	result := []string{}
	for _, item := range list {
		if !slices.Contains(other, item) {
			result = append(result, item)
		}
	}
	return result
}

// The made-up questionnaires of the model comparison, entered as a student
// would enter them. testdata/model_input holds copies of
// Вики/Автобиография/Сравнение моделей/анкеты.
var comparisonAnswers = map[string]answerSet{
	"01 Минимальная.json": {
		"birth_date":     "2008-03-15",
		"education_list": []record{{"edu_type": "school", "edu_profile": "none", "edu_year": "2025", "edu_medal": "no"}},
		"hse_campus":     "msk", "hse_level": "bakalavriat", "hse_faculty_msk": "fakultet_ekonomicheskikh_nauk", "hse_program": "Экономика", "hse_start": "2025-09", "hse_end": "2029-06",
		"work_has": "no", "served": "no", "lang_has": "no", "marital": "single", "children_has": "no",
		"upbringing": "both", "rel_has": "yes", "rel_list": []record{
			{"kind": "father", "known": "yes", "alive": "yes", "work": "works", "sphere": "promyshlennost_i_stroitelstvo", "position": "мастер участка", "foreign": "no", "crim": "no"},
			{"kind": "mother", "known": "yes", "alive": "yes", "work": "works", "sphere": "finansy_i_torgovlya", "position": "продавец-кассир", "foreign": "no", "crim": "no"}},
		"trips_has": "no", "crim_has": "no", "awards_has": "no",
		"sport_has": "no", "gto": "none", "creative_has": "no", "patriot_has": "no", "science_has": "no", "vol_has": "no",
		"hobbies": []string{"games"},
	},
	"02 Спорт и военно-патриотическое.json": {
		"birth_date":     "2008-01-20",
		"education_list": []record{{"edu_type": "cadet", "edu_profile": "cadet", "edu_year": "2025", "edu_medal": "no"}},
		"hse_campus":     "msk", "hse_level": "bakalavriat", "hse_faculty_msk": "moskovskiy_institut_elektroniki_i_matematiki_im_a_n_tikhonova", "hse_program": "Информационная безопасность", "hse_start": "2025-09", "hse_end": "2029-06",
		"work_has": "no", "served": "no",
		"lang_has": "yes", "lang_list": []record{{"language": "angliyskiy", "degree": "dict", "practice": []string{"school"}, "school_kind": "regular"}},
		"marital": "single", "children_has": "no",
		"upbringing": "both", "rel_has": "yes", "rel_list": []record{
			{"kind": "father", "known": "yes", "alive": "yes", "work": "works", "sphere": "voennaya_sluzhba", "position": "заместитель командира батальона", "foreign": "no", "crim": "no"},
			{"kind": "mother", "known": "yes", "alive": "yes", "work": "works", "sphere": "obrazovanie_i_nauka", "position": "учитель", "foreign": "no", "crim": "no"},
			{"kind": "brother", "known": "yes", "alive": "yes", "work": "student", "foreign": "no", "crim": "no"}},
		"trips_has": "no", "crim_has": "no", "awards_has": "no",
		"sport_has": "yes", "sport_list": []record{
			{"sport_kind": "sambo", "sport_now": "now", "sport_years": "y5", "sport_rank": "kms", "sport_level": "russia", "sport_ach": "призёр первенства федерального округа"},
			{"sport_kind": "legkaya_atletika", "sport_now": "before", "sport_years": "y13", "sport_rank": "r3", "sport_level": "city"}},
		"gto":          "gold",
		"creative_has": "no",
		"patriot_has":  "yes", "patriot_what": []string{"yunarmia", "zarnitsa", "cadet_class", "camps"}, "patriot_role": "leader", "patriot_term": "y3", "patriot_level": "region",
		"patriot_skills": []string{"fire", "drill", "medicine", "topography"}, "patriot_ach": "отделение заняло второе место на региональном этапе «Зарницы 2.0»",
		"science_has": "no",
		"vol_has":     "yes", "vol_dirs": []string{"patriotic"}, "vol_term": "y1", "vol_role": "member",
		"hobbies": []string{"hiking", "history"},
	},
	"03 Наука и языки.json": {
		"birth_date":     "2007-05-02",
		"education_list": []record{{"edu_type": "lyceum", "edu_profile": "fiziko_matematicheskiy", "edu_year": "2024", "edu_medal": "yes"}},
		"hse_campus":     "msk", "hse_level": "bakalavriat", "hse_faculty_msk": "fakultet_kompyuternykh_nauk", "hse_program": "Прикладная математика и информатика", "hse_start": "2024-09", "hse_end": "2028-06",
		"work_has": "no", "served": "no",
		"lang_has": "yes", "lang_list": []record{
			{"language": "angliyskiy", "degree": "fluent", "practice": []string{"school", "exam", "native"}, "school_kind": "advanced", "exam": "ielts", "exam_result": "7.5", "exam_year": "2025"},
			{"language": "nemetskiy", "degree": "talk", "practice": []string{"courses", "exam", "abroad"}, "courses_what": "языковая школа", "courses_time": "y13",
				"exam": "goethe", "exam_result": "B1", "exam_year": "2024", "abroad_country": "DE", "abroad_time": "m1"}},
		"marital": "single", "children_has": "no",
		"upbringing": "mother", "rel_has": "yes", "rel_list": []record{
			{"kind": "mother", "known": "yes", "alive": "yes", "work": "works", "sphere": "meditsina", "position": "врач-терапевт", "foreign": "no", "crim": "no"},
			{"kind": "father", "known": "no"}},
		"trips_has": "yes", "trips_list": []record{{"country": "DE", "year": "2024", "purpose": "study"}, {"country": "CN", "year": "2025", "purpose": "sport"}},
		"crim_has": "no", "awards_has": "no",
		"sport_has": "yes", "sport_list": []record{{"sport_kind": "shakhmaty", "sport_now": "before", "sport_years": "y35", "sport_rank": "r2", "sport_level": "city"}},
		"gto":          "none",
		"creative_has": "yes", "creative_list": []record{{"direction": "instrument", "detail": "фортепиано", "learning": "art_school_done", "level": "city"}},
		"patriot_has": "no",
		"science_has": "yes", "science_forms": []string{"olympiads", "conferences", "tech"}, "science_area": "math", "science_level": "russia", "science_result": "prize",
		"science_ach": "призёр заключительного этапа всероссийской олимпиады школьников по математике",
		"vol_has":     "no",
		"hobbies":     []string{"coding", "chess"},
	},
	"04 Служил, работал, женат.json": {
		"birth_date": "2003-02-11",
		"education_list": []record{{"edu_type": "college", "edu_profile": "inzhenerno_tekhnicheskiy", "edu_year": "2021",
			"edu_speciality": "техническое обслуживание и ремонт автомобильного транспорта", "edu_finished": "yes", "edu_medal": "no"}},
		"hse_campus": "nnov", "hse_level": "bakalavriat", "hse_faculty_nnov": "fakultet_menedzhmenta", "hse_program": "Логистика и управление цепями поставок", "hse_start": "2024-09", "hse_end": "2028-06",
		"work_has": "yes", "work_list": []record{
			{"sphere": "transport_i_svyaz", "position": "автослесарь", "from": "2020-06", "still": "no", "to": "2021-04", "leave_reason": "other"},
			{"sphere": "promyshlennost_i_stroitelstvo", "position": "слесарь-ремонтник", "from": "2022-07", "still": "no", "to": "2024-08", "leave_reason": "study"}},
		"served": "yes", "service_branch": "motostrelkovye_voyska", "service_from": "2021-05", "service_to": "2022-05", "service_position": "механик-водитель", "service_rank": "mladshiy_serzhant",
		"lang_has": "no",
		"marital":  "married", "children_has": "yes", "children_list": []record{{"child_birth": "2025-11-03"}},
		"upbringing": "both", "rel_has": "yes", "rel_list": []record{
			{"kind": "father", "known": "yes", "alive": "yes", "work": "pensioner", "foreign": "no", "crim": "no"},
			{"kind": "mother", "known": "yes", "alive": "yes", "work": "works", "sphere": "sfera_uslug", "position": "администратор", "foreign": "no", "crim": "no"}},
		"trips_has": "no", "crim_has": "no", "awards_has": "no",
		"sport_has": "yes", "sport_list": []record{{"sport_kind": "girevoy_sport", "sport_now": "now", "sport_years": "y35", "sport_rank": "r1", "sport_level": "region"}},
		"gto":          "silver",
		"creative_has": "no",
		"patriot_has":  "yes", "patriot_what": []string{"school_camps"}, "patriot_role": "member", "patriot_term": "once", "patriot_level": "school", "patriot_skills": []string{"fire", "drill"},
		"science_has": "no",
		"vol_has":     "yes", "vol_dirs": []string{"donor"}, "vol_term": "y13", "vol_role": "member",
		"hobbies": []string{"tech", "hunting"},
	},
	"05 Ловушка.json": {
		"birth_date":     "2007-01-30",
		"education_list": []record{{"edu_type": "school", "edu_profile": "none", "edu_year": "2024", "edu_medal": "no"}},
		"hse_campus":     "msk", "hse_level": "bakalavriat", "hse_faculty_msk": "fakultet_prava", "hse_program": "Юриспруденция", "hse_start": "2024-09", "hse_end": "2028-06",
		"work_has": "yes", "work_list": []record{{"sphere": "sfera_uslug", "position": "официант", "from": "2023-06", "still": "no", "to": "2024-08", "leave_reason": "study"}},
		"served":   "no",
		"lang_has": "yes", "lang_list": []record{{"language": "angliyskiy", "degree": "dict", "practice": []string{"school"}, "school_kind": "regular"}},
		"marital": "single", "children_has": "no",
		"upbringing": "orphanage", "rel_has": "yes", "rel_list": []record{
			{"kind": "mother", "known": "no"},
			{"kind": "brother", "known": "yes", "alive": "yes", "work": "works", "sphere": "sfera_uslug", "position": "курьер", "foreign": "no",
				"crim": "yes", "crim_year": "2022", "crim_what": "ст. 158 УК РФ, кража", "crim_measure": "исправительные работы", "crim_grave": "no"},
			{"kind": "other", "known": "yes", "alive": "yes", "work": "pensioner", "foreign": "yes", "foreign_country": "KZ", "crim": "no"}},
		"trips_has": "no",
		"crim_has":  "yes", "crim_year": "2023", "crim_what": "мелкое хулиганство", "crim_measure": "штраф",
		"awards_has": "no",
		"sport_has":  "yes", "sport_list": []record{{"sport_kind": "futbol", "sport_now": "now", "sport_years": "y35", "sport_rank": "none", "sport_level": "city"}},
		"gto":          "none",
		"creative_has": "no", "patriot_has": "no", "science_has": "no",
		"vol_has": "yes", "vol_dirs": []string{"social", "animals"}, "vol_term": "y13", "vol_role": "member",
		"hobbies": []string{"reading", "cooking"},
	},
	"06 Творчество и волонтёрство.json": {
		"birth_date":     "2008-06-01",
		"education_list": []record{{"edu_type": "gymnasium", "edu_profile": "gumanitarnyy", "edu_year": "2025", "edu_medal": "no"}},
		"hse_campus":     "spb", "hse_level": "bakalavriat", "hse_faculty_spb": "shkola_sotsialnykh_nauk", "hse_program": "Политология", "hse_start": "2025-09", "hse_end": "2029-06",
		"work_has": "no", "served": "no",
		"lang_has": "yes", "lang_list": []record{
			{"language": "angliyskiy", "degree": "talk", "practice": []string{"school", "university"}, "school_kind": "advanced"},
			{"language": "frantsuzskiy", "degree": "dict", "practice": []string{"courses"}, "courses_what": "репетитор", "courses_time": "y1"}},
		"marital": "single", "children_has": "no",
		"upbringing": "both", "rel_has": "yes", "rel_list": []record{
			{"kind": "father", "known": "yes", "alive": "yes", "work": "works", "sphere": "sobstvennoe_delo", "position": "индивидуальный предприниматель", "foreign": "no", "crim": "no"},
			{"kind": "mother", "known": "yes", "alive": "yes", "work": "works", "sphere": "kultura_sport_smi", "position": "журналист", "foreign": "no", "crim": "no"},
			{"kind": "sister", "known": "yes", "alive": "yes", "work": "student", "foreign": "no", "crim": "no"}},
		"trips_has": "yes", "trips_list": []record{{"country": "TR", "year": "2023", "purpose": "tourism"}},
		"crim_has": "no", "awards_has": "no",
		"sport_has": "yes", "sport_list": []record{{"sport_kind": "plavanie", "sport_now": "before", "sport_years": "y13", "sport_rank": "junior", "sport_level": "school"}},
		"gto":          "none",
		"creative_has": "yes", "creative_list": []record{
			{"direction": "vocal", "detail": "эстрадный вокал", "learning": "art_school_done", "level": "russia", "ach": "лауреат всероссийского вокального конкурса"},
			{"direction": "kvn", "learning": "self", "level": "school"}},
		"patriot_has": "yes", "patriot_what": []string{"first", "honor"}, "patriot_role": "member", "patriot_term": "y13", "patriot_level": "city", "patriot_skills": []string{"drill"},
		"science_has": "yes", "science_forms": []string{"conferences"}, "science_area": "humanities", "science_level": "city", "science_result": "member",
		"vol_has": "yes", "vol_dirs": []string{"events", "patriotic"}, "vol_term": "y3", "vol_role": "coordinator",
		"hobbies": []string{"reading", "history"},
	},
}

// The payload has the same fields as the questionnaires the models are
// compared on, and the levels counted by the scale are the ones written in them.
// Texts are not compared: the questionnaires write them in lower case.
func TestPayloadHasTheStructureOfTheComparisonQuestionnaires(t *testing.T) {
	form := autobiography(t)
	files, err := filepath.Glob(filepath.Join("testdata", "model_input", "*.json"))
	if err != nil || len(files) != len(comparisonAnswers) {
		t.Fatalf("questionnaires %v (%v), answers for %d", files, err, len(comparisonAnswers))
	}
	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			values, ok := comparisonAnswers[name]
			if !ok {
				t.Fatalf("no answers for %s", name)
			}
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var want map[string]any
			if err = json.Unmarshal(data, &want); err != nil {
				t.Fatal(err)
			}
			payload := autobiographyPayload(form, answersOf(t, form, values), today)
			var got map[string]any
			if err = json.Unmarshal(canonicalJSON(payload), &got); err != nil {
				t.Fatal(err)
			}
			wantPaths, wantNumbers := flatten(want)
			gotPaths, gotNumbers := flatten(got)
			if missing, extra := without(wantPaths, gotPaths), without(gotPaths, wantPaths); len(missing)+len(extra) > 0 {
				t.Errorf("fields differ\nmissing: %q\nextra: %q", missing, extra)
			}
			if !slices.Equal(gotNumbers, wantNumbers) {
				t.Errorf("numbers %q, want %q", gotNumbers, wantNumbers)
			}
			if !reflect.DeepEqual(got["уровни"], want["уровни"]) {
				t.Errorf("levels %v, want %v", got["уровни"], want["уровни"])
			}
		})
	}
}

// The copies in testdata stay equal to the questionnaires in the wiki. The wiki
// is outside the repository: without it next to the project the test is skipped.
func TestComparisonQuestionnairesAreCurrent(t *testing.T) {
	originals, _ := filepath.Glob(filepath.Join("..", "..", "..", "Вики", "Автобиография", "Сравнение моделей", "анкеты", "*.json"))
	if len(originals) == 0 {
		t.Skip("the wiki is not next to the project")
	}
	copies, _ := filepath.Glob(filepath.Join("testdata", "model_input", "*.json"))
	if len(copies) != len(originals) {
		t.Fatalf("%d copies of %d questionnaires", len(copies), len(originals))
	}
	for _, original := range originals {
		want, _ := os.ReadFile(original)
		got, err := os.ReadFile(filepath.Join("testdata", "model_input", filepath.Base(original)))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("testdata/model_input/%s differs from the wiki: %v", filepath.Base(original), err)
		}
	}
}

// personalAnswers is an autobiography with names, dates, addresses and
// organisations in every field that holds them.
func personalAnswers() answerSet {
	return answerSet{
		"name_changed": "yes", "prev_names": "Сидоров Пётр Ильич", "name_change_reason": "own",
		"birth_date": "2004-12-24", "birth_country": "BY", "birth_place": "Гомель",
		"nationality": "other", "nationality_other": "Сету",
		"education_list": []record{{"edu_type": "college", "edu_profile": "inzhenerno_tekhnicheskiy", "edu_region": "44", "school_city": "Кострома",
			"school_name": "Костромской энергетический техникум имени Ф. В. Чижова", "edu_year": "2021", "edu_speciality": "электроснабжение", "edu_finished": "yes", "edu_medal": "no"}},
		"hse_campus": "perm", "hse_level": "bakalavriat", "hse_faculty_perm": "fakultet_sotsialno_ekonomicheskikh_i_kompyuternykh_nauk",
		"hse_program": "Программная инженерия", "hse_start": "2024-09", "hse_end": "2028-06",
		"work_has": "yes", "work_list": []record{{"org": "ООО «Ромашка-Север»", "sphere": "it", "position": "монтажник слаботочных систем", "from": "2023-07", "still": "yes"}},
		"served": "yes", "service_branch": "motostrelkovye_voyska", "service_from": "2021-11", "service_to": "2022-11",
		"service_place": "войсковая часть 54321, Кострома", "service_position": "стрелок", "service_vus": "100182", "service_rank": "ryadovoy",
		"vk_region": "44", "vk_district": "Заволжский район", "vk_same": "no", "vk_prev_region": "76", "vk_prev_district": "Рыбинск",
		"lang_has": "no",
		"marital":  "married", "spouse_last": "Кузнецова", "spouse_maiden": "Белова", "spouse_first": "Ульяна", "spouse_middle": "Тимуровна",
		"spouse_birth": "2005-02-17", "spouse_birthplace": "Галич", "spouse_job": "АО «Газпром межрегионгаз Кострома», оператор", "spouse_with_me": "no",
		"spouse_address": "ул. Сусанина, д. 41, кв. 8", "cert_number": "I-ГО № 654321", "cert_issued_by": "Отдел ЗАГС Заволжского района", "cert_date": "2025-06-14",
		"children_has": "yes", "children_list": []record{{"child_name": "Ярослава", "child_birth": "2026-02-09"}},
		"upbringing": "both", "rel_has": "yes", "rel_list": []record{
			{"kind": "father", "known": "yes", "name": "Кузнецов Тимофей Андреевич", "birth": "1971-08-30", "alive": "no", "death_year": "2016"},
			{"kind": "mother", "known": "yes", "name": "Кузнецова Елена Викторовна", "birth": "1974-03-05", "alive": "yes", "work": "works", "org": "МБОУ «Гимназия № 15»",
				"sphere": "obrazovanie_i_nauka", "position": "учитель", "with_me": "no", "address": "пр. Мира, д. 102, кв. 3", "foreign": "no",
				"crim": "yes", "crim_year": "2011", "crim_what": "ст. 264 УК РФ", "crim_measure": "лишение права управления на три года", "crim_grave": "no"}},
		"trips_has": "no",
		"crim_has":  "yes", "crim_year": "2018", "crim_what": "хулиганство у клуба «Орбита»", "crim_measure": "обязательные работы 120 часов",
		"awards_has": "no",
		"reg_region": "44", "reg_address": "г. Кострома, ул. Советская, д. 7, кв. 12", "lives_at_reg": "no", "live_kind": "dorm",
		"live_address": "Москва, ул. Кирпичная, д. 33, общежитие № 4",
		"sport_has":    "no", "gto": "none", "creative_has": "no", "patriot_has": "no", "science_has": "no", "vol_has": "no",
		"hobbies": []string{"reading", "history"},
	}
}

// personalValues collects what fields without the llm mark hold: typed values
// and the names from the lists of regions, countries and nationalities.
func personalValues(form model.Form, answers map[string]json.RawMessage) []string {
	values := []string{}
	var walk func([]model.FormField, formAnswers)
	walk = func(fields []model.FormField, a formAnswers) {
		for _, field := range fields {
			value := a.text(field.ID)
			switch {
			case field.Type == "group":
				for _, r := range a.records(field.ID) {
					walk(field.Fields, r)
				}
			case field.LLM || value == "":
			case field.Type != "choice":
				values = append(values, value)
			case field.Ref != "" && value != "other":
				for _, option := range form.Options(field) {
					if option.ID == value {
						values = append(values, option.Text)
					}
				}
			}
		}
	}
	for _, s := range form.Sections {
		walk(s.Fields, formAnswers(answers))
	}
	return values
}

func TestPayloadLeavesOutNamesDatesAddressesAndOrganisations(t *testing.T) {
	form := autobiography(t)
	answers := answersOf(t, form, personalAnswers())
	personal := personalValues(form, answers)
	if len(personal) < 35 {
		t.Fatalf("only %d personal values reached the stored answers: %q", len(personal), personal)
	}
	payload := autobiographyPayload(form, answers, today)
	data := string(canonicalJSON(payload))
	for _, value := range personal {
		if strings.Contains(data, value) {
			t.Errorf("%q reached the model input", value)
		}
	}

	var got map[string]any
	_ = json.Unmarshal([]byte(data), &got)
	family := got["семья"].(map[string]any)["родственники"].([]any)
	checks := []struct {
		name      string
		got, want any
	}{
		{"age", got["возраст"], 21.0},
		{"marital status", got["семейное_положение"], map[string]any{"статус": "Женат", "детей": 1.0}},
		{"own criminal record", got["уголовная_ответственность"], "привлекался"},
		{"dead father", family[0], map[string]any{"кем_приходится": "Отец", "сведения": "Есть сведения", "жив": "Нет"}},
		{"mother's record", family[1].(map[string]any)["статья"], "ст. 264 УК РФ"},
		{"current job", got["работа_до_вуза"], []any{map[string]any{"сфера": "ИТ", "должность": "монтажник слаботочных систем", "период": "с 2023-07 по настоящее время"}}},
		{"service", got["служба"], map[string]any{"служил": "Да", "род_войск": "Мотострелковые войска", "период": "с 2021-11 по 2022-11", "должность": "стрелок", "звание": "Рядовой"}},
	}
	for _, c := range checks {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, c.got, c.want)
		}
	}
}

// markPersonal replaces the typed values and dates of the fields without the
// llm mark, in group records too, with markers that must not reach the model.
func markPersonal(form model.Form, answers map[string]json.RawMessage) int {
	count := 0
	var mark func([]model.FormField, map[string]json.RawMessage)
	mark = func(fields []model.FormField, values map[string]json.RawMessage) {
		for _, field := range fields {
			value, ok := values[field.ID]
			switch {
			case !ok:
			case field.Type == "group":
				var records []map[string]json.RawMessage
				_ = json.Unmarshal(value, &records)
				for _, r := range records {
					mark(field.Fields, r)
				}
				values[field.ID], _ = json.Marshal(records)
			case !field.LLM && field.Type != "choice" && field.Type != "multi":
				values[field.ID], _ = json.Marshal("ЛИЧНОЕ " + field.ID)
				count++
			}
		}
	}
	for _, s := range form.Sections {
		mark(s.Fields, answers)
	}
	return count
}

func TestPayloadOfTheSampleLeavesOutUnmarkedFields(t *testing.T) {
	form := autobiography(t)
	answers, err := sampleAnswers(form, everySection(form), today)
	if err != nil {
		t.Fatal(err)
	}
	if marked := markPersonal(form, answers); marked < 15 {
		t.Fatalf("only %d unmarked fields in the sample", marked)
	}
	data := string(canonicalJSON(autobiographyPayload(form, answers, today)))
	if leaked := regexp.MustCompile(`ЛИЧНОЕ \w+`).FindAllString(data, -1); len(leaked) > 0 {
		t.Fatalf("unmarked fields reached the model input: %q", leaked)
	}
}

// comparisonAllowed is a copy of ALLOWED from
// Вики/Автобиография/Сравнение моделей/compare.py: the fields the comparison
// lets through to the model. Change it together with compare.py.
var comparisonAllowed = []string{
	"возраст",
	"учёба_до_вуза[].тип", "учёба_до_вуза[].профиль", "учёба_до_вуза[].год_окончания",
	"учёба_до_вуза[].специальность", "учёба_до_вуза[].окончил", "учёба_до_вуза[].медаль",
	"вшэ.кампус", "вшэ.уровень", "вшэ.факультет", "вшэ.направление", "вшэ.поступление", "вшэ.окончание",
	"работа_до_вуза[].сфера", "работа_до_вуза[].должность", "работа_до_вуза[].период", "работа_до_вуза[].причина_ухода",
	"служба.служил", "служба.род_войск", "служба.период", "служба.должность", "служба.звание",
	"языки[].язык", "языки[].степень",
	"языки[].практика[].вид", "языки[].практика[].программа", "языки[].практика[].курсы",
	"языки[].практика[].длительность", "языки[].практика[].экзамен", "языки[].практика[].результат",
	"языки[].практика[].год", "языки[].практика[].страна", "языки[].практика[].срок", "языки[].практика[].работа",
	"семейное_положение.статус", "семейное_положение.детей",
	"семья.кто_воспитывал",
	"семья.родственники[].кем_приходится", "семья.родственники[].сведения", "семья.родственники[].жив",
	"семья.родственники[].занятость", "семья.родственники[].сфера", "семья.родственники[].должность",
	"семья.родственники[].иностранное_гражданство", "семья.родственники[].страна_гражданства",
	"семья.родственники[].судимость", "семья.родственники[].статья", "семья.родственники[].тяжкое",
	"поездки_за_границу[].страна", "поездки_за_границу[].год", "поездки_за_границу[].цель",
	"уголовная_ответственность",
	"награды[].название", "награды[].год",
	"спорт.занятия[].вид", "спорт.занятия[].сейчас", "спорт.занятия[].стаж", "спорт.занятия[].разряд",
	"спорт.занятия[].соревнования", "спорт.занятия[].достижения", "спорт.гто",
	"творчество[].направление", "творчество[].уточнение", "творчество[].обучение", "творчество[].конкурсы",
	"творчество[].достижения",
	"военно_патриотическое.участвовал", "военно_патриотическое.формы", "военно_патриотическое.роль",
	"военно_патриотическое.длительность", "военно_патриотическое.уровень", "военно_патриотическое.навыки",
	"военно_патриотическое.достижения",
	"наука.занимался", "наука.формы", "наука.область", "наука.уровень", "наука.результат", "наука.достижения",
	"волонтёрство.занимался", "волонтёрство.направления", "волонтёрство.длительность", "волонтёрство.роль",
	"увлечения",
	"уровни.спорт", "уровни.музыка и творчество", "уровни.военно-патриотическая деятельность", "уровни.наука",
	"уровни.волонтёрство", "уровни.иностранные языки",
}

// Every field the builder writes passes the list of compare.py, the language
// practice with all its kinds included.
func TestPayloadFieldsAreAllowedByTheComparison(t *testing.T) {
	allowed := map[string]bool{"": true}
	for _, leaf := range comparisonAllowed {
		allowed[leaf+"[]"] = true // a value may be a list of strings
		for path := leaf; path != ""; {
			allowed[path] = true
			cut := max(strings.LastIndex(path, "."), strings.LastIndex(path, "[]"))
			if cut < 0 {
				break
			}
			if strings.HasSuffix(path, "[]") {
				path = strings.TrimSuffix(path, "[]")
			} else {
				path = path[:cut]
			}
		}
	}
	form := autobiography(t)
	every := personalAnswers()
	every["lang_has"], every["lang_list"] = "yes", []record{{
		"language": "angliyskiy", "degree": "talk",
		"practice":    []string{"school", "university", "courses", "exam", "abroad", "env", "work", "native", "second"},
		"school_kind": "advanced", "courses_what": "курсы при вузе", "courses_time": "y13",
		"exam": "ielts", "exam_result": "6.5", "exam_year": "2024", "abroad_country": "GB", "abroad_time": "m6",
		"env_country": "US", "env_years": "2", "work_use": "переписка с поставщиками"}}
	every["trips_has"], every["trips_list"] = "yes", []record{{"country": "TR", "year": "2023", "purpose": "tourism"}}
	every["awards_has"], every["awards_list"] = "yes", []record{{"award_name": "Медаль «За отвагу»", "award_year": "2023"}}
	sets := []answerSet{every}
	for _, name := range sortedKeys(comparisonAnswers) {
		sets = append(sets, comparisonAnswers[name])
	}
	for _, values := range sets {
		paths, _ := flatten(autobiographyPayload(form, answersOf(t, form, values), today))
		for _, path := range paths {
			path = path[:strings.LastIndex(path, " ")]
			if !allowed[path] {
				t.Errorf("compare.py does not allow %q", path)
			}
		}
	}
}

// The marks in autobiography.json decide what the model gets: every marked
// field reaches the builder, and the builder asks for no other field.
func TestPayloadUsesExactlyTheMarkedFields(t *testing.T) {
	form := autobiography(t)
	sample, err := sampleAnswers(form, everySection(form), today)
	if err != nil {
		t.Fatal(err)
	}
	asked := map[string]bool{}
	for _, answers := range []map[string]json.RawMessage{sample, answersOf(t, form, personalAnswers())} {
		_, seen := buildPayload(form, answers, today)
		maps.Copy(asked, seen)
	}
	var walk func(string, []model.FormField)
	walk = func(path string, fields []model.FormField) {
		for _, field := range fields {
			switch _, used := asked[path+"."+field.ID]; {
			case field.Type == "group":
				walk(path+"."+field.ID, field.Fields)
			case field.LLM && !used:
				t.Errorf("%s.%s is marked llm but does not reach the model", path, field.ID)
			}
		}
	}
	for _, s := range form.Sections {
		walk(s.ID, s.Fields)
	}
	for _, path := range sortedKeys(asked) {
		if !asked[path] {
			t.Errorf("the builder asks for %s, which is not marked llm", path)
		}
	}
}

func TestEqualAnswersGiveEqualFingerprints(t *testing.T) {
	form := autobiography(t)
	fingerprint := func(values answerSet) string {
		return payloadFingerprint(autobiographyPayload(form, answersOf(t, form, values), today))
	}
	first := fingerprint(personalAnswers())
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(first) {
		t.Fatalf("fingerprint %q", first)
	}
	for range 20 {
		if again := fingerprint(personalAnswers()); again != first {
			t.Fatalf("the same answers gave %s and %s", first, again)
		}
	}

	var stored model.FormSubmission
	if err := json.Unmarshal(raw(t, model.FormSubmission{Answers: answersOf(t, form, personalAnswers())}), &stored); err != nil {
		t.Fatal(err)
	}
	if read := payloadFingerprint(autobiographyPayload(form, stored.Answers, today)); read != first {
		t.Errorf("answers read back from storage gave another fingerprint")
	}

	// Another student: other names, places and organisations, the same age.
	other := personalAnswers()
	other["prev_names"], other["birth_date"], other["birth_place"] = "Орлов Глеб Сергеевич", "2004-11-30", "Минск"
	other["education_list"].([]record)[0]["school_name"] = "Техникум № 3"
	other["work_list"].([]record)[0]["org"] = "ИП Орлов"
	other["rel_list"].([]record)[1]["address"] = "ул. Ленина, д. 1"
	other["spouse_first"], other["reg_address"], other["crim_what"] = "Дарья", "г. Кострома, ул. Ткачей, д. 2", "другое"
	if fingerprint(other) != first {
		t.Error("answers equal without personal data gave another fingerprint")
	}

	clicked := personalAnswers()
	clicked["hobbies"] = []string{"history", "reading"}
	if fingerprint(clicked) != first {
		t.Error("the order of clicks changed the fingerprint")
	}

	changed := personalAnswers()
	changed["work_list"].([]record)[0]["position"] = "электрик"
	if fingerprint(changed) == first {
		t.Error("another position gave the same fingerprint")
	}
}

func TestAgeIsCountedOnTheMoscowDate(t *testing.T) {
	cases := []struct {
		birth string
		asOf  time.Time
		age   int
	}{
		{"2006-10-10", time.Date(2026, 10, 9, 20, 0, 0, 0, time.UTC), 19},
		{"2006-10-10", time.Date(2026, 10, 9, 21, 30, 0, 0, time.UTC), 20}, // 00:30 on the 10th in Moscow
		{"2008-02-29", time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC), 17},
		{"2008-02-29", time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), 18},
	}
	for _, c := range cases {
		if age := ageOn(c.birth, c.asOf); age != c.age {
			t.Errorf("born %s, on %s: %v, want %d", c.birth, c.asOf, age, c.age)
		}
	}
	if age := ageOn("", today); age != nil {
		t.Errorf("no birth date: %v", age)
	}
}

func TestSubmissionPayloadCountsTheAgeOnTheDayOfSubmission(t *testing.T) {
	form := autobiography(t)
	values := personalAnswers()
	values["birth_date"] = "2004-10-10"
	submission := model.FormSubmission{FormID: autobiographyForm, Answers: answersOf(t, form, values)}
	if _, _, err := submissionPayload(form, submission); err == nil {
		t.Fatal("a draft gave a model input")
	}
	submission.Completed, submission.CompletedAt = true, "2026-10-09T21:30:00Z"
	payload, fingerprint, err := submissionPayload(form, submission)
	if err != nil {
		t.Fatal(err)
	}
	if payload["возраст"] != 22 {
		t.Errorf("age %v, want 22 on the Moscow date of submission", payload["возраст"])
	}
	if fingerprint != payloadFingerprint(payload) {
		t.Errorf("fingerprint %s of another input", fingerprint)
	}
}
