package usecase

import (
	"math"
	"vucdiagnostic/internal/domain/model"
)

// The comparison of the student with the image of the defender of the
// Fatherland by three points: the image, the ideal self and the real self, by
// the five groups of the first chapter; and the congruence of the real and the
// ideal self by C. Rogers. The rules are the decisions Д1, Д2 and Д5 of
// Вики/Тестирование/План портрета test-v2.md, approved on 23 October 2026.

// RegisterDefender connects the tables of the image (questionbank.Defender).
// Without them the portrait has no groups of the image.
func (a *Service) RegisterDefender(image model.DefenderImage) {
	a.defender = model.Clone(image)
}

// imageTop is the image on the scale of module A: it holds the qualities of the
// group in full, the top of the scale 0–10.
const imageTop = 10.0

// The situations of a group (Д1) and the type of recommendation each calls for.
const (
	situationBelow       = "below"
	situationNotAccepted = "not_accepted"
	situationFar         = "far"
	situationAchieved    = "achieved"
)

var situationText = map[string]string{
	situationBelow:       "Я-идеальное ниже Я-реального",
	situationNotAccepted: "Образ не принят как свой",
	situationFar:         "Образ принят, до него далеко",
	situationAchieved:    "Образ принят и во многом достигнут",
	"":                   "не определена",
}

var situationType = map[string]string{
	situationBelow:       "reflexive",
	situationNotAccepted: "value",
	situationFar:         "activity",
	situationAchieved:    "support",
}

// defenderSituation checks the rounded distances in order: the ideal below the
// real, the image not taken as one's own (10 − I ≥ 3), the image far (I − R ≥ 3),
// otherwise reached. The thresholds are those of Prikhozhan ×10, as in gapText.
func defenderSituation(distance, gap float64) string {
	switch {
	case math.Round(gap) < 0:
		return situationBelow
	case math.Round(distance) >= 3:
		return situationNotAccepted
	case math.Round(gap) >= 3:
		return situationFar
	}
	return situationAchieved
}

// DefenderKind is a kind of activity that forms the group, with its level.
type DefenderKind struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Forms     string `json:"forms,omitempty"`
	Level     string `json:"level"`
	LevelText string `json:"levelText"`
}

// DefenderGroup is one group of the image. Real and Ideal are the means of
// module A over the qualities answered in both images; Distance is 10 − Ideal,
// Gap is Ideal − Real. All are nil when no quality of the group is answered:
// then there is no situation. Statements is the mean of module B (1–5) over the
// qualities, a description without a level.
type DefenderGroup struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	Real          *float64         `json:"real"`
	Ideal         *float64         `json:"ideal"`
	Distance      *float64         `json:"distance"`
	Gap           *float64         `json:"gap"`
	Situation     string           `json:"situation"`
	SituationText string           `json:"situationText"`
	Type          string           `json:"type"`
	TypeText      string           `json:"typeText"`
	Answered      int              `json:"answered"`
	Total         int              `json:"total"`
	Statements    *float64         `json:"statements"`
	Indicators    []IndicatorScore `json:"indicators"`
	Signs         []string         `json:"signs"`
	Methods       []MethodLevel    `json:"methods"`
	Kinds         []DefenderKind   `json:"kinds"`
	Conditions    []IndicatorScore `json:"conditions"`
	Note          string           `json:"note,omitempty"`
}

// round1 rounds to tenths; a negative zero becomes zero.
func round1(v float64) float64 {
	if v = math.Round(v*10) / 10; v == 0 {
		return 0
	}
	return v
}

func meanOf(xs []float64) float64 {
	total := 0.0
	for _, x := range xs {
		total += x
	}
	return total / float64(len(xs))
}

