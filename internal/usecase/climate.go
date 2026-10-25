package usecase

import (
	"cmp"
	"slices"
	"vucdiagnostic/internal/domain/model"
)

// The climate of the platoon by the express method of А. Н. Лутошкин
// (Вики/Тестирование/Лутошкин — адаптация для ВУЦ.md and decisions Д5, Д6 of
// План П2, П5 и климат по Лутошкину.md, approved on 25 October 2026). The
// student rates his platoon; the program sums the ratings and finds the sides
// of the climate rated strongest and weakest. There are no levels of the
// climate: no key of them is agreed on.

// ClimatePairScore is one pair: the side of the climate it is about, its two
// statements and the student's rating from +3 to −3, nil without an answer.
type ClimatePairScore struct {
	ID       string `json:"id"`
	Side     string `json:"side"`
	Text     string `json:"text"`
	Opposite string `json:"opposite"`
	Score    *int   `json:"score"`
	Answer   string `json:"answer,omitempty"`
}

// Climate is the student's rating of the climate of his platoon. Sum,
// Strongest and Weakest are counted only when all the pairs are answered:
// Strongest are the sides of his highest rating, Weakest those of his lowest;
// Equal says all the ratings are the same, and then there are no sides.
type Climate struct {
	Method    string             `json:"method"`
	Pairs     []ClimatePairScore `json:"pairs"`
	Sum       *int               `json:"sum"`
	Min       int                `json:"min"`
	Max       int                `json:"max"`
	Answered  int                `json:"answered"`
	Total     int                `json:"total"`
	Strongest []string           `json:"strongest"`
	Weakest   []string           `json:"weakest"`
	Equal     bool               `json:"equal"`
}

// Complete says the climate has a sum: every pair is answered.
func (c *Climate) Complete() bool { return c != nil && c.Sum != nil }

// climateMethod is that of the session's bank version; a session of an older
// template that got the methods added at assignment uses the current one.
func (a *Service) climateMethod(engine *Service) *model.ClimateMethod {
	if engine.bank.Climate != nil {
		return engine.bank.Climate
	}
	return a.bank.Climate
}

// climate counts the student's rating; nil when the method was not given.
func (a *Service) climate(engine *Service, s model.Session) *Climate {
	method := a.climateMethod(engine)
	if method == nil {
		return nil
	}
	asked := map[string]bool{}
	for _, item := range engine.itemsFor(s) {
		asked[item.ID] = true
	}
	c := &Climate{Method: method.Method, Pairs: []ClimatePairScore{}, Strongest: []string{}, Weakest: []string{}}
	sum := 0
	for _, pair := range method.Pairs {
		if !asked[pair.Item] {
			continue
		}
		item, option := answeredOption(engine, s, pair.Item)
		row := ClimatePairScore{ID: pair.Item, Side: pair.Side, Text: item.Text, Opposite: item.Opposite}
		top, bottom := 0, 0
		for _, o := range item.Options {
			top, bottom = max(top, int(o.Score)), min(bottom, int(o.Score))
		}
		c.Max, c.Min = c.Max+top, c.Min+bottom
		if option != nil {
			score := int(option.Score)
			row.Score, row.Answer = &score, option.Text
			sum += score
			c.Answered++
		}
		c.Pairs = append(c.Pairs, row)
	}
	c.Total = len(c.Pairs)
	if c.Total == 0 {
		return nil
	}
	if c.Answered < c.Total {
		return c
	}
	c.Sum = &sum
	highest, lowest := *c.Pairs[0].Score, *c.Pairs[0].Score
	for _, p := range c.Pairs {
		highest, lowest = max(highest, *p.Score), min(lowest, *p.Score)
	}
	if c.Equal = highest == lowest; c.Equal {
		return c
	}
	for _, p := range c.Pairs {
		switch *p.Score {
		case highest:
			c.Strongest = append(c.Strongest, p.Side)
		case lowest:
			c.Weakest = append(c.Weakest, p.Side)
		}
	}
	return c
}

