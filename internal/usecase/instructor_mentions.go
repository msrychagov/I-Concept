package usecase

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
	"vucdiagnostic/internal/domain/model"
)

// Mentions of listed instructors are replaced with labels before a text goes
// to a language model: one person gets one label whatever the author called
// them, and names from the list never leave the program.
//
// A mention is a run of adjacent words that all fit one instructor: rank (with
// an optional «товарищ»), name, patronymic, surname, initials. It counts when
// it has the surname, or the name with the patronymic; a name or a rank alone
// is left as written. Case does not matter (decided by the user on October 23:
// a surname equal to a common word is replaced in plain text too). When a
// mention fits several entries it is left as written, unless the rank in the
// text leaves exactly one of them.

// InstructorLabel ties a label in masked texts to an entry of the list.
type InstructorLabel struct {
	Number     int              `json:"number"`
	Label      string           `json:"label"`
	Instructor model.Instructor `json:"instructor"`
}

func instructorLabel(number int) string {
	return fmt.Sprintf("[преподаватель %d]", number)
}

// MaskInstructors masks the texts of one author together, so that an
// instructor has the same label in every text. Labels are numbered in order
// of first appearance.
func MaskInstructors(list []model.Instructor, texts []string) ([]string, []InstructorLabel) {
	people := make([]personForms, len(list))
	for i, instructor := range list {
		people[i] = newPersonForms(instructor)
	}
	numbers := map[int]int{}
	labels := []InstructorLabel{}
	masked := make([]string, len(texts))
	for t, text := range texts {
		tokens := tokenize(text)
		var out strings.Builder
		written, i := 0, 0
		for i < len(tokens) {
			person, end := mentionAt(tokens, i, people)
			if person < 0 {
				i++
				continue
			}
			number, seen := numbers[person]
			if !seen {
				number = len(labels) + 1
				numbers[person] = number
				labels = append(labels, InstructorLabel{number, instructorLabel(number), list[person]})
			}
			out.WriteString(text[written:tokens[i].start])
			out.WriteString(instructorLabel(number))
			written = tokens[end-1].end
			i = end
		}
		out.WriteString(text[written:])
		masked[t] = out.String()
	}
	return masked, labels
}

type textWord struct {
	start, end int // bytes of the text; an initial includes its dot
	word       string
	initial    bool // a capital letter followed by a dot
	// joined: only spaces separate the word from the previous one, so
	// «Иванов, Петров» are two mentions and «Иванов Игорь» is one.
	joined bool
}

func isHyphen(r rune) bool { return r == '-' || r == '‐' }

// tokenize splits a text into words: runs of letters, with hyphens inside
// («генерал-майор», «Римский-Корсаков»).
func tokenize(text string) []textWord {
	tokens := []textWord{}
	previous := 0
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if !unicode.IsLetter(r) {
			i += size
			continue
		}
		start, end := i, i+size
		for end < len(text) {
			next, n := utf8.DecodeRuneInString(text[end:])
			if unicode.IsLetter(next) {
				end += n
				continue
			}
			if after, m := utf8.DecodeRuneInString(text[end+n:]); isHyphen(next) && m > 0 && unicode.IsLetter(after) {
				end += n
				continue
			}
			break
		}
		original := text[start:end]
		word := strings.Map(func(r rune) rune {
			if isHyphen(r) {
				return '-'
			}
			return r
		}, normalWord(original))
		t := textWord{start: start, end: end, word: word, joined: len(tokens) > 0 && strings.TrimSpace(text[previous:start]) == ""}
		if utf8.RuneCountInString(original) == 1 && unicode.IsUpper(r) && end < len(text) && text[end] == '.' {
			t.initial, t.end = true, end+1
		}
		tokens = append(tokens, t)
		previous, i = t.end, t.end
	}
	return tokens
}

const (
	roleSurname = 1 << iota
	roleName
	rolePatronymic
	roleInitial
)

type personForms struct {
	surname, name, patronymic map[string]bool
	initials                  map[string]bool
	rank                      []string
}

func newPersonForms(i model.Instructor) personForms {
	female := femaleInstructor(i)
	p := personForms{
		surname:    wordForms(normalWord(i.Surname), func(w string) []string { return surnameForms(w, female) }),
		name:       wordForms(normalWord(i.Name), func(w string) []string { return nameForms(w, female) }),
		patronymic: wordForms(normalWord(i.Patronymic), patronymicForms),
		initials:   map[string]bool{},
		rank:       strings.Fields(normalWord(i.Rank)),
	}
	for _, part := range []string{i.Name, i.Patronymic} {
		if part != "" {
			r, _ := utf8.DecodeRuneInString(normalWord(part))
			p.initials[string(r)] = true
		}
	}
	return p
}

func (p personForms) roles(t textWord) int {
	if t.initial {
		if p.initials[t.word] {
			return roleInitial
		}
		return 0
	}
	roles := 0
	if p.surname[t.word] {
		roles |= roleSurname
	}
	if p.name[t.word] {
		roles |= roleName
	}
	if p.patronymic[t.word] {
		roles |= rolePatronymic
	}
	return roles
}

// mentionAt finds a mention starting at token i and returns the instructor and
// the token after the mention, or -1.
func mentionAt(tokens []textWord, i int, people []personForms) (int, int) {
	k := i
	if comrade[tokens[k].word] && k+1 < len(tokens) && tokens[k+1].joined {
		k++
	}
	var rank []string
	if words, next := rankAt(tokens, k); next > k {
		rank, k = words, next
	}
	if k > i && (k >= len(tokens) || !tokens[k].joined) {
		return -1, 0
	}
	candidates := make([]int, len(people))
	for x := range people {
		candidates[x] = x
	}
	roles := make([]int, len(people))
	end := k
	for q := k; q < len(tokens) && (q == k || tokens[q].joined); q++ {
		next := []int{}
		for _, x := range candidates {
			if people[x].roles(tokens[q]) != 0 {
				next = append(next, x)
			}
		}
		if len(next) == 0 {
			break
		}
		for _, x := range next {
			roles[x] |= people[x].roles(tokens[q])
		}
		candidates, end = next, q+1
	}
	found := []int{}
	if end > k {
		for _, x := range candidates {
			if roles[x]&roleSurname != 0 || roles[x]&(roleName|rolePatronymic) == roleName|rolePatronymic {
				found = append(found, x)
			}
		}
	}
	if len(found) > 1 && rank != nil {
		ranked := []int{}
		for _, x := range found {
			if startsWith(people[x].rank, rank) {
				ranked = append(ranked, x)
			}
		}
		found = ranked
	}
	if len(found) != 1 {
		return -1, 0
	}
	return found[0], end
}

func startsWith(words, prefix []string) bool {
	if len(prefix) > len(words) {
		return false
	}
	for i := range prefix {
		if words[i] != prefix[i] {
			return false
		}
	}
	return true
}
