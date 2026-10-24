package usecase

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	"vucdiagnostic/internal/promptbank"
)

// goodPortrait answers the way the instruction asks, for the input it is given.
func goodPortrait(user string) map[string]any {
	user, _, _ = strings.Cut(user, "\n\n"+retryNote)
	var input struct {
		Levels   map[string]string `json:"уровни"`
		Criminal string            `json:"уголовная_ответственность"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	potential := []any{}
	for _, area := range potentialAreas {
		name := lowerFirst(area.Title)
		potential = append(potential, map[string]any{"area": name, "level": input.Levels[name], "facts": "Студент указал опыт.", "conclusion": "Опыт может пригодиться."})
	}
	consider := []any{"Нет спортивного опыта."}
	if input.Criminal == "привлекался" {
		consider = append(consider, "Привлекался к уголовной ответственности; подробности — в ответах студента.")
	}
	return map[string]any{
		"summary":   "Студент сообщил о себе, по сведениям, которые сообщил студент, следующее.",
		"potential": potential,
		"hobbies":   []any{},
		"attention": map[string]any{"use": []any{"Опыт организатора."}, "consider": consider},
	}
}

// portraitModel answers with goodPortrait changed by the mutate function.
func portraitModel(mutate func(map[string]any)) *fakeModel {
	return &fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		answer := goodPortrait(c.User)
		if mutate != nil {
			mutate(answer)
		}
		data, _ := json.Marshal(answer)
		return string(data), nil
	}}
}

func newAutobiographyFixture(t *testing.T) *formFixture {
	t.Helper()
	f := newFormFixture(t)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	f.service.RegisterPrompts(prompts...)
	f.service.portraits.pause = 0
	t.Cleanup(f.service.Close)
	return f
}

func (f *formFixture) submit() {
	f.t.Helper()
	f.consent()
	f.fillAll(nil)
	if _, err := f.command("completion", Input{}); err != nil {
		f.t.Fatal(err)
	}
}

// another is a second student of the same assignment, with the same answers.
func (f *formFixture) another(email string) *formFixture {
	f.t.Helper()
	login := signUp(f.t, f.service, "student", email, "101")
	return &formFixture{t: f.t, service: f.service, form: f.form, token: login.Token, student: login.Account, assignment: f.assignment, submission: addSubmission(f.t, f.service, login.Account.ID, f.assignment)}
}

func (f *formFixture) idle() {
	f.t.Helper()
	(&portraitFixture{t: f.t, service: f.service}).idle()
}

func (f *formFixture) portrait(submission string) PortraitView {
	f.t.Helper()
	view, err := f.service.PortraitFor(f.student.OwnerID, "autobiography", submission)
	if err != nil {
		f.t.Fatal(err)
	}
	return view
}

func TestSubmittedAutobiographyGetsAPortraitOnceAndTheSameForEqualAnswers(t *testing.T) {
	f := newAutobiographyFixture(t)
	m := portraitModel(nil)
	f.service.UseModel(m)
	f.submit()
	f.idle()
	view := f.portrait(f.submission)
	if view.Status != portraitReady || view.Portrait == nil || m.count() != 1 {
		t.Fatalf("%+v after %d calls", view, m.count())
	}
	call := m.calls[0]
	if !strings.HasPrefix(call.System, "Ты помогаешь преподавателю военного учебного центра") || !strings.Contains(string(call.Schema), `"required": ["summary", "potential", "hobbies", "attention"]`) {
		t.Fatalf("instruction or schema: %.80s", call.System)
	}
	// What was sent is the anonymised input and nothing of the person.
	db := mustSnapshot(t, f.service.repo)
	for _, banned := range []string{f.student.Name, f.student.Email, "Тестов"} {
		if strings.Contains(call.User, banned) {
			t.Fatalf("%q reached the model", banned)
		}
	}
	if db.Portraits[0].Input != call.User || !strings.HasPrefix(call.User, "{\n \"возраст\"") {
		t.Fatalf("input: %.60s", call.User)
	}

	second := f.another("second@example.test")
	second.submit()
	f.idle()
	again := f.portrait(second.submission)
	if m.count() != 1 || again.Status != portraitReady || string(again.Portrait.Output) != string(view.Portrait.Output) || len(mustSnapshot(t, f.service.repo).Portraits) != 1 {
		t.Fatalf("second student: %+v after %d calls", again, m.count())
	}
}

func TestAutobiographyPortraitIsMissingBeforeSubmissionAndHiddenFromOtherTeachers(t *testing.T) {
	f := newAutobiographyFixture(t)
	f.service.UseModel(portraitModel(nil))
	if _, err := f.service.PortraitFor(f.student.OwnerID, "autobiography", f.submission); kind(err) != NotFound {
		t.Fatalf("a draft has a portrait: %v", err)
	}
	f.submit()
	f.idle()
	for name, call := range map[string]func() error{
		"get": func() error {
			_, err := f.service.PortraitFor("another-teacher", "autobiography", f.submission)
			return err
		},
		"retry": func() error {
			_, err := f.service.RetryPortrait("another-teacher", "autobiography", f.submission)
			return err
		},
	} {
		if kind(call()) != NotFound {
			t.Errorf("%s by another teacher is not 404", name)
		}
	}
	if view, err := f.service.PortraitFor("", "autobiography", f.submission); err != nil || view.Status != portraitReady {
		t.Fatalf("without a teacher scope: %+v %v", view, err)
	}
}

func TestSubmissionWorksWithoutAModel(t *testing.T) {
	f := newAutobiographyFixture(t)
	f.submit()
	view := f.portrait(f.submission)
	if view.Status != portraitOff || view.Portrait != nil {
		t.Fatalf("%+v", view)
	}
	m := portraitModel(nil)
	f.service.UseModel(m)
	f.idle()
	if m.count() != 1 || f.portrait(f.submission).Status != portraitReady {
		t.Fatalf("not made at start: %d calls", m.count())
	}
}

func TestAnswerThatBreaksTheRulesFailsAfterThreeTries(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate  func(map[string]any)
		problem string
	}{
		"changed level": {func(a map[string]any) {
			a["potential"].([]any)[0].(map[string]any)["level"] = "высокий-превосходный"
		}, "уровень изменён"},
		"six points": {func(a map[string]any) {
			a["attention"].(map[string]any)["consider"] = []any{"1", "2", "3", "4", "5", "6"}
		}, "пунктов больше пяти"},
		"five areas": {func(a map[string]any) { a["potential"] = a["potential"].([]any)[:5] }, "направления не те"},
		"other order": {func(a map[string]any) {
			p := a["potential"].([]any)
			p[0], p[1] = p[1], p[0]
		}, "в другом порядке"},
		"no summary":   {func(a map[string]any) { a["summary"] = " " }, "нет характеристики"},
		"empty facts":  {func(a map[string]any) { a["potential"].([]any)[2].(map[string]any)["facts"] = "" }, "пустое поле facts"},
		"no attention": {func(a map[string]any) { delete(a, "attention") }, "нет блока"},
		"use not text": {func(a map[string]any) { a["attention"].(map[string]any)["use"] = []any{1} }, "не список строк"},
		"consequence in consider": {func(a map[string]any) {
			a["attention"].(map[string]any)["consider"] = []any{"Женат, имеет одного ребёнка — обстоятельства жизни, влияющие на распределение времени."}
		}, "вывод вместо факта"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newAutobiographyFixture(t)
			m := portraitModel(tc.mutate)
			f.service.UseModel(m)
			f.submit()
			f.idle()
			view := f.portrait(f.submission)
			if m.count() != 3 || view.Status != portraitFailed || !strings.Contains(view.Reason, tc.problem) {
				t.Fatalf("%d calls, %+v", m.count(), view)
			}
			// «Повторить» with a good model makes the portrait.
			m.reply = portraitModel(nil).reply
			if retry, err := f.service.RetryPortrait(f.student.OwnerID, "autobiography", f.submission); err != nil || retry.Status == portraitFailed {
				t.Fatalf("retry: %+v %v", retry, err)
			}
			f.idle()
			if f.portrait(f.submission).Status != portraitReady || m.count() != 4 {
				t.Fatalf("after retry: %+v, %d calls", f.portrait(f.submission), m.count())
			}
		})
	}
}

func TestNewInstructionVersionMakesANewPortraitAndKeepsTheOld(t *testing.T) {
	f := newAutobiographyFixture(t)
	release := make(chan struct{})
	m := &fakeModel{name: "fake", reply: func(call int, c Completion) (string, error) {
		answer := goodPortrait(c.User)
		answer["summary"] = "Текст первой версии."
		if call > 1 {
			<-release
			answer["summary"] = "Текст второй версии."
		}
		data, _ := json.Marshal(answer)
		return string(data), nil
	}}
	f.service.UseModel(m)
	f.submit()
	f.idle()
	task, _ := f.service.portraits.task("autobiography")
	next := task
	next.Version = task.Version + "-next"
	f.service.portraits.tasks = []portraitTask{next}
	f.service.UseModel(m)
	// The portrait of v1 is shown at once, the one of v2 is made in the background.
	if view := f.portrait(f.submission); view.Status != portraitReady || !strings.Contains(string(view.Portrait.Output), "первой версии") {
		t.Fatalf("while the new version is made: %+v", view)
	}
	close(release)
	f.idle()
	if view := f.portrait(f.submission); view.Status != portraitReady || !strings.Contains(string(view.Portrait.Output), "второй версии") {
		t.Fatalf("after the new version: %+v", view)
	}
	db := mustSnapshot(t, f.service.repo)
	if m.count() != 2 || len(db.Portraits) != 2 || db.Portraits[0].TaskVersion != task.Version || db.Portraits[1].TaskVersion != next.Version || db.Portraits[0].ID == db.Portraits[1].ID || db.Portraits[0].Fingerprint != db.Portraits[1].Fingerprint {
		t.Fatalf("%d calls: %+v", m.count(), db.Portraits)
	}
}

func TestPortraitUsesTheLevelsOfTheProgramAndFindsThemInTheInput(t *testing.T) {
	f := newAutobiographyFixture(t)
	f.submit()
	db := mustSnapshot(t, f.service.repo)
	payload, _, err := submissionPayload(f.form, db.Submissions[0])
	if err != nil {
		t.Fatal(err)
	}
	levels := autobiographyPotential(f.form, db.Submissions[0].Answers)
	want := []string{}
	for _, area := range levels.Areas {
		want = append(want, lowerFirst(area.Title)+"="+area.LevelText)
	}
	got := []string{}
	for _, key := range payload.values["уровни"].(*object).keys {
		got = append(got, fmt.Sprintf("%s=%s", key, payload.values["уровни"].(*object).values[key]))
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("%v and %v", got, want)
	}
	if problems := checkAutobiographyAnswer(goodPortrait(string(canonicalJSON(payload))), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
}

// The points are answers of the models in the comparison of 19 October 2026.
func TestWhatToConsiderHoldsFactsWithoutConclusions(t *testing.T) {
	payload := newObject()
	payload.set("уголовная_ответственность", "не привлекался")
	for _, point := range []string{
		"Женат, имеет одного ребёнка — обстоятельства жизни, влияющие на планирование времени и нагрузок.",
		"Женат, имеет одного ребёнка — обстоятельства жизни, влияющие на распределение времени.",
		"Служба в Вооружённых силах уже пройдена — учитывать при планировании учебных сборов и допусков.",
		"В военно-патриотической деятельности опыт ограничен учебными сборами и службой; расширять практику командных ролей.",
		"Студент обучается на факультете компьютерных наук и занимается математикой, что указывает на сильную академическую подготовку.",
		"Отец — военнослужащий, что следует учесть при распределении по нарядам.",
	} {
		if problems := considerProblems([]any{point}, payload); len(problems) != 1 || !strings.Contains(problems[0], "вывод вместо факта") || strings.Contains(problems[0], point) {
			t.Errorf("%q: %v", point, problems)
		}
	}
	for _, point := range []string{
		"Женат, имеет одного ребёнка.",
		"Служба в Вооружённых силах уже пройдена, период 2021–2022.",
		"Воспитывался матерью, сведений об отце не имеет.",
		"Отец — военнослужащий, заместитель командира батальона.",
		"У брата есть судимость по ст. 158 УК РФ, кража.",
		"Другой близкий родственник имеет гражданство Казахстана.",
		"Спортивный опыт ограничен плаванием 1–3 года, юношеский разряд.",
		"Наряду с плаванием занимался футболом.",
		"Студент не проходил военную службу по призыву.",
		"Опыт владения иностранными языками не указан.",
	} {
		if problems := considerProblems([]any{point}, payload); len(problems) != 0 {
			t.Errorf("%q: %v", point, problems)
		}
	}
	payload.set("уголовная_ответственность", "привлекался")
	for point, ok := range map[string]bool{
		"Привлекался к уголовной ответственности.":                                                     false,
		"Привлекался к уголовной ответственности; подробности — в ответах студента.":                   true,
		"Привлекался к уголовной ответственности, подробности - в ответах студента.":                   true,
		"Студент сообщил о привлечении к уголовной ответственности (подробности — в ответах студента)": true,
	} {
		if problems := considerProblems([]any{point}, payload); (len(problems) == 0) != ok {
			t.Errorf("%q: %v", point, problems)
		}
	}
}

func TestWaitingForANewPortraitDoesNotBlockTheSubmission(t *testing.T) {
	f := newAutobiographyFixture(t)
	release := make(chan struct{})
	slow := &fakeModel{name: "slow", reply: func(call int, c Completion) (string, error) {
		<-release
		data, _ := json.Marshal(goodPortrait(c.User))
		return string(data), nil
	}}
	f.service.UseModel(slow)
	started := time.Now()
	f.submit()
	if time.Since(started) > 2*time.Second {
		t.Fatal("the submission waited for the model")
	}
	if view := f.portrait(f.submission); view.Status != portraitQueued && view.Status != portraitWorking {
		t.Fatalf("%+v", view)
	}
	close(release)
	f.idle()
	if f.portrait(f.submission).Status != portraitReady {
		t.Fatal("not ready")
	}
}

// submitWithSport saves a sport record with the given achievements and submits.
func (f *formFixture) submitWithSport(achievements string) {
	f.t.Helper()
	f.consent()
	f.fillAll(nil)
	record := map[string]any{"sport_kind": "sambo", "sport_now": "now", "sport_years": "y5", "sport_rank": "kms", "sport_level": "russia", "sport_ach": achievements}
	if _, err := f.save("sport", map[string]json.RawMessage{"sport_has": raw(f.t, "yes"), "sport_list": raw(f.t, []any{record})}); err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.command("completion", Input{}); err != nil {
		f.t.Fatal(err)
	}
}

func TestPersonalDataInAchievementsIsCutOutOfTheAutobiographyInput(t *testing.T) {
	f := newAutobiographyFixture(t)
	m := portraitModel(nil)
	f.service.UseModel(m)
	f.submitWithSport("Первое место 12.05.2024, тренер Петров Иван Сергеевич, тел. 8 900 123-45-67")
	f.idle()
	if m.count() != 1 || f.portrait(f.submission).Status != portraitReady {
		t.Fatalf("%d calls, %+v", m.count(), f.portrait(f.submission))
	}
	user := m.calls[0].User
	for _, leaked := range []string{"12.05.2024", "Сергеевич", "8 900", "123-45-67"} {
		if strings.Contains(user, leaked) {
			t.Fatalf("%q reached the model", leaked)
		}
	}
	if !strings.Contains(user, "Первое место [скрыто], тренер Петров Иван [скрыто], тел. [скрыто]") {
		t.Fatalf("achievements in the input: %s", user)
	}
}

func TestTwoAutobiographiesThatDifferOnlyInADateGiveOneCallAndOnePortrait(t *testing.T) {
	f := newAutobiographyFixture(t)
	m := portraitModel(nil)
	f.service.UseModel(m)
	f.submitWithSport("Первое место 12.05.2024")
	f.idle()
	second := f.another("second@example.test")
	second.submitWithSport("Первое место 03.11.2023")
	f.idle()
	if m.count() != 1 || len(mustSnapshot(t, f.service.repo).Portraits) != 1 || f.portrait(second.submission).Status != portraitReady {
		t.Fatalf("%d calls, %d portraits", m.count(), len(mustSnapshot(t, f.service.repo).Portraits))
	}
}

// Every field the form sends is a kind of value the cut knows: with a real
// questionnaire the cut passes and finds nothing in made-up answers.
func TestEveryComparisonQuestionnairePassesTheCut(t *testing.T) {
	form := autobiography(t)
	for _, name := range sortedKeys(comparisonAnswers) {
		payload := autobiographyPayload(form, answersOf(t, form, comparisonAnswers[name]), today)
		if n, err := maskPersonal(payload); err != nil || n != 0 {
			t.Errorf("%s: %d cuts, %v", name, n, err)
		}
	}
}

func TestTheNewestInstructionVersionIsTheOneThatIsUsed(t *testing.T) {
	s := testApp(t)
	t.Cleanup(s.Close)
	schema := []byte(`{"type":"object"}`)
	s.RegisterPrompts(
		promptbank.Prompt{Task: "autobiography", Version: "v2", Instruction: "вторая", Schema: schema},
		promptbank.Prompt{Task: "autobiography", Version: "v10", Instruction: "десятая", Schema: schema},
		promptbank.Prompt{Task: "autobiography", Version: "v9", Instruction: "девятая", Schema: schema},
	)
	task, ok := s.portraits.task("autobiography")
	if !ok || task.Version != "v10" || task.Prompt.Instruction != "десятая" || len(s.portraits.tasks) != 1 {
		t.Fatalf("task: %+v", task)
	}
}

func TestRetryTellsTheModelWhatTheCheckFound(t *testing.T) {
	f := newAutobiographyFixture(t)
	messages := []string{}
	m := &fakeModel{name: "fake", reply: func(call int, c Completion) (string, error) {
		messages = append(messages, c.User)
		answer := goodPortrait(c.User)
		if call == 1 {
			answer["attention"].(map[string]any)["consider"] = []any{"Женат — обстоятельство, влияющее на нагрузку."}
		}
		data, _ := json.Marshal(answer)
		return string(data), nil
	}}
	f.service.UseModel(m)
	f.submit()
	f.idle()
	if f.portrait(f.submission).Status != portraitReady || len(messages) != 2 {
		t.Fatalf("%d calls", len(messages))
	}
	if strings.Contains(messages[0], retryNote) || !strings.HasPrefix(messages[1], messages[0]+"\n\n"+retryNote) || !strings.Contains(messages[1], "вывод вместо факта") {
		t.Fatalf("retry message: %q", messages[1][len(messages[0]):])
	}
	if stored := mustSnapshot(t, f.service.repo).Portraits[0].Input; stored != messages[0] {
		t.Fatal("the stored input has the retry note")
	}
}
