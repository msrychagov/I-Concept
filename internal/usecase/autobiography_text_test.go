package usecase

import (
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

// sokolov is the example of Вики/Автобиография/Шаблон автобиографии.md: the
// comparison questionnaire «04 Служил, работал, женат» with made-up names,
// dates and addresses.
var sokolov = answerSet{
	"name_changed": "no", "birth_date": "2001-05-14", "birth_country": "RU", "birth_region": "52", "birth_place": "г. Дзержинск", "nationality": "russkie",
	"education_list": []record{{"edu_type": "college", "edu_profile": "inzhenerno_tekhnicheskiy", "edu_region": "52", "school_city": "г. Дзержинск",
		"school_name": "ГБПОУ «Дзержинский технический колледж»", "edu_year": "2021", "edu_speciality": "Техническое обслуживание и ремонт автомобильного транспорта", "edu_finished": "yes", "edu_medal": "no"}},
	"hse_campus": "nnov", "hse_level": "bakalavriat", "hse_faculty_nnov": "fakultet_menedzhmenta", "hse_program": "Логистика и управление цепями поставок", "hse_start": "2024-09", "hse_end": "2028-06",
	"work_has": "yes", "work_list": []record{
		{"org": "АО «Дзержинский машиностроительный завод»", "sphere": "promyshlennost_i_stroitelstvo", "position": "слесарь-ремонтник", "from": "2022-07", "still": "no", "to": "2024-08", "leave_reason": "study"},
		{"org": "ООО «Автосервис-НН»", "sphere": "transport_i_svyaz", "position": "автослесарь", "from": "2020-06", "still": "no", "to": "2021-04", "leave_reason": "other"}},
	"served": "yes", "service_branch": "motostrelkovye_voyska", "service_from": "2021-05", "service_to": "2022-05", "service_place": "войсковая часть 12345, Нижегородская область",
	"service_position": "механик-водитель", "service_vus": "123182", "service_rank": "mladshiy_serzhant",
	"vk_region": "52", "vk_district": "города Дзержинска", "vk_same": "yes",
	"lang_has": "no",
	"marital":  "married", "spouse_last": "Соколова", "spouse_maiden": "Белова", "spouse_first": "Ольга", "spouse_middle": "Игоревна", "spouse_birth": "2002-02-03",
	"spouse_birthplace": "г. Дзержинск", "spouse_job": "ООО «Аптека Плюс», фармацевт", "spouse_with_me": "yes",
	"cert_number": "II-ТН № 654321", "cert_issued_by": "отдел ЗАГС г. Дзержинска", "cert_date": "2023-08-12",
	"children_has": "yes", "children_list": []record{{"child_name": "Мария", "child_birth": "2024-06-05"}},
	"upbringing": "both", "rel_has": "yes", "rel_list": []record{
		{"kind": "mother", "known": "yes", "name": "Соколова Татьяна Ивановна", "birth": "1975-10-08", "alive": "yes", "work": "works", "org": "ООО «Уют»", "sphere": "sfera_uslug", "position": "администратор",
			"with_me": "no", "address": "Нижегородская обл., г. Дзержинск, ул. Гагарина, д. 5, кв. 12", "foreign": "no", "crim": "no"},
		{"kind": "father", "known": "yes", "name": "Соколов Виктор Петрович", "birth": "1972-01-20", "alive": "yes", "work": "pensioner",
			"with_me": "no", "address": "Нижегородская обл., г. Дзержинск, ул. Гагарина, д. 5, кв. 12", "foreign": "no", "crim": "no"}},
	"trips_has": "no", "crim_has": "no", "awards_has": "no",
	"reg_region": "52", "reg_address": "г. Дзержинск, ул. Гагарина, д. 5, кв. 12", "lives_at_reg": "no", "live_kind": "rent", "live_address": "г. Нижний Новгород, ул. Белинского, д. 10, кв. 4",
	"sport_has": "yes", "sport_list": []record{{"sport_kind": "girevoy_sport", "sport_now": "now", "sport_years": "y35", "sport_rank": "r1", "sport_level": "region"}},
	"gto": "silver", "creative_has": "no",
	"patriot_has": "yes", "patriot_what": []string{"school_camps"}, "patriot_role": "member", "patriot_term": "once", "patriot_level": "school", "patriot_skills": []string{"fire", "drill"},
	"science_has": "no", "vol_has": "yes", "vol_dirs": []string{"donor"}, "vol_term": "y13", "vol_role": "member",
	"hobbies": []string{"tech", "hunting"},
}

func printed(t *testing.T, values answerSet, name string) AutobiographyText {
	t.Helper()
	form := autobiography(t)
	submission := model.FormSubmission{FormID: autobiographyForm, Completed: true, CompletedAt: "2024-09-14T22:30:00Z", Answers: answersOf(t, form, values)}
	return autobiographyText(form, model.Account{Name: name}, submission)
}

func TestAutobiographyTextFollowsTheTemplate(t *testing.T) {
	got := printed(t, sokolov, " Соколов  Андрей Викторович ")
	want := []string{
		"Я, Соколов Андрей Викторович, родился 14 мая 2001 года; место рождения — г. Дзержинск, Нижегородская область. По национальности — русский.",
		"В 2021 году окончил ГБПОУ «Дзержинский технический колледж» (г. Дзержинск, Нижегородская область), специальность «Техническое обслуживание и ремонт автомобильного транспорта».",
		"До поступления в университет работал: с июня 2020 по апрель 2021 года — ООО «Автосервис-НН», автослесарь; с июля 2022 по август 2024 года — АО «Дзержинский машиностроительный завод», слесарь-ремонтник, уволился в связи с поступлением в вуз.",
		"С мая 2021 по май 2022 года проходил военную службу в Вооружённых силах: Мотострелковые войска; место службы — войсковая часть 12345, Нижегородская область; должность — механик-водитель; ВУС — 123182; воинское звание — младший сержант.",
		"В сентябре 2024 года поступил в НИУ ВШЭ — Нижний Новгород на программу бакалавриата «Логистика и управление цепями поставок» (Факультет менеджмента). Окончание обучения — июнь 2028 года.",
		"Иностранными языками не владею.",
		"Семейное положение — женат. Супруга — Соколова (девичья фамилия — Белова) Ольга Игоревна, 3 февраля 2002 года рождения, место рождения — г. Дзержинск; место работы и должность — ООО «Аптека Плюс», фармацевт; проживает со мной. Свидетельство о браке II-ТН № 654321; кем выдано — отдел ЗАГС г. Дзержинска; дата выдачи — 12 августа 2023 года. Дети: Мария, 5 июня 2024 года рождения.",
		"Близкие родственники:",
		"Отец — Соколов Виктор Петрович, 20 января 1972 года рождения, на пенсии; проживает по адресу: Нижегородская обл., г. Дзержинск, ул. Гагарина, д. 5, кв. 12.",
		"Мать — Соколова Татьяна Ивановна, 8 октября 1975 года рождения, место работы и должность — ООО «Уют», администратор; проживает по адресу: Нижегородская обл., г. Дзержинск, ул. Гагарина, д. 5, кв. 12.",
		"За границей не был. Близких родственников, имеющих иностранное гражданство или подданство, не имею. Ни я, ни мои близкие родственники к уголовной ответственности не привлекались.",
		"Государственных наград не имею.",
		"Постоянно зарегистрирован по адресу: Нижегородская область, г. Дзержинск, ул. Гагарина, д. 5, кв. 12. Проживаю в съёмном жилье по адресу: г. Нижний Новгород, ул. Белинского, д. 10, кв. 4.",
		"На воинском учёте состою в военном комиссариате города Дзержинска (Нижегородская область). До поступления в университет состоял на учёте там же.",
	}
	if len(got.Paragraphs) != len(want) {
		t.Fatalf("%d paragraphs, want %d:\n%s", len(got.Paragraphs), len(want), strings.Join(got.Paragraphs, "\n"))
	}
	for n := range want {
		if got.Paragraphs[n] != want[n] {
			t.Errorf("paragraph %d:\n got %s\nwant %s", n+1, got.Paragraphs[n], want[n])
		}
	}
	if got.Date != "15 сентября 2024 г." || got.Surname != "Соколов" || got.Student != "Соколов Андрей Викторович" {
		t.Errorf("footer %q, %q, %q", got.Date, got.Surname, got.Student)
	}
	// Only what the paper form asks: nothing of the sport, the hobbies or the
	// fields added over the form.
	all := strings.Join(got.Paragraphs, " ")
	for _, word := range []string{"гирев", "ГТО", "донор", "охот", "Инженерно-технический", "Промышленность", "Оба родителя", "обоими"} {
		if strings.Contains(all, word) {
			t.Errorf("%q is not on the paper form", word)
		}
	}
}

// The other branches: a changed name, birth abroad, a nationality typed by
// hand, an unfinished college, a school given by its number, languages,
// relatives dead, unknown or abroad, criminal records and awards.
func TestAutobiographyTextVariants(t *testing.T) {
	values := answerSet{
		"name_changed": "yes", "prev_names": "Петров Иван Сергеевич", "name_change_reason": "adoption",
		"birth_date": "2005-12-01", "birth_country": "UZ", "birth_place": "г. Ташкент", "nationality": "other", "nationality_other": "каракалпак",
		"education_list": []record{
			{"edu_type": "college", "edu_profile": "none", "edu_region": "77", "school_city": "г. Москва", "school_name": "ГБПОУ «Политехнический техникум № 2»", "edu_year": "2023", "edu_speciality": "сварочное производство", "edu_finished": "no", "edu_medal": "no"},
			{"edu_type": "school", "edu_profile": "none", "edu_region": "abroad", "school_city": "Ташкент", "school_name": "15", "edu_year": "2021", "edu_medal": "yes"}},
		"hse_campus": "msk", "hse_level": "magistratura", "hse_faculty_msk": "other", "hse_program": "Право", "hse_start": "2025-09", "hse_end": "2027-06",
		"work_has": "yes", "work_list": []record{{"org": "ООО «Ромашка»", "sphere": "sfera_uslug", "position": "официант", "from": "2024-06", "still": "yes"}},
		"served": "no", "vk_region": "77", "vk_district": "Военный комиссариат Басманного района", "vk_same": "no", "vk_prev_region": "50", "vk_prev_district": "г. Химки",
		"lang_has": "yes", "lang_list": []record{
			{"language": "angliyskiy", "degree": "talk", "practice": []string{"school"}, "school_kind": "regular"},
			{"language": "other", "language_other": "узбекский", "degree": "fluent", "practice": []string{"second"}}},
		"marital": "single", "children_has": "no",
		"upbringing": "orphanage", "rel_has": "yes", "rel_list": []record{
			{"kind": "other", "known": "yes", "name": "Петрова Анна Ивановна", "birth": "1950-04-02", "alive": "no", "death_year": "2019"},
			{"kind": "brother", "known": "yes", "name": "Петров Олег Сергеевич", "birth": "2000-07-07", "alive": "yes", "work": "none", "with_me": "yes", "foreign": "yes", "foreign_country": "KZ",
				"crim": "yes", "crim_year": "2020", "crim_what": "кража, ст. 158 УК РФ", "crim_measure": "условный срок", "crim_grave": "no"},
			{"kind": "father", "known": "no"}},
		"trips_has": "yes", "trips_list": []record{{"country": "KZ", "year": "2023", "purpose": "relatives"}, {"country": "TR", "year": "2019", "purpose": "other"}},
		"crim_has": "yes", "crim_year": "2022", "crim_what": "мелкое хулиганство", "crim_measure": "штраф",
		"awards_has": "yes", "awards_list": []record{{"award_name": "медаль «За спасение погибавших»", "award_year": "2024"}},
		"reg_region": "77", "reg_address": "г. Москва, ул. Мира, д. 1", "lives_at_reg": "yes",
		"sport_has": "no", "gto": "none", "creative_has": "no", "patriot_has": "no", "science_has": "no", "vol_has": "no",
	}
	got := printed(t, values, "Иванов Иван Сергеевич")
	all := strings.Join(got.Paragraphs, "\n")
	for _, want := range []string{
		"Я, Иванов Иван Сергеевич (прежние ФИО — Петров Иван Сергеевич, изменены в связи с усыновлением (установлением отцовства)), родился 1 декабря 2005 года; место рождения — г. Ташкент, Узбекистан. По национальности — каракалпак.",
		"В 2021 году окончил школу № 15 (Ташкент). До 2023 года учился в техникуме — ГБПОУ «Политехнический техникум № 2» (г. Москва), специальность «сварочное производство»; обучение не окончил.",
		"До поступления в университет работал: с июня 2024 года по настоящее время — ООО «Ромашка», официант.",
		"В Вооружённых силах не служил.",
		"В сентябре 2025 года поступил в НИУ ВШЭ на программу магистратуры «Право». Окончание обучения — июнь 2027 года.",
		"Владею иностранными языками: английский — читаю и могу объясняться; узбекский — свободно.",
		"Семейное положение — холост. Детей не имею.",
		"Брат — Петров Олег Сергеевич, 7 июля 2000 года рождения, не работает; проживает со мной.",
		"Сведений об отце не имею.",
		"Близкий родственник — Петрова Анна Ивановна, 2 апреля 1950 года рождения, год смерти — 2019.",
		"За границей был: Турция — 2019 год; Казахстан — 2023 год, к родственникам. Из близких родственников иностранное гражданство имеет: брат Петров Олег Сергеевич — Казахстан. В 2022 году привлекался к уголовной ответственности: мелкое хулиганство; мера наказания — штраф. Из близких родственников к уголовной ответственности привлекались: брат Петров Олег Сергеевич — в 2020 году, кража, ст. 158 УК РФ, мера наказания — условный срок.",
		"Награждён государственной наградой: медаль «За спасение погибавших» — 2024 год.",
		"Постоянно зарегистрирован и проживаю по адресу: г. Москва, ул. Мира, д. 1.",
		"На воинском учёте состою в военном комиссариате Басманного района (Москва). До поступления в университет состоял на учёте в военном комиссариате г. Химки (Московская область).",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("missing:\n%s\nin:\n%s", want, all)
		}
	}
	// Brother before the father? No: the order of the form, then «other».
	if strings.Index(all, "Сведений об отце") > strings.Index(all, "Брат —") {
		t.Errorf("relatives are not in the order of the form:\n%s", all)
	}
	for _, word := range []string{"Детский дом", "детском доме", "тяжк"} {
		if strings.Contains(all, word) {
			t.Errorf("%q is not on the paper form", word)
		}
	}
}

func TestAutobiographyTextWithoutRelatives(t *testing.T) {
	values := answerSet{}
	for key, value := range sokolov {
		values[key] = value
	}
	values["rel_has"] = "no"
	delete(values, "rel_list")
	got := strings.Join(printed(t, values, "Соколов Андрей Викторович").Paragraphs, "\n")
	if !strings.Contains(got, "\nБлизких родственников не имею.\n") || strings.Contains(got, "Близкие родственники:") {
		t.Errorf("no relatives:\n%s", got)
	}
}

func TestEveryNationalityHasAMaleForm(t *testing.T) {
	form := autobiography(t)
	for _, option := range form.Refs["nationalities"] {
		if option.ID != "other" && maleNationality[option.ID] == "" {
			t.Errorf("no male form of %s (%s)", option.ID, option.Text)
		}
	}
}
