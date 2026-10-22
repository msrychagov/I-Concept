package usecase

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"vucdiagnostic/internal/domain/model"
)

var componentNames = map[string]string{"CS": "Ценностно-смысловой", "EV": "Эмоционально-волевой", "DP": "Деятельностно-практический", "O": "Организационный", "G": "Общепедагогический", "L": "Личностный"}
var related = map[string][]string{
	"Q01": {"D01", "D07", "D12"}, "Q02": {"D01", "D07", "D12"}, "Q03": {"D05", "D09", "D12"}, "Q04": {"D05", "D09", "D10"},
	"Q05": {"D03", "D07", "D12"}, "Q06": {"D01", "D04", "D05"}, "Q07": {"D07", "D09", "D11"}, "Q08": {"D03", "D09", "D11"},
	"Q09": {"D02", "D07", "D11"}, "Q10": {"D05", "D08", "D11"}, "Q11": {"D08", "D10", "D11"}, "Q12": {"D02", "D08", "D10"},
	"Q13": {"D05", "D09", "D12"}, "Q14": {"D04", "D09", "D12"}, "Q15": {"D06", "D09", "D11"}, "Q16": {"D06", "D09", "D11"},
	"Q17": {"D07", "D09", "D12"}, "Q18": {"D03", "D06", "D11"}, "Q19": {"D03", "D04", "D06"}, "Q20": {"D01", "D03", "D12"},
	"Q21": {"D01", "D09", "D12"}, "Q22": {"D02", "D08", "D10"}, "Q23": {"D03", "D06", "D09"}, "Q24": {"D03", "D08", "D10"},
	"P01": {"D04", "D05", "D12"}, "P02": {"D01", "D05", "D12"}, "P03": {"D04", "D06", "D10"},
}
var personal = map[string]string{
	"CS": "Обсудить конкретный выбор, его основания и последствия; связать принятое обязательство с личной целью.",
	"EV": "Разобрать один напряжённый эпизод; согласовать приём паузы, план действий и последующую самопроверку.",
	"DP": "Согласовать посильную роль в общем деле, наблюдаемый результат и разбор собственного вклада.",
}
var environmental = map[string]string{
	"O": "Уточнить требования, ресурсы, сроки и распределение ответственности; проверить доступность условий для студента.",
	"G": "Организовать обсуждение оснований решения, обратную связь и разбор опыта после совместного задания.",
	"L": "Обеспечить уважительный диалог, содержательный отклик на инициативу и доступное индивидуальное сопровождение.",
}

func (a *Service) itemsFor(s model.Session) []model.Item {
	out := []model.Item{}
	engine := a.engineFor(s)
	for _, i := range engine.bank.Items {
		ok := s.Kind == "observer" && i.Module == "E" || s.Kind != "observer" && (i.Module != "E" && i.Module != "F" && i.Module != "V" && !methodModule(i.Module) || i.Module == "F" && s.Activity || i.Module == "V" && (s.Motivation || s.Methods) || methodModule(i.Module) && s.Methods)
		if ok && a.indicatorEnabled(s.Profile, i.Indicator) {
			out = append(out, i)
		}
	}
	return out
}

func (a *Service) indicatorEnabled(p model.Profile, id string) bool {
	if id == "" || len(p.Values) == 0 {
		return true
	}
	known := false
	for _, indicator := range a.bank.Indicators {
		if indicator.ID == id {
			known = true
			break
		}
	}
	if !known {
		return true
	}
	_, enabled := p.Values[id]
	return enabled
}
func (a *Service) item(id string) (model.Item, bool) {
	for _, i := range a.bank.Items {
		if i.ID == id {
			return i, true
		}
	}
	return model.Item{}, false
}
func score(i model.Item, v int) float64 {
	if len(i.Options) > 0 {
		return i.Options[v].Score
	}
	if i.Reverse {
		return float64(i.Min + i.Max - v)
	}
	return float64(v)
}
func (a *Service) mean(s model.Session, ids []string) *float64 {
	if len(ids) == 0 {
		return nil
	}
	n := 0.0
	for _, id := range ids {
		v, ok := s.Answers[id]
		if !ok {
			return nil
		}
		i, ok := a.engineFor(s).item(id)
		if !ok || i.ResponseOnly || v < i.Min || v > i.Max || (len(i.Options) > 0 && v >= len(i.Options)) {
			return nil
		}
		n += score(i, v)
	}
	v := n / float64(len(ids))
	return &v
}

