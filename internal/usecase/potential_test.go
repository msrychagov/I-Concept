package usecase

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

type answerSet map[string]any

func potentialOf(t *testing.T, values answerSet) Potential {
	t.Helper()
	answers := map[string]json.RawMessage{}
	for key, value := range values {
		answers[key] = raw(t, value)
	}
	return autobiographyPotential(autobiography(t), answers)
}

func areaOf(t *testing.T, p Potential, id string) PotentialArea {
	t.Helper()
	for _, area := range p.Areas {
		if area.ID == id {
			return area
		}
	}
	t.Fatalf("no area %s in %v", id, p.Areas)
	return PotentialArea{}
}

type record = map[string]any

// The examples of the scale (Вики/Автобиография/Шкала потенциала.md) and the
// edge cases its rules name explicitly.
func TestPotentialLevelsFollowTheScale(t *testing.T) {
	cases := []struct {
		name, area, level, reason string
		answers                   answerSet
	}{
		{"sambo master candidate, earlier athletics", "sport", "high", "Самбо: разряд — КМС, уровень соревнований — Россия", answerSet{"sport_has": "yes", "sport_list": []record{
			{"sport_kind": "sambo", "sport_now": "now", "sport_years": "y5", "sport_rank": "kms", "sport_level": "russia"},
			{"sport_kind": "legkaya_atletika", "sport_now": "before", "sport_years": "y13", "sport_rank": "r3", "sport_level": "none"}}}},
		{"football, city competitions", "sport", "medium", "Футбол: уровень соревнований — Город или район, стаж — 3–5 лет", answerSet{"sport_has": "yes", "sport_list": []record{
			{"sport_kind": "futbol", "sport_now": "now", "sport_years": "y35", "sport_rank": "none", "sport_level": "city"}}}},
		{"sport given up earlier is not lower", "sport", "high", "Самбо: разряд — МС", answerSet{"sport_has": "yes", "sport_list": []record{
			{"sport_kind": "sambo", "sport_now": "before", "sport_years": "y1", "sport_rank": "ms", "sport_level": "none"}}}},
		{"sport without rank and competitions", "sport", "initial", "Футбол: стаж — 1–3 года, без разряда и соревнований", answerSet{"sport_has": "yes", "sport_list": []record{
			{"sport_kind": "futbol", "sport_now": "now", "sport_years": "y13", "sport_rank": "none", "sport_level": "none"}}}},
		{"gold GTO without sport", "sport", "medium", "Знак ГТО — Золото", answerSet{"sport_has": "no", "gto": "gold"}},
		{"bronze GTO without sport", "sport", "initial", "Знак ГТО — Бронза", answerSet{"sport_has": "no", "gto": "bronze"}},
		{"no sport and no GTO", "sport", "none", "", answerSet{"sport_has": "no", "gto": "none"}},
		{"piano, art school, city contest", "creative", "medium", "Музыкальный инструмент (фортепиано): обучение — Школа искусств — окончил(а), уровень конкурсов — Город или район", answerSet{"creative_has": "yes", "creative_list": []record{
			{"direction": "instrument", "detail": "фортепиано", "learning": "art_school_done", "level": "city"}}}},
		{"creative college", "creative", "high", "Танцы: обучение — Колледж", answerSet{"creative_has": "yes", "creative_list": []record{
			{"direction": "dance", "learning": "college", "level": "none"}}}},
		{"self-taught without contests", "creative", "initial", "Вокал: обучение — Самостоятельно, без конкурсов", answerSet{"creative_has": "yes", "creative_list": []record{
			{"direction": "vocal", "learning": "self", "level": "none"}}}},
		{"yunarmia squad leader for years", "patriotic", "high", "Участие: роль — Командир отделения или отряда, как долго — Больше 3 лет, уровень — Регион", answerSet{"patriot_has": "yes", "patriot_what": []string{"yunarmia"}, "patriot_role": "leader", "patriot_term": "y3", "patriot_level": "region"}},
		{"one-off camps but served", "patriotic", "high", "Служил(а) в Вооружённых силах: уровень не ниже высокого", answerSet{"patriot_has": "yes", "patriot_what": []string{"school_camps"}, "patriot_role": "member", "patriot_term": "once", "patriot_level": "school", "served": "yes"}},
		{"no participation, cadet school", "patriotic", "medium", "Кадетская школа или корпус: уровень не ниже среднего", answerSet{"patriot_has": "no", "education_list": []record{{"edu_type": "cadet", "edu_profile": "none"}}}},
		{"suvorov school", "patriotic", "high", "Суворовское, нахимовское, президентское кадетское училище: уровень не ниже высокого", answerSet{"patriot_has": "no", "education_list": []record{{"edu_type": "school", "edu_profile": "none"}, {"edu_type": "suvorov", "edu_profile": "cadet"}}}},
		{"cadet profile in a school", "patriotic", "medium", "Профиль «Кадетский, оборонно-спортивный»: уровень не ниже среднего", answerSet{"patriot_has": "no", "education_list": []record{{"edu_type": "lyceum", "edu_profile": "cadet"}}}},
		{"raised for four forms", "patriotic", "medium", "Уровень поднят на ступень: отмечено форм участия — 4", answerSet{"patriot_has": "yes", "patriot_what": []string{"yunarmia", "zarnitsa", "first", "search"}, "patriot_role": "member", "patriot_term": "y1", "patriot_level": "city"}},
		{"raised for four skills", "patriotic", "high", "Уровень поднят на ступень: отмечено навыков — 4", answerSet{"patriot_has": "yes", "patriot_what": []string{"club"}, "patriot_skills": []string{"fire", "drill", "medicine", "drones"}, "patriot_role": "member", "patriot_term": "y13", "patriot_level": "school"}},
		{"not raised for a one-off event", "patriotic", "initial", "Участие: роль — Участник, как долго — Разово, уровень — Город или район", answerSet{"patriot_has": "yes", "patriot_what": []string{"yunarmia", "zarnitsa", "first", "search"}, "patriot_role": "member", "patriot_term": "once", "patriot_level": "city"}},
		{"one-off organiser", "patriotic", "medium", "Участие: роль — Организатор, инструктор, как долго — Разово, уровень — Школа", answerSet{"patriot_has": "yes", "patriot_what": []string{"zarnitsa"}, "patriot_role": "organizer", "patriot_term": "once", "patriot_level": "school"}},
		{"regional member for a year or more", "patriotic", "high", "Участие: роль — Участник, как долго — 1–3 года, уровень — Россия", answerSet{"patriot_has": "yes", "patriot_what": []string{"yunarmia"}, "patriot_role": "member", "patriot_term": "y13", "patriot_level": "russia"}},
		{"no patriotic experience", "patriotic", "none", "", answerSet{"patriot_has": "no", "served": "no", "education_list": []record{{"edu_type": "school", "edu_profile": "none"}}}},
		{"all-russian olympiad prize", "science", "high", "Результат — Призёр, уровень — Россия", answerSet{"science_has": "yes", "science_forms": []string{"olympiads"}, "science_level": "russia", "science_result": "prize"}},
		{"publications", "science", "high", "Есть публикации", answerSet{"science_has": "yes", "science_forms": []string{"publications", "society"}, "science_level": "school", "science_result": "member"}},
		{"university lab", "science", "medium", "Лаборатория или научный проект в вузе", answerSet{"science_has": "yes", "science_forms": []string{"lab"}, "science_level": "school", "science_result": "member"}},
		{"regional participant", "science", "medium", "Результат — Участник, уровень — Регион", answerSet{"science_has": "yes", "science_forms": []string{"conferences"}, "science_level": "region", "science_result": "member"}},
		{"school participant", "science", "initial", "Результат — Участник, уровень — Школа или вуз", answerSet{"science_has": "yes", "science_forms": []string{"society", "olympiads", "tech"}, "science_level": "school", "science_result": "member"}},
		{"donor for years", "volunteering", "medium", "Роль — Участник, как долго — 1–3 года", answerSet{"vol_has": "yes", "vol_dirs": []string{"donor"}, "vol_term": "y13", "vol_role": "member"}},
		{"volunteer for more than three years", "volunteering", "high", "Роль — Участник, как долго — Больше 3 лет", answerSet{"vol_has": "yes", "vol_dirs": []string{"social"}, "vol_term": "y3", "vol_role": "member"}},
		{"one-off coordinator", "volunteering", "medium", "Роль — Координатор, организатор, как долго — Разово", answerSet{"vol_has": "yes", "vol_dirs": []string{"events"}, "vol_term": "once", "vol_role": "coordinator"}},
		{"one-off volunteer", "volunteering", "initial", "Роль — Участник, как долго — Разово", answerSet{"vol_has": "yes", "vol_dirs": []string{"events", "eco", "animals"}, "vol_term": "once", "vol_role": "member"}},
		{"fluent english with ielts, german", "languages", "high", "Английский: свободно", answerSet{"lang_has": "yes", "lang_list": []record{
			{"language": "angliyskiy", "degree": "fluent", "practice": []string{"exam"}, "exam": "ielts", "exam_result": "7.5", "exam_year": "2024"},
			{"language": "nemetskiy", "degree": "talk", "practice": []string{"school"}, "school_kind": "regular"}}}},
		{"two languages to talk", "languages", "high", "Языков на уровне «Читаю и могу объясняться» — 2: Английский, Немецкий", answerSet{"lang_has": "yes", "lang_list": []record{
			{"language": "angliyskiy", "degree": "talk", "practice": []string{"school"}},
			{"language": "nemetskiy", "degree": "talk", "practice": []string{"courses"}}}}},
		{"second native language", "languages", "high", "Мой язык: второй родной язык", answerSet{"lang_has": "yes", "lang_list": []record{
			{"language": "other", "language_other": "Мой язык", "degree": "dict", "practice": []string{"second"}}}}},
		{"certificate with a dictionary", "languages", "medium", "Английский: сертификат TOEFL", answerSet{"lang_has": "yes", "lang_list": []record{
			{"language": "angliyskiy", "degree": "dict", "practice": []string{"exam"}, "exam": "toefl"}}}},
		{"dictionary only", "languages", "initial", "Английский: читаю и перевожу со словарём", answerSet{"lang_has": "yes", "lang_list": []record{
			{"language": "angliyskiy", "degree": "dict", "practice": []string{"school"}}}}},
		{"no languages", "languages", "none", "", answerSet{"lang_has": "no"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			area := areaOf(t, potentialOf(t, c.answers), c.area)
			if area.Level != c.level {
				t.Fatalf("level %s, want %s; reasons %q", area.Level, c.level, area.Reasons)
			}
			if c.reason == "" && len(area.Reasons) != 0 || c.reason != "" && !slices.Contains(area.Reasons, c.reason) {
				t.Fatalf("reasons %q, want %q", area.Reasons, c.reason)
			}
		})
	}
}

