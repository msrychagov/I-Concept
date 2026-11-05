package usecase

import (
	"encoding/json"
	"math"
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/promptbank"
)

// samplesV5 are the conclusions «как должно быть» of the questionnaires 1–4 of
// the comparison (Вики/Для руководителя/Образцы — автобиография.md), approved
// by the user 5 November 2026: the check must let them through.
var samplesV5 = map[string]map[string]any{
	"01 Минимальная.json": {
		"paragraphs": []any{
			"Из занятий указал увлечение компьютерными играми и киберспортом.",
			"Быстро реагирует, умеет долго удерживать внимание и принимать решения в быстро меняющейся обстановке, привык действовать в команде по общему замыслу. Для будущего военного специалиста это хорошие задатки: быстрота решений и слаженность действий.",
		},
		"areas": []any{}, "psychologist": []any{}, "training": []any{}, "consider": []any{},
	},
	"02 Спорт и военно-патриотическое.json": {
		"paragraphs": []any{
			"Окончил кадетскую школу. Больше трёх лет участвовал в «Юнармии», «Зарнице 2.0» и военно-спортивных сборах, где выполнял обязанности командира отделения; имеет навыки огневой и строевой подготовки, топографии и ориентирования, первой помощи и тактической медицины, его отделение заняло второе место на региональном этапе «Зарницы 2.0». Больше пяти лет занимается самбо — кандидат в мастера спорта, призёр первенства федерального округа; раньше занимался лёгкой атлетикой (третий разряд), имеет золотой знак ГТО. Меньше года участвовал в патриотических волонтёрских акциях, увлекается походами и историей.",
			"Интерес к военному делу у него давний и устойчивый, к истории страны и памяти её защитников относится с уважением. Основы военной подготовки ему уже знакомы: привык к распорядку и требованиям, умеет командовать небольшой группой сверстников и отвечать за общий результат. Физически крепок и вынослив, привычен к большим нагрузкам и к борьбе, где важно сохранять самообладание.",
		},
		"areas": []any{
			map[string]any{"area": "спорт", "text": "Самбо и лёгкая атлетика развили силу, выносливость, быстроту и навыки самообороны; к соревнованиям и большим нагрузкам привычен."},
			map[string]any{"area": "военно-патриотическая деятельность", "text": "Патриотизм у него деятельный: готов брать на себя ответственность за товарищей и общее дело, уважает воинские традиции и память ветеранов."},
			map[string]any{"area": "волонтёрство", "text": "Способен откликнуться на общее дело и помогать людям старшего поколения."},
			map[string]any{"area": "иностранные языки", "text": "Английский — на уровне чтения и перевода со словарём, по школьной программе."},
		},
		"psychologist": []any{"Отец — военнослужащий, заместитель командира батальона. В беседе уточнить, сам ли студент выбрал военную подготовку и чего он от неё ждёт."},
		"training":     []any{}, "consider": []any{},
	},
	"03 Наука и языки.json": {
		"paragraphs": []any{
			"Призёр заключительного этапа всероссийской олимпиады школьников по математике, участвовал в научных конференциях и конкурсах проектов, занимался в технических кружках. Свободно владеет английским (IELTS 7,5), немецким — на уровне B1, стажировался в Германии. Окончил школу искусств по классу фортепиано и выступал на городских конкурсах; три–пять лет занимался шахматами, имеет второй разряд. Увлекается программированием.",
			"Как будущий военный специалист он силён прежде всего мышлением: умеет разобраться в сложной задаче, просчитать варианты на несколько ходов вперёд и довести решение до точного результата. Усидчив, привык к долгой самостоятельной работе, сохраняет выдержку в напряжённой борьбе и уверенно держится перед публикой. Свободно читает иноязычные источники и может объясняться с иностранцами.",
		},
		"areas": []any{
			map[string]any{"area": "спорт", "text": "Шахматы развили выдержку, сосредоточенность и умение просчитывать ходы наперёд."},
			map[string]any{"area": "музыка и творчество", "text": "Способен к долгой кропотливой работе и к выступлению перед публикой."},
			map[string]any{"area": "наука", "text": "Способен к сложной аналитической работе, точным расчётам и самостоятельному поиску решений, умеет представить свою работу."},
			map[string]any{"area": "иностранные языки", "text": "Английский — свободно: IELTS 7,5 соответствует уровню C1, то есть свободному владению языком в учёбе и работе; немецкий — B1 по Goethe-Zertifikat: читает и объясняется на бытовые и учебные темы."},
		},
		"psychologist": []any{"Воспитывался матерью, сведений об отце не имеет. В беседе уточнить, кто из взрослых мужчин был для него примером; обратить внимание на то, как он входит в мужской коллектив и принимает требования командиров."},
		"training":     []any{}, "consider": []any{},
	},
	"04 Служил, работал, женат.json": {
		"paragraphs": []any{
			"Год служил в мотострелковых войсках механиком-водителем, младший сержант; прошёл учебные военные сборы с огневой и строевой подготовкой. Три–пять лет занимается гиревым спортом, имеет первый разряд и выступает на соревнованиях региона, сдал нормы ГТО на серебряный знак. Один–три года сдаёт кровь как донор. Увлекается автомобильной и мототехникой, охотой и рыбалкой.",
			"Армейский порядок знает изнутри: привык к воинской дисциплине и распорядку, умеет обращаться с боевой техникой. Хорошо разбирается в машинах и привычен к полевым условиям. Физически силён и вынослив, умеет долго выдерживать нагрузку.",
		},
		"areas": []any{
			map[string]any{"area": "спорт", "text": "Гиревой спорт развил силу, силовую выносливость и упорство; к соревнованиям привычен."},
			map[string]any{"area": "военно-патриотическая деятельность", "text": "Воинский долг уже исполнил и к службе относится как к знакомому, понятному делу."},
			map[string]any{"area": "волонтёрство", "text": "Способен на постоянное, а не разовое доброе дело — несколько лет сдаёт кровь."},
		},
		"psychologist": []any{},
		"training": []any{
			"Служил механиком-водителем, младший сержант: знает распорядок и требования воинской части, устройство и обслуживание боевой машины — на практических занятиях может опираться на собственный опыт службы.",
			"Работал автослесарем и слесарем-ремонтником: умеет обслуживать и ремонтировать машины, работать с инструментом — хорошая основа для изучения устройства военной техники.",
			"Женат, растит ребёнка: привык заботиться о близких и отвечать за них.",
		},
		"consider": []any{},
	},
}

