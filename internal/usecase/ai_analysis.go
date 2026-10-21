package usecase

// Local AI reports only explain deterministic scores. They do not compute new
// scores, infer protected beliefs, diagnose, or make decisions about students.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"vucdiagnostic/internal/domain/model"
)

type AIIndicator struct {
	Status         string   `json:"status"`
	Definition     string   `json:"definition,omitempty"`
	Evidence       string   `json:"evidence"`
	Interpretation string   `json:"interpretation"`
	NextStep       string   `json:"nextStep"`
	Discussion     string   `json:"discussion,omitempty"`
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Score          *float64 `json:"score"`
	Coverage       int      `json:"coverage"`
	Total          int      `json:"total"`
	N              int      `json:"n,omitempty"`
	Text           string   `json:"text"`
}

type AIReport struct {
	FormatVersion   string        `json:"formatVersion"`
	GenerationMode  string        `json:"generationMode"`
	SourceNote      string        `json:"sourceNote"`
	DataQuality     string        `json:"dataQuality"`
	Findings        []string      `json:"findings"`
	Kind            string        `json:"kind"`
	Model           string        `json:"model"`
	Summary         string        `json:"summary"`
	ModelResponses  []string      `json:"modelResponses,omitempty"`
	GenerationError string        `json:"generationError,omitempty"`
	Indicators      []AIIndicator `json:"indicators"`
	Recommendations []string      `json:"recommendations"`
	Limitations     string        `json:"limitations"`
	Participants    int           `json:"participants,omitempty"`
	Included        int           `json:"included,omitempty"`
	Excluded        int           `json:"excluded,omitempty"`
	Cohort          string        `json:"cohort,omitempty"`
	Created         string        `json:"created"`
}

type AIReportState struct {
	Status string    `json:"status"`
	Report *AIReport `json:"report,omitempty"`
	Reason string    `json:"reason,omitempty"`
}

type aiMetric struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Definition  string         `json:"definition,omitempty"`
	Limitations string         `json:"limitations,omitempty"`
	Block       string         `json:"block"`
	Value       *float64       `json:"value"`
	Answered    int            `json:"answered"`
	Total       int            `json:"total"`
	N           int            `json:"n,omitempty"`
	Component   string         `json:"component,omitempty"`
	SelfImage   map[string]int `json:"selfImage,omitempty"`
}

type aiOpenAnswer struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type aiInput struct {
	OpenAnswers  []aiOpenAnswer `json:"openAnswers,omitempty"`
	Kind         string         `json:"kind"`
	Metrics      []aiMetric     `json:"metrics"`
	Participants int            `json:"participants,omitempty"`
	Included     int            `json:"included,omitempty"`
	Excluded     int            `json:"excluded,omitempty"`
	Cohort       string         `json:"cohort,omitempty"`
}

type aiLayer struct {
	mu      sync.Mutex
	model   TextModel
	jobs    map[string]AIReportState
	workers chan struct{}
}

func newAILayer() *aiLayer {
	return &aiLayer{jobs: make(map[string]AIReportState), workers: make(chan struct{}, 1)}
}

// UseAnalysisModel sets an explicitly local model (independent of GigaChat).
func (a *Service) UseAnalysisModel(model TextModel) {
	a.ai.mu.Lock()
	defer a.ai.mu.Unlock()
	a.ai.model = model
	a.ai.jobs = make(map[string]AIReportState)
}

