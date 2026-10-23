package usecase

import (
	"slices"
	"strings"
	"unicode"
	"vucdiagnostic/internal/domain/model"
)

// Namesakes among the instructors. The model matches a mention with the list,
// but when two instructors share a surname, or a name and a patronymic, and the
// student wrote only the part they share («Кузнецов», «Игорь Петрович»), the
// choice is a guess and a teacher would read a stranger's review under a name.
// The program then takes the label off (decided 23 October 2026): the entry
// stays, written as the student wrote, with the labels of those who fit.

// namesakesOf are the other instructors of the center with the same surname or
// with the same name and patronymic (a missing patronymic is no match).
func namesakesOf(db model.Database, vuc string, self model.Instructor) []model.Instructor {
	same := func(a, b string) bool { return normalWord(a) == normalWord(b) }
	list := []model.Instructor{}
	for _, other := range centerInstructors(db, vuc) {
		if other.ID == self.ID {
			continue
		}
		if same(other.Surname, self.Surname) || self.Patronymic != "" && same(other.Name, self.Name) && same(other.Patronymic, self.Patronymic) {
			list = append(list, other)
		}
	}
	return list
}

// caseEndings are what may stand after the stem of a name, a surname or a rank
// in the cases of Russian: «Кузнецов» is found in «Кузнецову», «Игорь» in
// «Игоря», «старший» in «старшего». «ич» is none of them, so «Петрович» is not
// «Петров» and «Ильича» is not «Илья». It is not a morphology, only enough for
// the names of instructors.
var caseEndings = []string{"", "а", "я", "у", "ю", "е", "ы", "и", "о", "ь", "й", "ом", "ем", "ём", "ым", "им", "ой", "ей", "ою", "ею", "ых", "их", "ую", "юю", "ий", "ый", "его", "ого", "ему", "ому", "ами", "ями", "ов", "ев", "ёв"}

// wordStem cuts the ending a word may change: the last letter if it is a vowel,
// «ь» or «й», two letters of «ий», «ый», «ой».
func wordStem(word string) []rune {
	runes := []rune(normalWord(word))
	n := len(runes)
	if n >= 6 {
		switch string(runes[n-2:]) {
		case "ий", "ый", "ой":
			return runes[:n-2]
		}
	}
	if n >= 4 && strings.ContainsRune("аяьйыиуюеоэ", runes[n-1]) {
		return runes[:n-1]
	}
	return runes
}

func isWordOf(token, attribute string) bool {
	stem := string(wordStem(attribute))
	ending, found := strings.CutPrefix(token, stem)
	return found && slices.Contains(caseEndings, ending)
}

// tokens splits a text into its words, with the form the words are compared in.
func tokens(text string) (written, normal []string) {
	for _, word := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) }) {
		written = append(written, word)
		normal = append(normal, normalWord(word))
	}
	return written, normal
}

// attributesOf are the four things a student may call an instructor by.
func attributesOf(i model.Instructor) map[string]string {
	return map[string]string{"surname": i.Surname, "name": i.Name, "patronymic": i.Patronymic, "rank": i.Rank}
}

// foundIn says which attributes of the instructor are in the texts, and the
// words as the student wrote them, in the order they come.
func foundIn(i model.Instructor, texts []string) (map[string]bool, []string) {
	found := map[string]bool{}
	words := []string{}
	for _, text := range texts {
		written, normal := tokens(text)
		for kind, value := range attributesOf(i) {
			parts := strings.Fields(value)
			if len(parts) == 0 {
				continue
			}
			every := true
			var hits []int
			for _, part := range parts {
				at := slices.IndexFunc(normal, func(token string) bool { return isWordOf(token, part) })
				if at < 0 {
					every = false
					break
				}
				hits = append(hits, at)
			}
			if !every {
				continue
			}
			found[kind] = true
			for _, at := range hits {
				if !slices.Contains(words, written[at]) {
					words = append(words, written[at])
				}
			}
		}
	}
	return found, words
}

// cleanEssayAnswer takes the label off the entries about instructors that the
// student's words do not tell from a namesake.
func cleanEssayAnswer(a *Service, db model.Database, s Source, output map[string]any, payload *object) (map[string]any, int) {
	submission, ok := findSubmission(db, s)
	student, known := studentOf(db, submission)
	teachers, listed := asList(output["teachers"])
	if !ok || !known || !listed {
		return output, 0
	}
	answers := map[string]string{}
	for _, row := range objectList(payload.values["эссе"]) {
		answers[objectText(row, "номер")] = objectText(row, "ответ")
	}
	changes := 0
	for _, raw := range teachers {
		record := asMap(raw)
		label := textOf(record["who"])
		self, ok := instructorByLabel(db, student.VUC, label)
		if record == nil || !ok {
			continue
		}
		namesakes := namesakesOf(db, student.VUC, self)
		if len(namesakes) == 0 {
			continue
		}
		texts := []string{}
		for _, quote := range quoteList(record["quotes"]) {
			if text, has := answers[textOf(quote["question"])]; has && !slices.Contains(texts, text) {
				texts = append(texts, text)
			}
		}
		found, words := foundIn(self, texts)
		if len(found) == 0 {
			// A typo or an abbreviation: nothing to check, the model's word stays.
			continue
		}
		candidates := []any{label}
		for _, other := range namesakes {
			distinct := false
			for kind, value := range attributesOf(other) {
				if found[kind] && normalWord(value) != normalWord(attributesOf(self)[kind]) {
					distinct = true
				}
			}
			if !distinct {
				candidates = append(candidates, instructorLabel(other))
			}
		}
		if len(candidates) == 1 {
			continue
		}
		record["who"] = strings.Join(words, " ")
		record["candidates"] = candidates
		changes++
	}
	return output, changes
}

func quoteList(raw any) []map[string]any {
	list, _ := asList(raw)
	quotes := []map[string]any{}
	for _, item := range list {
		if quote := asMap(item); quote != nil {
			quotes = append(quotes, quote)
		}
	}
	return quotes
}