// comparisonPayload is the input of a questionnaire of the comparison.
func comparisonPayload(t *testing.T, name string) *object {
	t.Helper()
	form := autobiography(t)
	return autobiographyPayload(form, answersOf(t, form, comparisonAnswers[name]), today)
}

// sampleV5 is a deep copy of a sample, to change it in a test.
func sampleV5(name string) map[string]any {
	data, _ := json.Marshal(samplesV5[name])
	answer := map[string]any{}
	_ = json.Unmarshal(data, &answer)
	return answer
}

func TestV5SamplesPassTheCheck(t *testing.T) {
	for name := range samplesV5 {
		if problems := checkAutobiographyAnswerV5(sampleV5(name), comparisonPayload(t, name)); len(problems) != 0 {
			t.Errorf("%s: %q", name, problems)
		}
	}
}

func TestV5ExamplesAreThoseOfTheInstruction(t *testing.T) {
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	prompt, ok := promptbank.Find(prompts, autobiographyForm, "v5")
	if !ok {
		t.Fatal("autobiography v5 is not embedded")
	}
	if n := strings.Count(prompt.Instruction, "Так: «"); n != len(autobiographyExamplesV5) {
		t.Fatalf("%d examples in the instruction, %d in the program", n, len(autobiographyExamplesV5))
	}
	for _, example := range autobiographyExamplesV5 {
		if !strings.Contains(prompt.Instruction, "Так: «"+example) {
			t.Errorf("not in the instruction: %q", example)
		}
		if len(wordsOfV5(example)) >= exampleRunV5 && !copiesExampleV5(example) {
			t.Errorf("not found as a copy: %q", example)
		}
	}
	if copiesExampleV5("Быстро ориентируется в игре и умеет действовать с командой") {
		t.Error("five words of an example are taken for a copy")
	}
}