func (a *Service) studentAIInput(s model.Session) aiInput {
	engine := a.engineFor(s)
	metrics := []aiMetric{}
	for _, ind := range engine.bank.Indicators {
		if ind.Descriptive || !engine.indicatorEnabled(s.Profile, ind.ID) {
			continue
		}
		answered := 0
		for _, id := range ind.Items {
			if _, ok := s.Answers[id]; ok {
				answered++
			}
		}
		metric := aiMetric{ID: ind.ID, Name: ind.Name, Definition: ind.Definition, Limitations: ind.Limitations, Block: ind.Block, Component: ind.Component, Value: engine.mean(s, ind.Items), Answered: answered, Total: len(ind.Items)}

		// Four self-images are separate from the 1..5 questionnaire: R/I are
		// personal perceptions; Z/N are the student's guesses about others.
		if strings.HasPrefix(ind.ID, "Q") {
			for _, aspect := range []string{"R", "I", "Z", "N"} {
				if v, ok := s.Answers[ind.ID+"_"+aspect]; ok && v >= 0 && v <= 10 {
					if metric.SelfImage == nil {
						metric.SelfImage = make(map[string]int)
					}
					metric.SelfImage[aspect] = v
				}
			}
		}
		metrics = append(metrics, metric)
	}
	open := []aiOpenAnswer{}
	for _, item := range a.itemsFor(s) {
		_, answered := s.Answers[item.ID]
		if item.Type == "textarea" && answered && strings.TrimSpace(s.Notes[item.ID]) != "" {
			open = append(open, aiOpenAnswer{ID: item.ID, Question: item.Text, Answer: s.Notes[item.ID]})
		}
	}
	return aiInput{Kind: "student", Metrics: metrics, OpenAnswers: open}
}

// platoonAIInput uses the latest completed session per student, then the
// largest common questionnaire cohort. Incompatible profiles are excluded.
func (a *Service) platoonAIInput(db model.Database, teacherID, group string) (aiInput, error) {
	if strings.TrimSpace(group) == "" {
		return aiInput{}, problem(Invalid, "Выберите взвод.")
	}
	latest := map[string]model.Session{}
	for _, r := range db.Sessions {
		s := r.Session
		if s.Kind != "student" || !s.Completed || s.Group != group || !teacherOwnsSession(db, teacherID, s) {
			continue
		}
		id := s.StudentID
		if id == "" {
			id = "legacy:" + s.Participant
		}
		if old, exists := latest[id]; !exists || old.Updated < s.Updated {
			latest[id] = s
		}
	}
	if len(latest) == 0 {
		return aiInput{}, problem(Incomplete, "Во взводе ещё нет завершённых тестирований.")
	}
	cohorts := map[string][]model.Session{}
	for _, s := range latest {
		key := s.BankVersion + ":" + s.Profile.ID
		cohorts[key] = append(cohorts[key], s)
	}
	keys := make([]string, 0, len(cohorts))
	for key := range cohorts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(cohorts[keys[i]]) != len(cohorts[keys[j]]) {
			return len(cohorts[keys[i]]) > len(cohorts[keys[j]])
		}
		return keys[i] < keys[j]
	})
	cohort := cohorts[keys[0]]
	// With fewer than three participants, return a structured coverage-only
	// report. Individual scores are never exposed as a group average.
	reference := cohort[0]
	engine := a.engineFor(reference)
	metrics := []aiMetric{}
	for _, ind := range engine.bank.Indicators {
		if ind.Descriptive || !engine.indicatorEnabled(reference.Profile, ind.ID) {
			continue
		}
		sum, n, answered := 0.0, 0, 0
		for _, s := range cohort {
			for _, id := range ind.Items {
				if _, ok := s.Answers[id]; ok {
					answered++
				}
			}
			if v := a.engineFor(s).mean(s, ind.Items); v != nil {
				sum += *v
				n++
			}
		}
		var value *float64
		if n >= 3 {
			v := sum / float64(n)
			value = &v
		}
		metrics = append(metrics, aiMetric{ID: ind.ID, Name: ind.Name, Definition: ind.Definition, Limitations: ind.Limitations, Block: ind.Block, Value: value, Answered: answered, Total: len(ind.Items) * len(cohort), N: n})
	}
	return aiInput{Kind: "platoon", Metrics: metrics, Participants: len(latest), Included: len(cohort), Excluded: len(latest) - len(cohort), Cohort: keys[0]}, nil
}

func (a *Service) analysisInput(teacherID, kind, id string) (aiInput, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return aiInput{}, err
	}
	if kind == "platoon" {
		return a.platoonAIInput(db, teacherID, id)
	}
	if kind != "student" {
		return aiInput{}, problem(Invalid, "Неизвестный вид отчёта.")
	}
	for _, rec := range db.Sessions {
		s := rec.Session
		if s.ID != id {
			continue
		}
		if s.Kind == "observer" || !teacherOwnsSession(db, teacherID, s) {
			return aiInput{}, problem(NotFound, "Тестирование не найдено.")
		}
		if !s.Completed {
			return aiInput{}, problem(Incomplete, "ИИ-отчёт доступен только для завершённого тестирования.")
		}
		return a.studentAIInput(s), nil
	}
	return aiInput{}, problem(NotFound, "Тестирование не найдено.")
}

