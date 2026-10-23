package usecase

import (
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// Case forms of surnames, names, patronymics and ranks by the rules of Russian
// declension. Words are lowercased with «ё» as «е» (normalWord). The rules
// cover the usual endings; a rare word may lack a form and then stays as
// written in the text.

func hasSuffix(w string, endings ...string) bool {
	for _, ending := range endings {
		if strings.HasSuffix(w, ending) {
			return true
		}
	}
	return false
}

// cut drops the last n letters.
func cut(w string, n int) string {
	r := []rune(w)
	if len(r) <= n {
		return w
	}
	return string(r[:len(r)-n])
}

func withEndings(stem string, endings ...string) []string {
	forms := make([]string, len(endings))
	for i, ending := range endings {
		forms[i] = stem + ending
	}
	return forms
}

// Nouns in -а: Никита, Зима, старшина. After г, к, х and hissing letters the
// genitive ends in -и (Ольги).
func aEndings(stem string) []string {
	genitive := "ы"
	if hasSuffix(stem, "г", "к", "х", "ж", "ш", "ч", "щ") {
		genitive = "и"
	}
	return withEndings(stem, "а", genitive, "е", "у", "ой", "ою")
}

func yaEndings(stem string) []string {
	return withEndings(stem, "я", "и", "е", "ю", "ей", "ею")
}

// Masculine nouns ending in a consonant: Иван, Ковальчук, полковник.
func consonantEndings(w string) []string {
	forms := withEndings(w, "", "а", "у", "ом", "е")
	if hasSuffix(w, "ж", "ш", "ч", "щ", "ц") {
		forms = append(forms, w+"ем")
	}
	return forms
}

// wordForms declines every part of a hyphenated word and joins the parts.
func wordForms(w string, forms func(string) []string) map[string]bool {
	set := map[string]bool{}
	if w == "" {
		return set
	}
	joined := []string{""}
	for i, part := range strings.Split(w, "-") {
		next := []string{}
		for _, head := range joined {
			for _, form := range forms(part) {
				if i > 0 {
					form = head + "-" + form
				}
				next = append(next, form)
			}
		}
		joined = next
	}
	for _, form := range joined {
		set[form] = true
	}
	return set
}

func surnameForms(w string, female bool) []string {
	switch {
	case hasSuffix(w, "кий", "гий", "хий", "кой", "гой", "хой"):
		return withEndings(cut(w, 2), string([]rune(w)[len([]rune(w))-2:]), "ого", "ому", "им", "ом")
	case hasSuffix(w, "ый", "ой"):
		return withEndings(cut(w, 2), string([]rune(w)[len([]rune(w))-2:]), "ого", "ому", "ым", "ом")
	case hasSuffix(w, "ий"):
		return withEndings(cut(w, 2), "ий", "его", "ему", "им", "ем")
	case hasSuffix(w, "ая"):
		return withEndings(cut(w, 2), "ая", "ой", "ую")
	case hasSuffix(w, "яя"):
		return withEndings(cut(w, 2), "яя", "ей", "юю")
	case female && hasSuffix(w, "ова", "ева", "ина", "ына"):
		return withEndings(cut(w, 1), "а", "ой", "у")
	case !female && hasSuffix(w, "ов", "ев", "ин", "ын"):
		return withEndings(w, "", "а", "у", "ым", "е")
	case hasSuffix(w, "о", "е", "и", "у", "ю", "э", "ы", "ых", "их"):
		return []string{w}
	case hasSuffix(w, "а"):
		return aEndings(cut(w, 1))
	case hasSuffix(w, "я"):
		return yaEndings(cut(w, 1))
	case female:
		return []string{w}
	case hasSuffix(w, "ь"):
		return withEndings(cut(w, 1), "ь", "я", "ю", "ем", "е")
	case hasSuffix(w, "й"):
		return withEndings(cut(w, 1), "й", "я", "ю", "ем", "е")
	}
	return consonantEndings(w)
}

// Names with a vowel dropped in other cases.
var nameExceptions = map[string][]string{
	"павел":    {"павел", "павла", "павлу", "павлом", "павле"},
	"лев":      {"лев", "льва", "льву", "львом", "льве"},
	"любовь":   {"любовь", "любови", "любовью"},
	"пантелей": {"пантелей", "пантелея", "пантелею", "пантелеем", "пантелее"},
}

func nameForms(w string, female bool) []string {
	if forms, ok := nameExceptions[w]; ok {
		return forms
	}
	switch {
	case hasSuffix(w, "ия"):
		return withEndings(cut(w, 1), "я", "и", "ю", "ей", "ею")
	case hasSuffix(w, "ий"):
		return withEndings(cut(w, 1), "й", "я", "ю", "ем", "и")
	case hasSuffix(w, "а"):
		return aEndings(cut(w, 1))
	case hasSuffix(w, "я"):
		return yaEndings(cut(w, 1))
	case hasSuffix(w, "ь") && female:
		return withEndings(cut(w, 1), "ь", "и", "ью")
	case hasSuffix(w, "ь"):
		return withEndings(cut(w, 1), "ь", "я", "ю", "ем", "е")
	case hasSuffix(w, "й"):
		return withEndings(cut(w, 1), "й", "я", "ю", "ем", "е")
	case female:
		return []string{w}
	}
	return consonantEndings(w)
}

func patronymicForms(w string) []string {
	switch {
	case hasSuffix(w, "ич"):
		return withEndings(w, "", "а", "у", "ем", "ом", "е")
	case hasSuffix(w, "на"):
		return aEndings(cut(w, 1))
	}
	return []string{w}
}

// Male names ending in -а or -я.
var maleNamesInA = map[string]bool{"никита": true, "илья": true, "кузьма": true, "фома": true, "лука": true,
	"савва": true, "данила": true, "гаврила": true, "иона": true, "добрыня": true}

// femaleInstructor decides by the patronymic, or by the name without one:
// female surnames in -ова or consonant decline differently.
func femaleInstructor(i model.Instructor) bool {
	if p := normalWord(i.Patronymic); p != "" {
		return hasSuffix(p, "на", "кызы")
	}
	name := normalWord(i.Name)
	return !maleNamesInA[name] && (hasSuffix(name, "а", "я") || name == "любовь")
}

// Ranks a text may name before the person, longest first so that «старший
// лейтенант» wins over «лейтенант».
var rankWords = []string{"генерал армии", "генерал-полковник", "генерал-лейтенант", "генерал-майор",
	"старший лейтенант", "младший лейтенант", "старший прапорщик", "старший сержант", "младший сержант",
	"старший мичман", "старший матрос", "капитан-лейтенант",
	"генерал", "полковник", "подполковник", "майор", "капитан", "лейтенант", "прапорщик", "мичман",
	"старшина", "сержант", "ефрейтор", "матрос", "рядовой"}

func rankWordForms(w string) []string {
	switch {
	case w == "армии":
		return []string{w}
	case hasSuffix(w, "ший"):
		return withEndings(cut(w, 2), "ий", "его", "ему", "им", "ем")
	case hasSuffix(w, "ой"):
		return withEndings(cut(w, 2), "ой", "ого", "ому", "ым", "ом")
	case hasSuffix(w, "а"):
		return aEndings(cut(w, 1))
	}
	return consonantEndings(w)
}

type rankForms struct {
	words []string
	forms []map[string]bool
}

var ranks = func() []rankForms {
	list := []rankForms{}
	for _, rank := range rankWords {
		r := rankForms{words: strings.Fields(rank)}
		for _, word := range r.words {
			r.forms = append(r.forms, wordForms(word, rankWordForms))
		}
		list = append(list, r)
	}
	return list
}()

var comrade = wordForms("товарищ", consonantEndings)

// rankAt reads a rank starting at token k and returns its words in the
// nominative and the token after it; without a rank it returns k.
func rankAt(tokens []textWord, k int) ([]string, int) {
	for _, rank := range ranks {
		fits := k+len(rank.words) <= len(tokens)
		for j := 0; fits && j < len(rank.words); j++ {
			t := tokens[k+j]
			fits = !t.initial && (j == 0 || t.joined) && rank.forms[j][t.word]
		}
		if fits {
			return rank.words, k + len(rank.words)
		}
	}
	return nil, k
}
