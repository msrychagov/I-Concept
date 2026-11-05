package usecase

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// The terms of the activities in the first paragraph of v5 (checks added after
// the run of 6 November 2026, decided by the user): a number of years next to
// an activity must fall within the term the student chose in the questionnaire.
// «Год играл в футбол» when the student plays 3–5 years is a made-up fact
// (prohibition 1). A clause without an activity, the year of graduation
// («в 2025 году») and the service are not checked. The copy is in compare.py.

// termV5 is an interval of years, the upper end may be infinite.
type termV5 struct{ low, high float64 }

func (t termV5) overlaps(o termV5) bool { return t.low <= o.high && t.high >= o.low }

// activityTermV5 is an activity of the input with its words and its term.
type activityTermV5 struct {
	name  string
	stems []string
	term  termV5
	text  string
}

// numberWordsV5 are the numbers written in words.
var numberWordsV5 = map[string]float64{
	"один": 1, "одного": 1, "полтора": 1.5, "полутора": 1.5, "два": 2, "двух": 2, "три": 3, "трёх": 3, "трех": 3,
	"четыре": 4, "четырёх": 4, "четырех": 4, "пять": 5, "пяти": 5, "шесть": 6, "шести": 6, "семь": 7, "семи": 7,
	"восемь": 8, "восьми": 8, "девять": 9, "девяти": 9, "десять": 10, "десяти": 10,
}

const numberV5 = `\d+(?:[.,]\d+)?|один|одного|полтора|полутора|два|двух|три|трёх|трех|четыре|четырёх|четырех|пять|пяти|шесть|шести|семь|семи|восемь|восьми|девять|девяти|десять|десяти`

// durationV5 is a number of years in a text: «год», «три года», «больше пяти
// лет», «три–пять лет», «меньше года». «Году» is a date and is not taken.
var durationV5 = regexp.MustCompile(`(?i)(?:^|[^а-яё0-9])(?:(?P<mod>больше|более|свыше|около|почти|менее|меньше|не менее|до)\s+)?(?:(?P<from>` + numberV5 + `)(?:\s*[–—-]\s*(?P<to>` + numberV5 + `))?\s+)?(?P<unit>год|года|лет)(?:[^а-яё]|$)`)

// numberV5Value reads a number of durationV5.
func numberV5Value(text string) (float64, bool) {
	text = strings.ToLower(text)
	if n, ok := numberWordsV5[text]; ok {
		return n, true
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(text, ",", "."), 64)
	return n, err == nil
}

// durationsV5 are the numbers of years told in a clause; a year of a date
// («2025 года») is no duration.
func durationsV5(clause string) []struct {
	text string
	term termV5
} {
	found := []struct {
		text string
		term termV5
	}{}
	names := durationV5.SubexpNames()
	for _, match := range durationV5.FindAllStringSubmatch(clause, -1) {
		group := map[string]string{}
		for n, name := range names {
			if name != "" {
				group[name] = match[n]
			}
		}
		from, to := 1.0, 0.0
		if group["from"] != "" {
			n, ok := numberV5Value(group["from"])
			if !ok || n >= 1900 {
				continue
			}
			from = n
		} else if strings.EqualFold(group["unit"], "лет") {
			continue // «много лет»: no number
		}
		to = from
		if group["to"] != "" {
			if n, ok := numberV5Value(group["to"]); ok {
				to = n
			}
		}
		term := termV5{from, to}
		switch strings.ToLower(group["mod"]) {
		case "больше", "более", "свыше":
			term = termV5{to + 0.5, math.Inf(1)}
		case "не менее":
			term = termV5{from, math.Inf(1)}
		case "менее", "меньше", "до":
			term = termV5{0, to}
		case "около":
			term = termV5{from - 1, to + 1}
		case "почти":
			term = termV5{from - 1, to}
		}
		found = append(found, struct {
			text string
			term termV5
		}{strings.Trim(strings.ToLower(match[0]), " ,.;:()—"), term})
	}
	return found
}