// AIAnalysisFor sends no identity. Student answers go only to the local model.
// GET observes the current job; POST starts fresh unless a job is already running.
func (a *Service) AIAnalysisFor(teacherID, kind, id string, start bool) (AIReportState, error) {
	input, err := a.analysisInput(teacherID, kind, id)
	if err != nil {
		return AIReportState{}, err
	}
	a.ai.mu.Lock()
	model := a.ai.model
	if model == nil {
		a.ai.mu.Unlock()
		return AIReportState{Status: "off", Reason: "Локальная модель не подключена."}, nil
	}
	data, err := json.Marshal(input)
	if err != nil {
		a.ai.mu.Unlock()
		return AIReportState{}, err
	}
	sum := sha256.Sum256(append([]byte("ai-reports-v4-model-only:"+model.Name()+":"), data...))
	key := "batched-report-v2:" + hex.EncodeToString(sum[:])
	old, found := a.ai.jobs[key]
	if found && (!start || old.Status == "working") {
		a.ai.mu.Unlock()
		return old, nil
	}
	if !start {
		a.ai.mu.Unlock()
		return AIReportState{Status: "idle"}, nil
	}
	// Prevent unbounded memory growth while retaining all running jobs.
	if len(a.ai.jobs) > 300 {
		for k, state := range a.ai.jobs {
			if state.Status != "working" {
				delete(a.ai.jobs, k)
			}
			if len(a.ai.jobs) <= 220 {
				break
			}
		}
	}
	a.ai.jobs[key] = AIReportState{Status: "working"}
	a.ai.mu.Unlock()
	go a.generateAIReport(key, model, input)
	return AIReportState{Status: "working"}, nil
}

// aiSafety restricts the *optional* LLM contribution to discussion questions.
// All findings, evidence, coverage, interpretations and recommendations are
// created deterministically below; the model never authors the main verdict.
const aiSafety = `Ты помощник преподавателя. По предоставленным ПОДСЧИТАННЫМ показателям предложи только по одному конкретному, нейтральному вопросу для педагогического обсуждения на каждый ID.
Ответ только JSON вида {"indicators":[{"id":"Q01","text":"... ?"}]}. Сохраняй ID, не пропускай и не добавляй показатели.
Каждый text должен быть одним вопросительным предложением до 220 символов, завершаться знаком ?. Вопрос должен касаться темы названия/определения показателя, а не личности студента как факта. Не оценивай личностные качества, убеждения, лояльность, мораль, медицинские состояния, перспективы службы или дисциплину. Не делай выводов о человеке и не повторяй числовые результаты. Не используй пустые фразы, психологические диагнозы и общие оценки. Никаких итоговых выводов, характеристик личности или дополнительных полей.`