// broken changes a sample in one place; the check must name the rule.
func TestV5FindsTheBrokenRules(t *testing.T) {
	paragraph := func(n int, text string) func(map[string]any) {
		return func(a map[string]any) { a["paragraphs"].([]any)[n] = text }
	}
	area := func(n int, text string) func(map[string]any) {
		return func(a map[string]any) { a["areas"].([]any)[n].(map[string]any)["text"] = text }
	}
	set := func(field string, points ...any) func(map[string]any) {
		return func(a map[string]any) { a[field] = points }
	}
	for name, tc := range map[string]struct {
		sample  string
		mutate  func(map[string]any)
		problem string
	}{
		"three paragraphs":    {"02 Спорт и военно-патриотическое.json", func(a map[string]any) { a["paragraphs"] = append(a["paragraphs"].([]any), "Ещё.") }, "абзацев 3, а нужно ровно два"},
		"one paragraph":       {"01 Минимальная.json", func(a map[string]any) { a["paragraphs"] = a["paragraphs"].([]any)[:1] }, "абзацев 1"},
		"by experience":       {"02 Спорт и военно-патриотическое.json", paragraph(1, "По опыту в самбо у него выносливость."), "машинный оборот («по опыту»)"},
		"this shows":          {"02 Спорт и военно-патриотическое.json", paragraph(1, "Он командовал отделением, это показывает ответственность."), "машинный оборот («это показывает»)"},
		"useful in vuc":       {"03 Наука и языки.json", paragraph(1, "Умение считать пригодится на занятиях ВУЦ."), "где пригодится («пригодится»)"},
		"vuc alone":           {"03 Наука и языки.json", area(1, "Сможет выступать на смотрах ВУЦ."), "где пригодится («вуц»)"},
		"family in paragraph": {"03 Наука и языки.json", paragraph(1, "Рос с матерью и привык к самостоятельности."), "семья, брак, гражданство или судимость («матерью»)"},
		"work in paragraph":   {"04 Служил, работал, женат.json", paragraph(1, "Как слесарь-ремонтник хорошо знает технику."), "работа до вуза не фактом"},
		"work skill":          {"04 Служил, работал, женат.json", paragraph(0, "До вуза работал автослесарем, освоил ремонт машин."), "работа до вуза не фактом"},
		"work in area":        {"04 Служил, работал, женат.json", area(0, "Работа слесарем-ремонтником развила силу."), "работа до вуза — о ней пиши в training"},
		"appointment":         {"02 Спорт и военно-патриотическое.json", paragraph(1, "Его можно назначить командиром отделения."), "запрет 4"},
		"copy of an example":  {"01 Минимальная.json", paragraph(1, "Быстро ориентируется в игре, умеет действовать заодно с командой."), "повторяет пример из инструкции"},
		"absence":             {"02 Спорт и военно-патриотическое.json", paragraph(0, "Окончил кадетскую школу без медали."), "об отсутствии («без медали»)"},
		"empty area named":    {"02 Спорт и военно-патриотическое.json", paragraph(0, "Пел в вокальном ансамбле."), "направление без опыта («музыка и творчество»)"},
		"missing area": {"04 Служил, работал, женат.json", func(a map[string]any) { a["areas"] = a["areas"].([]any)[:2] },
			"пиши строки только для направлений: спорт, военно-патриотическая деятельность, волонтёрство"},
		"area without experience": {"01 Минимальная.json", set("areas", map[string]any{"area": "спорт", "text": "Опыта в спорте не указал."}), "ни одного: оставь список пустым"},
		"areas of texts":          {"01 Минимальная.json", set("areas", "спорт"), "areas не список записей"},
		"level in area":           {"02 Спорт и военно-патриотическое.json", area(0, "Высокий уровень: развиты сила и выносливость."), "уровень потенциала («высокий уровень»)"},
		"potential in area":       {"02 Спорт и военно-патриотическое.json", area(3, "Потенциал в языках начальный: читает со словарём."), "уровень потенциала («потенциал»)"},
		"three sentences":         {"04 Служил, работал, женат.json", area(0, "Силён. Вынослив. Упорен."), "предложений 3"},
		"language not named":      {"03 Наука и языки.json", area(3, "Английский — свободно, IELTS 7,5; объясняется на бытовые темы."), "не назван язык «немецкий»"},
		"result not named":        {"03 Наука и языки.json", area(3, "Английский — свободно; немецкий — B1, объясняется на бытовые темы."), "не назван результат экзамена «7.5»"},
		"psychologist not needed": {"04 Служил, работал, женат.json", set("psychologist", "Воспитывался обоими родителями."), "обстоятельств для психолога нет"},
		"psychologist missing":    {"03 Наука и языки.json", set("psychologist"), "psychologist: пусто"},
		"no question":             {"03 Наука и языки.json", set("psychologist", "Воспитывался матерью, сведений об отце не имеет."), "нет того, что уточнить"},
		"label":                   {"03 Наука и языки.json", set("psychologist", "Рос в неполной семье. В беседе уточнить, как он привык к требованиям."), "ярлык или прогноз («неполн»)"},
		"prognosis":               {"03 Наука и языки.json", set("psychologist", "Воспитывался матерью, поэтому в беседе уточнить, есть ли трудности с командирами."), "ярлык или прогноз («поэтому»)"},
		"not family":              {"03 Наука и языки.json", set("psychologist", "В беседе уточнить, чего он ждёт от военной подготовки."), "это не о семье и воспитании"},
		"four points":             {"05 Ловушка.json", func(a map[string]any) {}, "psychologist: пунктов больше трёх"},
		"training not needed":     {"02 Спорт и военно-патриотическое.json", set("training", "Холост: может уделять учёбе всё время."), "training: студент не служил"},
		"training missing":        {"04 Служил, работал, женат.json", set("training"), "training: пусто"},
		"training load":           {"04 Служил, работал, женат.json", set("training", "Женат, есть ребёнок — учитывать при планировании нагрузки."), "планирование, допуск или рекомендация"},
		"training upbringing":     {"04 Служил, работал, женат.json", set("training", "Служил в армии; воспитывался обоими родителями."), "воспитание («воспитывался»)"},
		"training not about":      {"04 Служил, работал, женат.json", set("training", "Любит охоту и рыбалку."), "в training — только служба"},
		"consider not needed":     {"04 Служил, работал, женат.json", set("consider", "Служил в Вооружённых силах."), "судимостей и иностранного гражданства нет"},
		"consider conclusion":     {"05 Ловушка.json", func(a map[string]any) {}, "вывод вместо факта"},
		// Checks added after the run of 6 November 2026.
		"wrong term":       {"05 Ловушка.json", paragraph(0, "Год играл в футбол, участвовал в волонтёрских акциях."), "срок не тот («год» — в анкете «футбол»: 3–5 лет)"},
		"term of a group":  {"02 Спорт и военно-патриотическое.json", paragraph(0, "Год состоял в «Юнармии»."), "срок не тот («год» — в анкете «военно-патриотическое»: больше 3 лет)"},
		"not worked":       {"01 Минимальная.json", paragraph(0, "Увлекается киберспортом. До поступления не работал."), "об отсутствии («не работал»)"},
		"without profile":  {"01 Минимальная.json", paragraph(0, "Увлекается киберспортом, окончил школу без профиля."), "об отсутствии («без профиля»)"},
		"knows little":     {"03 Наука и языки.json", paragraph(1, "Военное дело пока знает мало."), "чего ещё нет или что поможет потом («знает мало»)"},
		"will help":        {"03 Наука и языки.json", paragraph(1, "Привычка к дисциплине поможет освоить строевую подготовку."), "чего ещё нет или что поможет потом («поможет»)"},
		"new field":        {"03 Наука и языки.json", paragraph(1, "Военное дело для него — новая область."), "«новая область»"},
		"connector":        {"03 Наука и языки.json", area(3, "Английский — свободно, IELTS 7,5; немецкий — B1, что позволяет читать литературу."), "машинный оборот («что позволяет»)"},
		"shows readiness":  {"03 Наука и языки.json", paragraph(1, "Языки показывают готовность к новой среде."), "машинный оборот («показывают готовность»)"},
		"trip quality":     {"03 Наука и языки.json", paragraph(1, "Опыт поездок развил собранность."), "качества из поездок («поездок»)"},
		"training joined":  {"04 Служил, работал, женат.json", set("training", "Служил механиком-водителем, работал автослесарем: знает технику.", "Женат, растит ребёнка: привык заботиться о близких."), "в одном пункте сразу служба и работа"},
		"training too few": {"04 Служил, работал, женат.json", set("training", "Служил механиком-водителем: знает технику."), "пунктов 1, а обстоятельств 3"},
	} {
		t.Run(name, func(t *testing.T) {
			answer := trapAnswerV5(name)
			if tc.sample != "05 Ловушка.json" {
				answer = sampleV5(tc.sample)
			}
			tc.mutate(answer)
			problems := checkAutobiographyAnswerV5(answer, comparisonPayload(t, tc.sample))
			if !slices.ContainsFunc(problems, func(p string) bool { return strings.Contains(p, tc.problem) }) {
				t.Fatalf("no %q in %q", tc.problem, problems)
			}
		})
	}
}