// PlatoonClimatePair is a pair over the platoon: the mean rating of the
// students who answered all the pairs.
type PlatoonClimatePair struct {
	ID       string   `json:"id"`
	Side     string   `json:"side"`
	Text     string   `json:"text"`
	Opposite string   `json:"opposite"`
	Mean     *float64 `json:"mean"`
}

// PlatoonClimate is the climate of a platoon by the students with all the
// pairs answered (Rated): the mean of their sums and of every pair, the three
// sides with the highest and the three with the lowest mean, all of a mean
// equal at the border, a side in one list only. Equal says all the means are
// the same; then there are no sides.
type PlatoonClimate struct {
	Method    string               `json:"method"`
	Rated     []MethodsStudent     `json:"rated"`
	Sum       *float64             `json:"sum"`
	Min       int                  `json:"min"`
	Max       int                  `json:"max"`
	Pairs     []PlatoonClimatePair `json:"pairs"`
	Strongest []string             `json:"strongest"`
	Weakest   []string             `json:"weakest"`
	Equal     bool                 `json:"equal"`
}

// climateSides is how many sides each list takes before the ties at the border.
const climateSides = 3

// platoonClimateMinimum is the fewest students with all the pairs answered the
// climate of a platoon is counted of (decided by the user on 25 October 2026).
const platoonClimateMinimum = 3

// platoonClimate counts the climate of a platoon; nil when fewer than
// platoonClimateMinimum students answered all the pairs.
func platoonClimate(portraits []studentPortrait) *PlatoonClimate {
	var c *PlatoonClimate
	sums := []float64{}
	ratings := map[string][]float64{}
	for _, p := range portraits {
		own := p.data.Climate
		if !own.Complete() {
			continue
		}
		if c == nil {
			c = &PlatoonClimate{Method: own.Method, Rated: []MethodsStudent{}, Min: own.Min, Max: own.Max, Pairs: []PlatoonClimatePair{}, Strongest: []string{}, Weakest: []string{}}
			for _, pair := range own.Pairs {
				c.Pairs = append(c.Pairs, PlatoonClimatePair{ID: pair.ID, Side: pair.Side, Text: pair.Text, Opposite: pair.Opposite})
			}
		}
		c.Rated = append(c.Rated, p.student)
		sums = append(sums, float64(*own.Sum))
		for _, pair := range own.Pairs {
			ratings[pair.ID] = append(ratings[pair.ID], float64(*pair.Score))
		}
	}
	if c == nil || len(c.Rated) < platoonClimateMinimum {
		return nil
	}
	c.Sum = meanPointer(sums, 10)
	for n := range c.Pairs {
		c.Pairs[n].Mean = meanPointer(ratings[c.Pairs[n].ID], 100)
	}
	order := slices.Clone(c.Pairs)
	slices.SortStableFunc(order, func(x, y PlatoonClimatePair) int { return cmp.Compare(*y.Mean, *x.Mean) })
	if c.Equal = *order[0].Mean == *order[len(order)-1].Mean; c.Equal {
		return c
	}
	top := *order[min(climateSides, len(order))-1].Mean
	bottom := *order[max(len(order)-climateSides, 0)].Mean
	strongest := map[string]bool{}
	for _, p := range order {
		if *p.Mean >= top {
			c.Strongest = append(c.Strongest, p.Side)
			strongest[p.ID] = true
		}
	}
	// The weakest from the lowest mean up; equal means in the order of the method.
	slices.SortStableFunc(order, func(x, y PlatoonClimatePair) int { return cmp.Compare(*x.Mean, *y.Mean) })
	for _, p := range order {
		if *p.Mean <= bottom && !strongest[p.ID] {
			c.Weakest = append(c.Weakest, p.Side)
		}
	}
	return c
}