func (a *Service) emptyProfile() model.Profile {
	v := map[string]*float64{}
	for _, i := range a.bank.Indicators {
		target := 5.0
		v[i.ID] = &target
		if i.Descriptive {
			v[i.ID] = nil
		}
	}
	return model.Profile{Name: "Шаблон диагностики", Values: v, Approved: true, BankVersion: a.bank.Version}
}
func (a *Service) validateProfile(p model.Profile) error {
	if strings.TrimSpace(p.Name) == "" || len(p.Name) > 150 {
		return errors.New("укажите название шаблона")
	}
	if len(p.BandBasis) > 4000 {
		return errors.New("пояснение границы слишком длинное")
	}
	if p.PartialGap != nil && (*p.PartialGap <= 0 || *p.PartialGap > 4 || strings.TrimSpace(p.BandBasis) == "") {
		return errors.New("для границы укажите пояснение и значение от 0 до 4")
	}
	if len(p.Values) == 0 {
		return errors.New("оставьте в шаблоне хотя бы один показатель")
	}
	for id, v := range p.Values {
		known := false
		descriptive := false
		for _, i := range a.bank.Indicators {
			if i.ID == id {
				known = true
				descriptive = i.Descriptive
				break
			}
		}
		if !known {
			return fmt.Errorf("неизвестный показатель %s", id)
		}
		if descriptive {
			if v != nil {
				return fmt.Errorf("у описательного показателя %s не должно быть балльного ориентира", id)
			}
			continue
		}
		if v == nil || *v != 5 {
			return fmt.Errorf("ориентир %s должен быть равен 5", id)
		}
	}
	if p.Approved {
		if strings.TrimSpace(p.Author) == "" {
			return errors.New("укажите автора шаблона")
		}
	}
	return nil
}
func (a *Service) validateAnswer(s model.Session, id string, v *int) error {
	for _, i := range a.itemsFor(s) {
		if i.ID == id {
			if v == nil {
				return nil
			}
			if *v < i.Min || *v > i.Max {
				return errors.New("недопустимый ответ")
			}
			return nil
		}
	}
	return errors.New("неизвестный вопрос")
}

type reportRow struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Block     string   `json:"block"`
	Component string   `json:"component"`
	Actual    *float64 `json:"actual"`
	Target    *float64 `json:"target"`
	Gap       *float64 `json:"gap"`
	Status    string   `json:"status"`
	Coverage  int      `json:"coverage"`
	Total     int      `json:"total"`
}