// trapAnswerV5 is a good answer to the questionnaire 05 (brought up in an
// orphanage, nothing known of the mother, a brother convicted, a relative of
// a foreign citizenship, his own criminal record, a waiter before the
// university), broken by the case of the test.
func trapAnswerV5(broken string) map[string]any {
	answer := map[string]any{
		"paragraphs": []any{
			"Занимался футболом, участвовал в волонтёрских акциях. Читает со словарём по-английски, увлекается чтением и кулинарией.",
			"Привык действовать в команде и помогать людям.",
		},
		"areas": []any{
			map[string]any{"area": "спорт", "text": "Футбол развил выносливость и командную игру."},
			map[string]any{"area": "волонтёрство", "text": "Способен помогать людям."},
			map[string]any{"area": "иностранные языки", "text": "Английский — читает и переводит со словарём."},
		},
		"psychologist": []any{"Воспитывался в детском доме, сведений о матери не имеет. В беседе уточнить, кто поддерживал его в детстве."},
		"training":     []any{"Работал официантом: привык к быстрому темпу и общению с людьми."},
		"consider": []any{
			"Привлекался к уголовной ответственности; подробности — в ответах студента.",
			"У брата есть судимость.",
			"У близкого родственника иностранное гражданство.",
		},
	}
	switch broken {
	case "four points":
		answer["psychologist"] = []any{"Детский дом. В беседе уточнить.", "Мать. В беседе уточнить.", "Брат. В беседе уточнить.", "Семья. В беседе уточнить."}
	case "consider conclusion":
		answer["consider"].([]any)[1] = "У брата есть судимость — учитывать при допуске."
	}
	return answer
}