// defenderGroups compares the counted portrait with the image; score gives the
// mean of an indicator over its statements (1–5).
func defenderGroups(image model.DefenderImage, data *TestPortraitData, score func(id string) IndicatorScore) []DefenderGroup {
	if len(image.Groups) == 0 {
		return nil
	}
	pairs := map[string]ImagePair{}
	for _, pair := range data.Images {
		pairs[pair.ID] = pair
	}
	methods := map[string]MethodLevel{}
	for _, m := range data.Methods {
		methods[m.ID] = m
	}
	kinds := map[string]KindLevel{}
	for _, k := range data.Kinds {
		kinds[k.ID] = k
	}
	types := map[string]string{}
	for _, kind := range image.Types {
		types[kind.ID] = kind.Name
	}
	groups := []DefenderGroup{}
	for _, def := range image.Groups {
		g := DefenderGroup{ID: def.ID, Name: def.Name, Description: def.Description, Total: len(def.Qualities), Note: def.Note,
			Indicators: []IndicatorScore{}, Signs: []string{}, Methods: []MethodLevel{}, Kinds: []DefenderKind{}, Conditions: []IndicatorScore{}}
		real, ideal, statements := []float64{}, []float64{}, []float64{}
		for _, q := range def.Qualities {
			if pair, ok := pairs[q]; ok && pair.Real != nil && pair.Ideal != nil {
				real, ideal = append(real, *pair.Real), append(ideal, *pair.Ideal)
			}
			if s := score(q).Score; s != nil {
				statements = append(statements, *s)
			}
		}
		g.Answered = len(real)
		if len(real) > 0 {
			r, i := round1(meanOf(real)), round1(meanOf(ideal))
			distance, gap := round1(imageTop-i), round1(i-r)
			g.Real, g.Ideal, g.Distance, g.Gap = &r, &i, &distance, &gap
			g.Situation = defenderSituation(distance, gap)
			g.Type = situationType[g.Situation]
			g.TypeText = types[g.Type]
		}
		g.SituationText = situationText[g.Situation]
		if len(statements) > 0 {
			v := math.Round(meanOf(statements)*100) / 100
			g.Statements = &v
		}
		for _, id := range def.Indicators {
			if row := score(id); row.Score != nil {
				g.Indicators = append(g.Indicators, row)
			}
		}
		for _, tag := range def.Signs {
			g.Signs = append(g.Signs, signNames[tag])
		}
		for _, id := range def.Methods {
			if m, ok := methods[id]; ok {
				g.Methods = append(g.Methods, m)
			}
		}
		for _, link := range def.Kinds {
			if k, ok := kinds[link.ID]; ok {
				g.Kinds = append(g.Kinds, DefenderKind{ID: k.ID, Name: k.Name, Forms: link.Forms, Level: k.Level, LevelText: k.LevelText})
			}
		}
		for _, id := range def.Conditions {
			g.Conditions = append(g.Conditions, score(id))
		}
		groups = append(groups, g)
	}
	return groups
}

// Congruence is the agreement of the profiles of the real and the ideal self
// over the qualities of module A: Pearson's coefficient and its description,
// or why it is not counted (Value is nil then). There are no norms of Rogers
// for this scale: it is a description, not a level.
type Congruence struct {
	Value     *float64 `json:"value"`
	Text      string   `json:"text"`
	Qualities int      `json:"qualities"`
}

// congruenceMinimum is the fewest qualities a coefficient is counted over.
const congruenceMinimum = 3

// congruenceText describes a coefficient rounded to hundredths by the usual
// scale of the strength of a link.
func congruenceText(r float64) string {
	switch {
	case r >= 0.7:
		return "профили во многом совпадают"
	case r >= 0.3:
		return "профили совпадают частично"
	}
	return "профили почти не совпадают"
}

// congruence counts the agreement over the qualities answered in both images;
// nil when no quality is.
func congruence(pairs []ImagePair) *Congruence {
	real, ideal := []float64{}, []float64{}
	for _, pair := range pairs {
		if pair.Real != nil && pair.Ideal != nil {
			real, ideal = append(real, *pair.Real), append(ideal, *pair.Ideal)
		}
	}
	if len(real) == 0 {
		return nil
	}
	c := &Congruence{Qualities: len(real)}
	mr, mi := meanOf(real), meanOf(ideal)
	cross, sr, si := 0.0, 0.0, 0.0
	for n := range real {
		dr, di := real[n]-mr, ideal[n]-mi
		cross, sr, si = cross+dr*di, sr+dr*dr, si+di*di
	}
	switch {
	case len(real) < congruenceMinimum:
		c.Text = "Качеств с ответами меньше трёх — согласованность не определяется."
	case si == 0:
		c.Text = "Я-идеальное одинаково по всем качествам — согласованность не определяется."
	case sr == 0:
		c.Text = "Я-реальное одинаково по всем качествам — согласованность не определяется."
	default:
		r := math.Round(cross/math.Sqrt(sr*si)*100) / 100
		c.Value = &r
		c.Text = congruenceText(r)
	}
	return c
}