func TestPotentialUsesTheBestRecordAndNamesTheHighAreas(t *testing.T) {
	p := potentialOf(t, answerSet{
		"sport_has": "yes", "sport_list": []record{
			{"sport_kind": "futbol", "sport_now": "now", "sport_years": "y1", "sport_rank": "none", "sport_level": "none"},
			{"sport_kind": "sambo", "sport_now": "before", "sport_years": "y35", "sport_rank": "r1", "sport_level": "city"}},
		"gto":     "gold",
		"vol_has": "yes", "vol_dirs": []string{"donor"}, "vol_term": "y3", "vol_role": "coordinator",
		"awards_has": "yes", "awards_list": []record{{"award_name": "Медаль «За отвагу»", "award_year": "2023"}},
	})
	sport := areaOf(t, p, "sport")
	if sport.Level != "high" || sport.LevelText != "высокий" || len(sport.Reasons) != 1 || !strings.HasPrefix(sport.Reasons[0], "Самбо:") {
		t.Fatalf("sport: %+v", sport)
	}
	if len(p.Areas) != 6 || p.Areas[0].ID != "sport" || p.Areas[5].ID != "languages" {
		t.Fatalf("areas: %+v", p.Areas)
	}
	if !slices.Equal(p.High, []string{"Спорт", "Волонтёрство"}) {
		t.Fatalf("high: %v", p.High)
	}
	if len(p.Awards) != 1 || p.Awards[0].Name != "Медаль «За отвагу»" || p.Awards[0].Year != "2023" {
		t.Fatalf("awards: %v", p.Awards)
	}
	for _, area := range p.Areas {
		if area.ID != "sport" && area.ID != "volunteering" && area.Level != "none" {
			t.Fatalf("awards changed a level: %+v", area)
		}
	}
}

func TestPotentialOfTheSampleAutobiography(t *testing.T) {
	form := autobiography(t)
	answers, err := sampleAnswers(form, map[string]bool{"sport": true, "creative": true, "patriotic": true, "science": true, "volunteering": true, "languages": true}, today)
	if err != nil {
		t.Fatal(err)
	}
	p := autobiographyPotential(form, answers)
	for _, area := range p.Areas {
		if area.Level == "none" || len(area.Reasons) == 0 {
			t.Fatalf("every area was answered with yes: %+v", area)
		}
		for _, reason := range area.Reasons {
			if strings.Contains(reason, "—  ") || strings.HasSuffix(reason, "— ") {
				t.Fatalf("empty value in %q", reason)
			}
		}
	}
}