func TestV5TrapAnswerPasses(t *testing.T) {
	payload := comparisonPayload(t, "05 Ловушка.json")
	if problems := checkAutobiographyAnswerV5(trapAnswerV5(""), payload); len(problems) != 0 {
		t.Fatalf("%q", problems)
	}
	for name, change := range map[string]func(map[string]any){
		"no conviction":  func(a map[string]any) { a["consider"] = slices.Delete(a["consider"].([]any), 1, 2) },
		"no citizenship": func(a map[string]any) { a["consider"] = a["consider"].([]any)[:2] },
		"no own record":  func(a map[string]any) { a["consider"] = a["consider"].([]any)[1:] },
		"family fact": func(a map[string]any) {
			a["consider"] = append(a["consider"].([]any), "Воспитывался в детском доме.")
		},
	} {
		answer := trapAnswerV5("")
		change(answer)
		if problems := checkAutobiographyAnswerV5(answer, payload); len(problems) == 0 {
			t.Errorf("%s: passes", name)
		}
	}
}

// Decided by the user after the run of 5 November 2026: the work before the
// university may be a fact of the first paragraph, the points for the
// psychologist and the military training may repeat their examples, and
// «нагрузка» is no conclusion there.
func TestV5DecisionsAfterTheRun(t *testing.T) {
	payload := comparisonPayload(t, "04 Служил, работал, женат.json")
	answer := sampleV5("04 Служил, работал, женат.json")
	answer["paragraphs"].([]any)[0] = answer["paragraphs"].([]any)[0].(string) + " До вуза работал автослесарем и слесарем-ремонтником."
	answer["training"].([]any)[2] = "Женат, растит двоих детей: привык отвечать не только за себя, но и за близких."
	answer["training"] = append(answer["training"].([]any)[1:], "Служил механиком-водителем: привык к физическим нагрузкам и полевым условиям.")
	if problems := checkAutobiographyAnswerV5(answer, payload); len(problems) != 0 {
		t.Fatalf("%q", problems)
	}
	trap := trapAnswerV5("")
	trap["psychologist"] = []any{"Воспитывался в детском доме. В беседе уточнить, на кого из взрослых он привык опираться, и обратить внимание, как он входит в новый коллектив."}
	if problems := checkAutobiographyAnswerV5(trap, comparisonPayload(t, "05 Ловушка.json")); len(problems) != 0 {
		t.Fatalf("%q", problems)
	}
	// Nor the line of the languages (decided 6 November 2026).
	science := sampleV5("03 Наука и языки.json")
	science["areas"].([]any)[3].(map[string]any)["text"] = "Английский — свободно, IELTS 7,5; немецкий — B1: свободно читает и объясняется на учебные и бытовые темы."
	if problems := checkAutobiographyAnswerV5(science, comparisonPayload(t, "03 Наука и языки.json")); len(problems) != 0 {
		t.Fatalf("languages: %q", problems)
	}
	// The paragraphs still may not copy an example.
	answer["paragraphs"].([]any)[1] = "Быстро ориентируется в игре, умеет действовать заодно с командой."
	if problems := checkAutobiographyAnswerV5(answer, payload); len(problems) != 1 || !strings.Contains(problems[0], "повторяет пример") {
		t.Fatalf("%q", problems)
	}
}