// answerTermV5 reads a term of the questionnaire: «До года», «1–3 года»,
// «3–5 лет», «Больше 5 лет», «Разово».
func answerTermV5(text string) (termV5, bool) {
	lower := strings.ToLower(strings.TrimSpace(text))
	switch {
	case lower == "":
		return termV5{}, false
	case lower == "разово":
		return termV5{0, 0.5}, true
	case strings.HasPrefix(lower, "до года"):
		return termV5{0, 1}, true
	}
	numbers := []float64{}
	for _, part := range strings.FieldsFunc(lower, func(r rune) bool { return !unicode.IsDigit(r) }) {
		if n, err := strconv.ParseFloat(part, 64); err == nil {
			numbers = append(numbers, n)
		}
	}
	switch {
	case strings.HasPrefix(lower, "больше") && len(numbers) == 1:
		return termV5{numbers[0], math.Inf(1)}, true
	case len(numbers) == 2:
		return termV5{numbers[0], numbers[1]}, true
	}
	return termV5{}, false
}

// genericSportWordsV5 are the words of a kind of sport that tell nothing of it.
var genericSportWordsV5 = map[string]bool{"спорт": true, "спортивная": true, "спортивный": true, "игры": true, "другое": true}

// patrioticStemsV5 find the military-patriotic activity in a clause.
var patrioticStemsV5 = []string{"юнарм", "зарниц", "движени", "кадетск", "клуб", "сбор", "досааф", "поиск", "вахт", "караул", "спартакиад", "реконструкц", "военно-патриот", "военно-спортивн"}

// activityTermsV5 are the activities of the input that have a term.
func activityTermsV5(payload *object) []activityTermV5 {
	activities := []activityTermV5{}
	if sport, ok := payload.values["спорт"].(*object); ok {
		for _, record := range recordsV5(sport.values["занятия"]) {
			kind := fieldV5(record, "вид")
			term, ok := answerTermV5(fieldV5(record, "стаж"))
			if kind == "" || !ok {
				continue
			}
			stems := []string{}
			for _, word := range strings.FieldsFunc(strings.ToLower(kind), func(r rune) bool { return !unicode.IsLetter(r) }) {
				letters := []rune(word)
				if len(letters) < 4 || genericSportWordsV5[word] {
					continue
				}
				if len(letters) > 5 {
					letters = letters[:max(5, len(letters)-2)]
				}
				stems = append(stems, string(letters))
			}
			if len(stems) > 0 {
				activities = append(activities, activityTermV5{strings.ToLower(kind), stems, term, strings.ToLower(fieldV5(record, "стаж"))})
			}
		}
	}
	if help, ok := payload.values["волонтёрство"].(*object); ok {
		if term, ok := answerTermV5(fieldV5(help, "длительность")); ok {
			activities = append(activities, activityTermV5{"волонтёрство", []string{"волонт", "донор"}, term, strings.ToLower(fieldV5(help, "длительность"))})
		}
	}
	if patriot, ok := payload.values["военно_патриотическое"].(*object); ok {
		if term, ok := answerTermV5(fieldV5(patriot, "длительность")); ok {
			activities = append(activities, activityTermV5{"военно-патриотическое", patrioticStemsV5, term, strings.ToLower(fieldV5(patriot, "длительность"))})
		}
	}
	return activities
}

// termProblemsV5 checks the terms in the first paragraph: in a clause that
// names an activity, every number of years fits the term of one of the
// activities it names.
func termProblemsV5(where, text string, activities []activityTermV5) []string {
	problems := []string{}
	for _, clause := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return strings.ContainsRune(".;,:()", r) }) {
		named := []activityTermV5{}
		for _, activity := range activities {
			for _, stem := range activity.stems {
				if strings.Contains(clause, stem) {
					named = append(named, activity)
					break
				}
			}
		}
		if len(named) == 0 {
			continue
		}
		for _, duration := range durationsV5(clause) {
			fits := false
			for _, activity := range named {
				fits = fits || duration.term.overlaps(activity.term)
			}
			if !fits {
				problems = append(problems, fmt.Sprintf("%s: срок не тот («%s» — в анкете «%s»: %s) — пиши срок, как в анкете", where, duration.text, named[0].name, named[0].text))
			}
		}
	}
	return problems
}