func diff(target, actual *float64) *float64 {
	if target == nil || actual == nil {
		return nil
	}
	v := *target - *actual
	return &v
}
func (a *Service) analyze(s model.Session, all []model.SessionRecord) map[string]any {
	engine := a.engineFor(s)
	result := engine.analyzeSnapshot(s, all)
	result["psychological"] = engine.psychologicalReport(s)
	return result
}
func (a *Service) analyzeSnapshot(s model.Session, all []model.SessionRecord) map[string]any {
	rows := []reportRow{}
	for _, i := range a.bank.Indicators {
		if i.Descriptive {
			continue
		}
		if !a.indicatorEnabled(s.Profile, i.ID) {
			continue
		}
		actual := a.mean(s, i.Items)
		target := s.Profile.Values[i.ID]
		gap := diff(target, actual)
		status := "Недостаточно ответов для расчёта"
		if actual != nil {
			status = "Самоотчёт по утверждениям"
			if i.Block == "periphery" {
				status = "Восприятие условий студентом"
			}
		}
		coverage := 0
		for _, id := range i.Items {
			if _, ok := s.Answers[id]; ok {
				coverage++
			}
		}
		rows = append(rows, reportRow{i.ID, i.Name, i.Block, i.Component, actual, target, gap, status, coverage, len(i.Items)})
	}
	components := []map[string]any{}
	for _, code := range []string{"CS", "EV", "DP", "O", "G", "L"} {
		var av, tv float64
		n := 0
		complete := true
		for _, r := range rows {
			if r.Component == code {
				n++
				if r.Actual == nil {
					complete = false
				} else {
					av += *r.Actual
				}
				if r.Target == nil {
					tv = 0
				} else {
					tv += *r.Target
				}
			}
		}
		var ap, tp *float64
		if n > 0 && complete {
			x := av / float64(n)
			ap = &x
		}
		if n > 0 && tv > 0 {
			x := tv / float64(n)
			tp = &x
		}
		components = append(components, map[string]any{"code": code, "name": componentNames[code], "actual": ap, "target": tp, "n": n, "total": n})
	}
	tags := []map[string]any{}
	for _, tag := range []string{"LOVE", "SOB", "PAT"} {
		sum := 0.0
		n, total := 0, 0
		for _, i := range a.bank.Indicators {
			if !a.indicatorEnabled(s.Profile, i.ID) {
				continue
			}
			if contains(i.Tags, tag) {
				total++
				if row, ok := rowByIndicator(rows, i.ID); ok && row.Actual != nil {
					sum += *row.Actual
					n++
				}
			}
		}
		var actual *float64
		if n == total && n > 0 {
			x := sum / float64(n)
			actual = &x
		}
		names := map[string]string{"LOVE": "Любовь", "SOB": "Соборность", "PAT": "Патриотизм"}
		tags = append(tags, map[string]any{"tag": tag, "name": names[tag], "actual": actual, "n": n, "total": total})
	}
	images := []map[string]any{}
	for _, i := range a.bank.Indicators {
		if strings.HasPrefix(i.ID, "Q") && a.indicatorEnabled(s.Profile, i.ID) {
			m := map[string]any{"id": i.ID, "name": i.Name}
			for _, k := range []string{"R", "I", "Z", "N"} {
				if v, ok := s.Answers[i.ID+"_"+k]; ok {
					m[k] = v
				} else {
					m[k] = nil
				}
			}
			for _, d := range [][3]string{{"RI", "I", "R"}, {"RN", "N", "R"}, {"RZ", "Z", "R"}, {"IN", "N", "I"}} {
				x, xok := m[d[1]].(int)
				y, yok := m[d[2]].(int)
				if xok && yok {
					m[d[0]] = x - y
				} else {
					m[d[0]] = nil
				}
			}
			images = append(images, m)
		}
	}
	sjt := []map[string]any{}
	for _, i := range a.itemsFor(s) {
		if i.Module == "S" {
			m := map[string]any{"id": i.ID, "indicator": i.Indicator, "text": i.Text, "raw": nil, "keyStatus": "author_draft_expert_validation_required", "limitation": "Гипотетический выбор; авторское пояснение не подтверждает реальное поведение."}
			if v, ok := s.Answers[i.ID]; ok && v >= 0 && v < len(i.Options) {
				o := i.Options[v]
				m["raw"] = v
				m["score"] = score(i, v)
				m["level"] = o.Level
				m["selectedText"] = o.Text
				m["interpretation"] = o.Rationale
			}
			sjt = append(sjt, m)
		}
	}
	history := []map[string]any{}
	for _, r := range all {
		x := r.Session
		if x.Kind == "student" && x.Completed && sameParticipant(s, x) && a.comparable(s, x) {
			cm := map[string]any{}
			for _, code := range []string{"CS", "EV", "DP", "O", "G", "L"} {
				sum := 0.0
				n := 0
				ok := true
				for _, i := range a.bank.Indicators {
					if i.Component == code {
						n++
						v := a.mean(x, i.Items)
						if v == nil {
							ok = false
						} else {
							sum += *v
						}
					}
				}
				if ok && n > 0 {
					cm[code] = sum / float64(n)
				} else {
					cm[code] = nil
				}
			}
			history = append(history, map[string]any{"id": x.ID, "wave": x.Wave, "date": measurementDate(x), "profileId": x.Profile.ID, "components": cm})
		}
	}
	sort.Slice(history, func(i, j int) bool { return history[i]["date"].(string) < history[j]["date"].(string) })

	rowByID := map[string]reportRow{}
	for _, r := range rows {
		rowByID[r.ID] = r
	}
	recommendations := []map[string]any{}
	for _, r := range rows {
		if r.Block != "core" || r.Actual == nil || r.Target == nil {
			continue
		}
		envs := []map[string]any{}

		for _, id := range related[r.ID] {
			d, exists := rowByID[id]
			if !exists {
				continue
			}
			envs = append(envs, map[string]any{"id": d.ID, "name": d.Name, "actual": d.Actual, "target": d.Target, "gap": d.Gap})
		}
		route, action := "Сопоставление источников", "Сопоставить самоописание с ситуационным выбором и предыдущими ответами; обсудить доступные условия, согласовать следующий шаг и дату повторной диагностики."
		choices := []map[string]any{}
		for _, x := range sjt {
			if x["indicator"] == r.ID && x["raw"] != nil {
				choices = append(choices, x)
			}
		}
		priority := 0
		recommendations = append(recommendations, map[string]any{"indicator": r.ID, "name": r.Name, "gap": r.Gap, "route": route, "action": action, "environment": envs, "sjt": choices, "priority": priority, "verification": "Направление беседы, не установленная причина. Сопоставить самоописание, ситуации и динамику ответов; согласовать действие, ответственного и дату проверки."})
	}
	motivation := []map[string]any{}
	for _, ind := range []string{"V_GENERAL", "V_SPECIAL"} {
		ids := []string{}
		for _, i := range a.bank.Items {
			if i.Module == "V" && i.Indicator == ind {
				ids = append(ids, i.ID)
			}
		}
		m := a.mean(s, ids)
		var sum *float64
		if m != nil {
			x := *m * float64(len(ids))
			sum = &x
		}
		name := "Общая военная доминанта"
		if ind == "V_SPECIAL" {
			name = "Специальная военная доминанта"
		}
		motivation = append(motivation, map[string]any{"name": name, "sum": sum, "min": 6, "max": 24})
	}
	activity := []map[string]any{}
	for _, i := range a.itemsFor(s) {
		if i.Module == "F" {
			var value any = nil
			if v, ok := s.Answers[i.ID]; ok {
				value = v
			}
			if i.Type == "textarea" {
				value = nil
				if _, ok := s.Answers[i.ID]; ok {
					value = s.Notes[i.ID]
				}
			}
			activity = append(activity, map[string]any{"id": i.ID, "text": i.Text, "value": value, "type": i.Type, "topic": i.Topic, "note": s.Notes[i.ID]})
		}
	}
	return map[string]any{"evidence": a.evidence(s, all, rows), "interpretationNote": "Диагностический профиль описывает ответы студента, а не процент воспитанности. Источники не складываются в общий балл. Числовая динамика сама по себе не доказывает влияние педагогических условий.", "rows": rows, "components": components, "tags": tags, "images": images, "sjt": sjt, "history": history, "recommendations": recommendations, "motivation": motivation, "activity": activity, "profile": s.Profile, "notes": s.Notes, "plan": s.Plan, "answered": len(s.Answers), "total": len(a.itemsFor(s))}
}
func contains(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}