func TestV5WorkOfTheSameNameAsTheServiceIsAllowed(t *testing.T) {
	payload := levelsPayload(map[string]string{"военно-патриотическая деятельность": "высокий"})
	service := newObject()
	service.set("служил", "Да")
	service.set("должность", "механик-водитель")
	payload.set("служба", service)
	work := newObject()
	work.set("должность", "механик")
	payload.set("работа_до_вуза", []any{work})
	answer := map[string]any{
		"paragraphs":   []any{"Служил механиком-водителем.", "Привык к порядку и технике."},
		"areas":        []any{map[string]any{"area": "военно-патриотическая деятельность", "text": "К службе относится серьёзно."}},
		"psychologist": []any{}, "training": []any{"Служил механиком-водителем: привык к порядку.", "Работал механиком: знает устройство машин."}, "consider": []any{},
	}
	if problems := checkAutobiographyAnswerV5(answer, payload); len(problems) != 0 {
		t.Fatalf("%q", problems)
	}
}

func TestV5TermsOfTheQuestionnaire(t *testing.T) {
	for text, want := range map[string]termV5{
		"До года": {0, 1}, "1–3 года": {1, 3}, "3–5 лет": {3, 5}, "Больше 5 лет": {5, math.Inf(1)}, "Больше 3 лет": {3, math.Inf(1)}, "Разово": {0, 0.5},
	} {
		if got, ok := answerTermV5(text); !ok || got != want {
			t.Errorf("%q: %v", text, got)
		}
	}
	for text, want := range map[string]termV5{
		"Год играл в футбол":      {1, 1},
		"три года занимался":      {3, 3},
		"Три–пять лет занимается": {3, 5},
		"больше пяти лет":         {5.5, math.Inf(1)},
		"Меньше года участвовал":  {0, 1},
		"не менее двух лет":       {2, math.Inf(1)},
		"полтора года":            {1.5, 1.5},
		"занимается 4 года":       {4, 4},
	} {
		if got := durationsV5(strings.ToLower(text)); len(got) != 1 || got[0].term != want {
			t.Errorf("%q: %+v", text, got)
		}
	}
	for _, text := range []string{"в 2025 году окончил школу", "с сентября 2025 года", "много лет", "«Зарница 2.0»"} {
		if got := durationsV5(text); len(got) != 0 {
			t.Errorf("%q: %+v", text, got)
		}
	}
	activities := activityTermsV5(comparisonPayload(t, "02 Спорт и военно-патриотическое.json"))
	for _, text := range []string{
		"Больше пяти лет занимается самбо, раньше занимался лёгкой атлетикой.",
		"Меньше года участвовал в волонтёрских акциях.",
		"Три года командовал отделением в «Юнармии».",
		"Год служил в армии.",
	} {
		if problems := termProblemsV5("абзац", text, activities); len(problems) != 0 {
			t.Errorf("%q: %q", text, problems)
		}
	}
	if problems := termProblemsV5("абзац", "Два года занимается самбо.", activities); len(problems) != 1 || !strings.Contains(problems[0], "больше 5 лет") {
		t.Errorf("самбо: %q", problems)
	}
}