type aiGenerated struct {
	Indicators []struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"indicators"`
}

func aiQuestionsSchema(count int) json.RawMessage {
	obj := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"indicators"}, "properties": map[string]any{
		"indicators": map[string]any{"type": "array", "minItems": count, "maxItems": count, "items": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"id", "text"}, "properties": map[string]any{"id": map[string]any{"type": "string"}, "text": map[string]any{"type": "string"}}}},
	}}
	b, _ := json.Marshal(obj)
	return b
}

func (a *Service) generateAIReport(key string, model TextModel, input aiInput) {
	a.ai.workers <- struct{}{}
	defer func() { <-a.ai.workers }()
	started := time.Now()
	log.Printf("AI report started: model=%s kind=%s indicators=%d open_answers=%d", model.Name(), input.Kind, len(condensedNames(input)), len(input.OpenAnswers))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	report, err := produceAIReport(ctx, model, input)
	state := AIReportState{Status: "ready", Report: report}
	if err != nil {
		state = AIReportState{Status: "failed", Reason: "Не удалось сформировать отчёт. Попробуйте позже"}
		log.Printf("AI report failed after %s: %v", time.Since(started).Round(time.Second), err)
	} else {
		log.Printf("AI report ready after %s: model_responses=%d partial=%t", time.Since(started).Round(time.Second), len(report.ModelResponses), report.GenerationError != "")
	}
	a.ai.mu.Lock()
	a.ai.jobs[key] = state
	a.ai.mu.Unlock()
}

func aiNumber(v float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.2f", v), ".", ",")
}

// aiMetricReady is intentionally strict: a score without documented coverage
// is not presented as a valid result. Platoon scores require >=3 contributors.
func aiMetricReady(input aiInput, m aiMetric) bool {
	if m.Value == nil || math.IsNaN(*m.Value) || math.IsInf(*m.Value, 0) || m.Total <= 0 || m.Answered <= 0 || m.Answered > m.Total {
		return false
	}
	if input.Kind == "platoon" {
		return m.N >= 3
	}
	return m.Answered == m.Total
}

func aiEvidence(input aiInput, m aiMetric) string {
	if !aiMetricReady(input, m) {
		if input.Kind == "platoon" {
			return fmt.Sprintf("Заполнено %d из %d пунктов; полностью рассчитано для %d участников. Для публикации среднего нужно не менее трёх участников с полным расчётом.", m.Answered, m.Total, m.N)
		}
		return fmt.Sprintf("Заполнено %d из %d пунктов; итоговое значение по методике не рассчитано.", m.Answered, m.Total)
	}
	if input.Kind == "platoon" {
		return fmt.Sprintf("Среднее %s из 5 по %d участникам с полным расчётом; заполнено %d из %d пунктов во взводе.", aiNumber(*m.Value), m.N, m.Answered, m.Total)
	}
	return fmt.Sprintf("Среднее значение %s из 5; заполнено %d из %d пунктов.", aiNumber(*m.Value), m.Answered, m.Total)
}

func aiInterpretation(input aiInput, m aiMetric) string {
	if strings.HasPrefix(m.ID, "P") && aiMetricReady(input, m) {
		return "Это заявленная гражданская позиция; она не используется для оценки личности, психологического состояния или пригодности."
	}
	if !aiMetricReady(input, m) {
		return "Числовое значение не установлено: интерпретация по данному показателю не формируется. Отсутствие расчёта не означает низкий результат."
	}
	delta := math.Abs(5 - *m.Value)
	comparison := fmt.Sprintf("Разница с установленным в шаблоне ориентиром 5,00 составляет %s балла. ", aiNumber(delta))
	if m.Block == "periphery" {
		return comparison + "Показатель отражает восприятие условий обучения, а не объективную оценку преподавателя или образовательной среды."
	}
	if input.Kind == "platoon" {
		return comparison + "Это агрегированная оценка по группе, которую нельзя переносить на каждого студента."
	}
	return comparison + "Это описание ответов обучающегося, а не доказательство личностной черты или реального поведения."
}

func aiNextStep(input aiInput, m aiMetric) string {
	if !aiMetricReady(input, m) {
		if input.Kind == "platoon" {
			return "Проверить полноту групповых данных; при необходимости провести повторную диагностику по той же методике."
		}
		return "Проверить незаполненные пункты и при согласии обучающегося предложить завершить ответы."
	}
	if input.Kind == "platoon" {
		return "Обсудить групповые условия обучения без выводов об отдельных участниках."
	}
	if m.Block == "periphery" {
		return "Уточнить, какие учебные ситуации связаны с таким восприятием условий."
	}
	return "Сопоставить самоотчёт с конкретными учебными ситуациями в беседе с обучающимся."
}

func aiDefaultQuestion(input aiInput, m aiMetric) string {
	if !aiMetricReady(input, m) {
		return ""
	}
	if input.Kind == "platoon" {
		return "Какие условия обучения могут быть связаны с ответами по этому показателю во взводе?"
	}
	if m.Block == "periphery" {
		return "Какие конкретные учебные ситуации повлияли на ответы по этому показателю?"
	}
	return "Какой пример из учебной практики помог бы уточнить смысл этих ответов?"
}

// Questions that cannot be safely validated are ignored; there is no attempt
// to mechanically "sanitize" arbitrary claims from a small language model.
func aiValidQuestion(text string) bool {
	text = strings.TrimSpace(text)
	if len([]rune(text)) < 25 || len([]rune(text)) > 220 || !strings.HasSuffix(text, "?") || strings.Count(text, "?") != 1 || strings.ContainsAny(text, "\n\r") {
		return false
	}
	lowered := strings.ToLower(text)
	startsWithQuestion := false
	for _, opening := range []string{"как ", "какие ", "какой ", "каким ", "какова ", "что ", "чем ", "почему ", "когда ", "где ", "насколько ", "можно ли ", "в каких ", "в какой ", "в каком ", "с какими ", "от чего ", "какое "} {
		if strings.HasPrefix(lowered, opening) {
			startsWithQuestion = true
			break
		}
	}
	if !startsWithQuestion || strings.ContainsAny(text, ".!") {
		return false
	}
	for _, forbidden := range []string{
		"патриотич", "лояльност", "психическ", "диагноз", "пригодност", "дисциплинар", "наказан", "отчислен",
		"недостаточн", "ограниченн", "склонен", "склонна", "личностные черты", "убеждени", "политическ", "радикал",
	} {
		if strings.Contains(lowered, forbidden) {
			return false
		}
	}
	return true
}

// aiStudentStateSummary provides a short, evidence-linked reading of the
// student's *current self-report*. This is not a clinical or suitability
// assessment. The model cannot replace these claims with unverified prose.
func aiStudentStateSummary(input aiInput, complete []aiMetric, total, answered, possible int) string {
	prefix := fmt.Sprintf("Текущая картина по самоотчёту: рассчитано %d из %d показателей; учтено %d из %d пунктов анкеты. ", len(complete), total, answered, possible)
	if len(complete) == 0 {
		if answered == 0 {
			return prefix + "Сведения о самовосприятии и учебном опыте пока не представлены, поэтому описывать особенности студента по тестированию нельзя. Для содержательного вывода необходимо получить ответы хотя бы по одному полному показателю."
		}
		var partly []aiMetric
		for _, m := range input.Metrics {
			if m.Answered > 0 && m.Total > 0 {
				partly = append(partly, m)
			}
		}
		sort.Slice(partly, func(i, j int) bool {
			a, b := float64(partly[i].Answered)/float64(partly[i].Total), float64(partly[j].Answered)/float64(partly[j].Total)
			if a != b {
				return a > b
			}
			return partly[i].ID < partly[j].ID
		})
		if len(partly) == 0 {
			return prefix + "Часть данных присутствует, но ни по одному показателю не хватает завершённого набора ответов. Обсудить текущее состояние по баллам пока нельзя."
		}
		m := partly[0]
		return prefix + fmt.Sprintf("Есть частичные ответы по показателю «%s» (%d из %d пунктов), но ни один показатель ещё не рассчитан полностью. На данном этапе можно описать только ход заполнения; оценивать состояние студента по этим данным нельзя. Следующий шаг — завершить ответы по выбранным показателям.", m.Name, m.Answered, m.Total)
	}
	personal := make([]aiMetric, 0, len(complete))
	environment := make([]aiMetric, 0, len(complete))
	position := 0
	for _, m := range complete {
		switch {
		case strings.HasPrefix(m.ID, "P"):
			// Self-described beliefs/positions are never used to judge the
			// student's personal state, trustworthiness, or suitability.
			position++
		case m.Block == "periphery" || strings.HasPrefix(m.ID, "D"):
			environment = append(environment, m)
		default:
			personal = append(personal, m)
		}
	}
	parts := make([]string, 0, 5)
	if len(personal) > 0 {
		parts = append(parts, aiStateRange(personal, "сфере самооценки", "студент выше оценил", "относительно ниже оценил"))
	}
	if len(environment) > 0 {
		parts = append(parts, aiStateRange(environment, "восприятии условий обучения", "выше оценены", "относительно ниже оценены"))
	}
	if len(personal) == 0 && len(environment) == 0 {
		parts = append(parts, "Доступны только ответы о заявленных позициях и убеждениях: их значения перечислены в разделе показателей, но сами по себе они не дают оснований описывать личностное состояние или поведение студента.")
	} else if position > 0 {
		parts = append(parts, "Ответы по вопросам гражданской позиции показаны отдельно и не используются для оценки личностного состояния.")
	}
	if len(complete) < total {
		parts = append(parts, fmt.Sprintf("Вывод предварительный: по %d из %d показателей нет полного расчёта.", total-len(complete), total))
	}
	parts = append(parts, "Это описание текущих ответов, а не диагноз и не подтверждение устойчивых черт или фактического поведения; для уточнения стоит обсудить конкретные учебные ситуации.")
	return prefix + strings.Join(parts, " ")
}

// aiStateRange compares only *within* an existing set of questionnaire
// indicators on the same 1..5 response scale. It never labels values as
// clinical risk, failures, personal defects, or levels of suitability.
func aiStateRange(metrics []aiMetric, area, highLabel, lowLabel string) string {
	sorted := append([]aiMetric(nil), metrics...)
	sort.Slice(sorted, func(i, j int) bool {
		if *sorted[i].Value != *sorted[j].Value {
			return *sorted[i].Value < *sorted[j].Value
		}
		return sorted[i].ID < sorted[j].ID
	})
	low, high := sorted[0], sorted[len(sorted)-1]
	if len(sorted) == 1 {
		return fmt.Sprintf("В %s доступен один показатель — «%s» (%s из 5); по нему нельзя описать всю область.", area, high.Name, aiNumber(*high.Value))
	}
	if math.Abs(*high.Value-*low.Value) < 0.01 {
		return fmt.Sprintf("В %s все %d доступных показателей имеют одинаковое значение %s из 5, без выраженных различий внутри этой группы.", area, len(sorted), aiNumber(*high.Value))
	}
	return fmt.Sprintf("В %s %s «%s» (%s из 5), а %s «%s» (%s из 5). Разница между ними составляет %s балла; это сопоставление внутри собственной анкеты, а не с другими студентами.", area, highLabel, high.Name, aiNumber(*high.Value), lowLabel, low.Name, aiNumber(*low.Value), aiNumber(*high.Value-*low.Value))
}

func aiCore(input aiInput, modelName string) *AIReport {
	report := &AIReport{
		FormatVersion: "2", GenerationMode: "template", Model: modelName, Kind: input.Kind,
		Participants: input.Participants, Included: input.Included, Excluded: input.Excluded, Cohort: input.Cohort,
		Created:    time.Now().UTC().Format(time.RFC3339),
		Indicators: make([]AIIndicator, 0, len(input.Metrics)), Findings: []string{},
		Recommendations: []string{},
	}
	total, calculated, answered, possible := len(input.Metrics), 0, 0, 0
	complete := make([]aiMetric, 0, len(input.Metrics))
	for _, m := range input.Metrics {
		answered += m.Answered
		possible += m.Total
		ready := aiMetricReady(input, m)
		status := "Нет расчёта"
		if ready {
			calculated++
			status = "Рассчитан"
			complete = append(complete, m)
		}
		interpretation := aiInterpretation(input, m)
		row := AIIndicator{
			ID: m.ID, Name: m.Name, Score: nil, Coverage: m.Answered, Total: m.Total, N: m.N,
			Status: status, Definition: m.Definition, Evidence: aiEvidence(input, m), Interpretation: interpretation,
			NextStep: aiNextStep(input, m), Discussion: aiDefaultQuestion(input, m),
			Text: interpretation, // backward-compatible field for older clients
		}
		if ready {
			row.Score = m.Value
		}
		report.Indicators = append(report.Indicators, row)
	}
	report.DataQuality = fmt.Sprintf("Рассчитано %d из %d показателей. Заполнено %d из %d пунктов, учитываемых для показателей; повторное включение пунктов в разные показатели возможно.", calculated, total, answered, possible)
	report.Findings = append(report.Findings, fmt.Sprintf("Из %d предусмотренных показателей числовые значения доступны по %d; без расчёта остаётся %d.", total, calculated, total-calculated))
	if input.Kind == "platoon" {
		report.DataQuality += fmt.Sprintf(" В сводку вошло %d из %d завершённых сопоставляемых по взводу тестирований; исключено %d.", input.Included, input.Participants, input.Excluded)
		report.Findings = append(report.Findings, fmt.Sprintf("Сводка включает %d тестирований одной версии методики; исключено несовместимых: %d.", input.Included, input.Excluded))
	}
	if len(complete) > 0 {
		// Extremes are descriptive *within a questionnaire*, not norms or student rankings.
		sort.Slice(complete, func(i, j int) bool {
			if *complete[i].Value != *complete[j].Value {
				return *complete[i].Value < *complete[j].Value
			}
			return complete[i].ID < complete[j].ID
		})
		low, high := complete[0], complete[len(complete)-1]
		report.Findings = append(report.Findings,
			fmt.Sprintf("Минимальное из рассчитанных значений: %s (%s) — %s из 5.", low.ID, low.Name, aiNumber(*low.Value)),
			fmt.Sprintf("Максимальное из рассчитанных значений: %s (%s) — %s из 5.", high.ID, high.Name, aiNumber(*high.Value)),
		)
		report.Summary = fmt.Sprintf("Выполнен расчёт по %d из %d показателей. Диапазон полученных средних значений: от %s до %s из 5. ", calculated, total, aiNumber(*low.Value), aiNumber(*high.Value))
		if total > calculated {
			report.Summary += fmt.Sprintf("По %d показателям числовой вывод недоступен; они не приравниваются к низким результатам. ", total-calculated)
		}
		if input.Kind == "platoon" {
			report.Summary += "Это агрегированные ответы участников одной группы тестирований, а не характеристика каждого студента."
		} else {
			report.Summary = aiStudentStateSummary(input, complete, total, answered, possible)
		}
	} else {
		report.Summary = aiStudentStateSummary(input, complete, total, answered, possible)
		if input.Kind == "platoon" {
			report.Summary = fmt.Sprintf("Для взвода не рассчитано ни одного группового показателя: по каждому требуется как минимум три полностью рассчитанных индивидуальных значения. Учтено %d совместимых тестирований. Отсутствие групповой оценки не говорит о качествах участников.", input.Included)
		}
	}
	if input.Kind == "platoon" {
		report.Recommendations = append(report.Recommendations,
			"Проверить охват взвода завершёнными и сопоставимыми тестированиями.",
			"Обсудить групповые условия обучения с преподавателями без идентификации студентов.")
	} else {
		report.Recommendations = append(report.Recommendations,
			"Обсудить с обучающимся конкретные ситуации, относящиеся к показателям с рассчитанными значениями.",
			"Не трактовать отдельные оценки как диагнозы, устойчивые качества или прогноз поведения.")
	}
	if calculated < total {
		if input.Kind == "platoon" {
			report.Recommendations = append(report.Recommendations, "Если необходимо, провести повторный сопоставимый срез для показателей без расчёта.")
		} else {
			report.Recommendations = append(report.Recommendations, "При согласии обучающегося уточнить незаполненные ответы по показателям без расчёта.")
		}
	}
	report.Limitations = "Показатели отражают самоотчёт и восприятие условий, а не доказанные личностные качества. Педагог проверяет выводы перед использованием; отчёт не применяется для автоматических кадровых, дисциплинарных и иных решений о студенте."
	if input.Kind == "platoon" {
		report.Limitations = "Агрегированные данные одной версии методики; неполные и несовместимые результаты не подменяются нулями. Выводы не относятся к отдельным студентам. Требуется проверка преподавателем."
	}
	report.SourceNote = "Факты, структура и общий вывод сформированы сервером из рассчитанных показателей. Дополнительные вопросы сформированы по шаблону."
	return report
}

// Generation fails only when no model response could be obtained.
var errAIReport = errors.New("Не удалось сформировать отчёт. Попробуйте позже")

func produceAIReport(ctx context.Context, model TextModel, input aiInput) (*AIReport, error) {
	if model == nil || ctx.Err() != nil || (len(condensedNames(input)) == 0 && len(input.OpenAnswers) == 0) {
		return nil, errAIReport
	}
	report := aiCore(input, model.Name())
	report.Summary = ""
	report.Recommendations = nil
	for i := range report.Indicators {
		row := &report.Indicators[i]
		row.NextStep, row.Discussion = "", ""
		if aiMetricReady(input, input.Metrics[i]) && !strings.HasPrefix(row.ID, "P") {
			row.Interpretation, row.Text = "", ""
		}
	}
	return aiEnrichStudentPortrait(ctx, model, input, report)
}

func countAIReady(input aiInput) int {
	count := 0
	for _, m := range input.Metrics {
		if aiMetricReady(input, m) {
			count++
		}
	}
	return count
}