func rowByIndicator(rows []reportRow, id string) (reportRow, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return reportRow{}, false
}

func (a *Service) writeCSV(w io.Writer, sessions []model.SessionRecord) error {
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	write := func(row []string) error {
		for i, v := range row {
			if i > 0 {
				if _, err := io.WriteString(w, ";"); err != nil {
					return err
				}
			}
			if strings.ContainsAny(v, "\";\r\n") {
				v = "\"" + strings.ReplaceAll(v, "\"", "\"\"") + "\""
			}
			if _, err := io.WriteString(w, v); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, "\r\n")
		return err
	}
	if err := write([]string{"session_id", "participant", "group", "wave", "kind", "observer", "bank_version", "profile_id", "item_id", "module", "indicator", "raw", "scored", "level", "note", "option_id", "answer_text"}); err != nil {
		return err
	}
	for _, r := range sessions {
		s := r.Session
		for _, i := range a.itemsFor(s) {
			raw, ok := s.Answers[i.ID]
			rv, sv, level := "", "", ""
			if ok {
				rv = strconv.Itoa(raw)
				if !i.ResponseOnly {
					sv = strconv.FormatFloat(score(i, raw), 'f', -1, 64)
				}
				if len(i.Options) > raw {
					level = i.Options[raw].Level
				}
			}
			optionID, answerText := "", ""
			if ok && i.Type == "textarea" {
				rv = ""
				answerText = s.Notes[i.ID]
			}
			if ok && raw >= 0 && raw < len(i.Options) {
				optionID, answerText = i.Options[raw].ID, i.Options[raw].Text
			}
			if err := write([]string{s.ID, s.Participant, s.Group, s.Wave, s.Kind, s.Observer, s.BankVersion, s.Profile.ID, i.ID, i.Module, i.Indicator, rv, sv, level, s.Notes[i.ID], optionID, answerText}); err != nil {
				return err
			}
		}
	}
	return nil
}