func TestV5StudentWithoutActivitiesGetsTheFixedParagraph(t *testing.T) {
	payload := levelsPayload(nil)
	good := map[string]any{"paragraphs": []any{noActivities + "."}, "areas": []any{}, "psychologist": []any{}, "training": []any{}, "consider": []any{}}
	if problems := checkAutobiographyAnswerV5(good, payload); len(problems) != 0 {
		t.Fatalf("the fixed paragraph fails: %v", problems)
	}
	if problems := checkAutobiographyAnswerV5(good, levelsPayload(nil, "Чтение")); len(problems) == 0 {
		t.Fatal("the fixed paragraph passes with a hobby")
	}
}

// The program writes the line of an area without experience (D2).
func TestV5LinesOfAreasWithoutExperience(t *testing.T) {
	form := autobiography(t)
	for _, name := range []string{"01 Минимальная.json", "04 Служил, работал, женат.json"} {
		potential := autobiographyPotential(form, answersOf(t, form, comparisonAnswers[name]))
		for _, area := range potential.Areas {
			if want := withoutExperienceV5[area.ID]; (area.Level == "none") != (area.NoExperience == want) || want == "" {
				t.Errorf("%s, %s (%s): %q", name, area.ID, area.Level, area.NoExperience)
			}
		}
	}
	for _, area := range potentialAreas {
		if line := withoutExperienceV5[area.ID]; !strings.HasSuffix(line, "не указал.") {
			t.Errorf("%s: %q", area.ID, line)
		}
	}
}

func TestV5AnswerThatBreaksTheRulesFailsAfterFiveTries(t *testing.T) {
	f := newAutobiographyFixture(t)
	m := portraitModel(func(a map[string]any) {
		a["paragraphs"].([]any)[1] = "По опыту в чтении — усидчивость."
	})
	f.service.UseModel(m)
	f.submit()
	f.idle()
	view := f.portrait(f.submission)
	if m.count() != autobiographyAttemptsV5 || view.Status != portraitFailed || !strings.Contains(view.Reason, "машинный оборот") {
		t.Fatalf("%d calls, %+v", m.count(), view)
	}
	if c := m.calls[0]; c.MaxTokens != autobiographyTokensV5 || !strings.Contains(string(c.Schema), `"psychologist"`) {
		t.Fatalf("completion: %d tokens, %s", c.MaxTokens, c.Schema)
	}
	m.reply = portraitModel(nil).reply
	if retry, err := f.service.RetryPortrait(f.student.OwnerID, "autobiography", f.submission); err != nil || retry.Status == portraitFailed {
		t.Fatalf("retry: %+v %v", retry, err)
	}
	f.idle()
	if view := f.portrait(f.submission); view.Status != portraitReady || view.Portrait.Version != "v5" {
		t.Fatalf("after retry: %+v", view)
	}
}

// The input of v5 is that of v4: while the conclusion by v5 is made, the one
// by v4 on the same answers is shown.
func TestV4IsShownWhileV5IsMade(t *testing.T) {
	f := newAutobiographyFixtureAt(t, 4)
	f.service.UseModel(portraitModelV4(nil))
	f.submit()
	f.idle()
	if view := f.portrait(f.submission); view.Status != portraitReady || view.Portrait.Version != "v4" {
		t.Fatalf("v4: %+v", view)
	}
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	f.service.UseModel(nil)
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(prompts...)
	if view := f.portrait(f.submission); view.Status != portraitReady || view.Portrait.Version != "v4" {
		t.Fatalf("v4 while v5 is not made: %+v", view)
	}
	f.service.UseModel(portraitModel(nil))
	f.idle()
	if view := f.portrait(f.submission); view.Status != portraitReady || view.Portrait.Version != "v5" || !strings.Contains(string(view.Portrait.Output), `"areas"`) {
		t.Fatalf("v5: %+v", view)
	}
}
