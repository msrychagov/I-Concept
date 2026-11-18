package usecase

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
	"vucdiagnostic/internal/formbank"
)

// The people an author names in an essay do not reach the model (decided 13
// November 2026, Вики/Эссе выпускника/План обезличивания.md): every mention of
// a person — first name, patronymic, surname, initials — is replaced by a label
// «[человек N]», one number for one person in any case form, and the program
// puts the words of the author back before the teacher sees the answer. The
// table of labels is not stored: it is built anew from the same texts by the
// same rules and comes out the same.
//
// The people are found by rules, not by a list of the teachers (the head said
// on 25 October «без списка фамилий»): a word from the list of first names, a
// patronymic, an initial; a surname after a rank or a position, next to a first
// name, with a surname ending inside a sentence, or found so elsewhere in the
// same texts. A rare surname standing alone at the start of a sentence is missed.

// personLabel is the label of the person with the number.
func personLabel(n int) string { return "[человек " + strconv.Itoa(n) + "]" }

// personLabels finds a label in a text; personLoose finds one the model wrote
// without brackets.
var (
	personLabels = regexp.MustCompile(`\[человек (\d+)\]`)
	personLoose  = regexp.MustCompile(`(?i)\[?\s*человек\s+\d+\s*\]?`)
)

// peoplePiece is a piece of a text: shown is what goes to the model, orig what
// the author wrote. person is the number of the person a piece stands for, 0
// for a piece left as it is and -1 for personal data cut out.
type peoplePiece struct {
	shown, orig string
	person      int
}

type peopleText []peoplePiece

func (t peopleText) shown() string {
	var out strings.Builder
	for _, p := range t {
		out.WriteString(p.shown)
	}
	return out.String()
}

// markedPerson is one person of the texts: the key the mentions are joined by,
// the surnames as written and the mentions as written, in order.
type markedPerson struct {
	surnames []string
	mentions []string
	surname  string
}

// peopleMarks are the texts of a source with the people replaced by labels.
type peopleMarks struct {
	people []*markedPerson
	texts  []peopleText
}

// ---- the words ------------------------------------------------------------

type personWord struct {
	start, end int
	text       string
	// lower is the word in lower case with «ё» as «е».
	lower string
}

// personWords splits a text into words of letters, with a hyphen inside a word
// («Петров-Водкин») kept. An initial is a word of one capital with its dot.
func personWords(text string) []personWord {
	words := []personWord{}
	at := 0
	for at < len(text) {
		r, size := utf8.DecodeRuneInString(text[at:])
		if !unicode.IsLetter(r) {
			at += size
			continue
		}
		start := at
		for at < len(text) {
			r, size = utf8.DecodeRuneInString(text[at:])
			if unicode.IsLetter(r) {
				at += size
				continue
			}
			if r == '-' && at+size < len(text) {
				if next, _ := utf8.DecodeRuneInString(text[at+size:]); unicode.IsLetter(next) {
					at += size
					continue
				}
			}
			break
		}
		end := at
		if utf8.RuneCountInString(text[start:end]) == 1 && end < len(text) && text[end] == '.' {
			end++
		}
		words = append(words, personWord{start, end, text[start:end], normalWord(strings.TrimSuffix(text[start:end], "."))})
		at = end
	}
	return words
}

// capital says whether a word starts with a capital and is not written all in
// capitals («ВУЦ», «ВДВ» are not names).
func (w personWord) capital() bool {
	first, _ := utf8.DecodeRuneInString(w.text)
	if !unicode.IsUpper(first) {
		return false
	}
	letters := []rune(strings.TrimSuffix(w.text, "."))
	if len(letters) == 1 {
		return false
	}
	for _, r := range letters[1:] {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

func (w personWord) initial() bool {
	first, _ := utf8.DecodeRuneInString(w.text)
	return unicode.IsUpper(first) && utf8.RuneCountInString(w.text) == 2 && strings.HasSuffix(w.text, ".")
}

// joined says whether only spaces stand between two words, without a line break.
func joined(text string, left, right personWord) bool {
	gap := text[left.end:right.start]
	return strings.TrimFunc(gap, func(r rune) bool { return r == ' ' || r == '\t' || r == ' ' }) == "" && !strings.Contains(gap, "\n")
}

// sentenceStart says whether a word opens a sentence: before it, through quotes,
// dashes and brackets, stand only the start of the text, a line break or the end
// of a sentence.
func sentenceStart(text string, w personWord) bool {
	before := strings.TrimRightFunc(text[:w.start], func(r rune) bool {
		return r == ' ' || r == '\t' || r == ' ' || strings.ContainsRune("«»\"„“”'(—–-", r)
	})
	if before == "" {
		return true
	}
	last, _ := utf8.DecodeLastRuneInString(before)
	return strings.ContainsRune(".!?…\n\r", last)
}

// ---- what a word is -------------------------------------------------------

var patronymicWord = regexp.MustCompile(`^[а-я]+(?:ович|евич|ьич|овн|евн|ичн|иничн)(?:а|у|ем|ом|е|ы|ой|ою)?$`)

// patronymicBase is a patronymic without its case ending.
var patronymicBase = regexp.MustCompile(`(ович|евич|ьич|овн|евн|ичн)(?:а|у|ем|ом|е|ы|ой)?$`)

// nameEndings are the case endings of a first name on its stem.
var nameEndings = []string{"а", "я", "у", "ю", "ой", "ей", "ою", "ею", "ом", "ем", "е", "и", "ы", "ам", "ям", "ами", "ями", "ах", "ях"}

// fleeting are the first names that lose a vowel in the other cases.
var fleeting = map[string]string{"павел": "павл", "лев": "льв", "петр": "петр", "аким": "аким"}

type nameBook struct {
	// stems maps the stem of a first name to the names with it.
	stems map[string][]string
	names map[string]bool
	words map[string]bool
}

var (
	personBookOnce sync.Once
	personBook     nameBook
)

func loadedNames() nameBook {
	personBookOnce.Do(func() {
		var raw struct{ Names, Words []string }
		_ = json.Unmarshal(formbank.PersonNames(), &raw)
		personBook = nameBook{stems: map[string][]string{}, names: map[string]bool{}, words: map[string]bool{}}
		for _, name := range raw.Names {
			n := normalWord(name)
			personBook.names[n] = true
			stem := n
			if last, size := utf8.DecodeLastRuneInString(n); strings.ContainsRune("аяйьо", last) {
				stem = n[:len(n)-size]
			}
			personBook.stems[stem] = append(personBook.stems[stem], n)
			if short, ok := fleeting[n]; ok && short != stem {
				personBook.stems[short] = append(personBook.stems[short], n)
			}
		}
		for _, word := range raw.Words {
			personBook.words[normalWord(word)] = true
		}
	})
	return personBook
}

// firstName returns the first name a word is a case form of, "" for none.
func (b nameBook) firstName(lower string) string {
	if b.names[lower] {
		return lower
	}
	for _, ending := range nameEndings {
		if !strings.HasSuffix(lower, ending) {
			continue
		}
		stem := strings.TrimSuffix(lower, ending)
		if utf8.RuneCountInString(stem) < 2 {
			continue
		}
		for _, name := range b.stems[stem] {
			return name
		}
	}
	return ""
}

// titleStems are the ranks and positions a surname follows («майор Захаров»,
// «начальнику цикла Петрову»), with up to three letters of a case ending.
var titleStems = []string{"рядов", "ефрейтор", "сержант", "старшин", "прапорщик", "лейтенант", "капитан", "майор", "подполковник", "полковник", "генерал", "преподавател", "начальник", "куратор", "командир", "заместител", "руководител", "наставник", "учител", "тренер", "профессор", "доцент", "декан", "товарищ", "офицер", "педагог", "замполит", "курсант", "одногруппник", "однокурсник", "сослуживц", "сослуживец", "друг", "брат", "дед", "дядя", "дяд"}

// titleTails are the words that end the name of a position before the surname.
var titleTails = []string{"цикла", "кафедры", "взвода", "вуц", "вуца", "центра", "роты", "курса", "батальона", "отделения", "подготовки", "факультета"}

func isTitle(lower string) bool {
	if slices.Contains(titleTails, lower) {
		return true
	}
	for _, stem := range titleStems {
		if strings.HasPrefix(lower, stem) && len([]rune(lower))-len([]rune(stem)) <= 3 {
			return true
		}
	}
	return false
}

// surnameShape finds the endings of a surname that may stand alone: ‑ов, ‑ев,
// ‑ин, ‑ын and their cases, ‑енко, ‑ук, ‑юк, ‑ич, ‑ян, ‑швили, ‑дзе. The endings
// ‑ский and ‑ых are surnames only next to a first name or a rank: «Российской»,
// «Вооружённых» stand capital inside a sentence too.
var surnameShape = regexp.MustCompile(`^[а-я-]{2,}(?:(?:ов|ев|ин|ын)(?:а|у|ым|ыми|е|ой|ою|ых)?|енко|[ую]к(?:а|у|ом|е)?|[^аеиоуыэюя]ич(?:а|у|ем|е)?|ян(?:а|у|ом|е)?|швили|дзе)$`)

// notSurnames are the words with the endings of a surname that an essay writes
// with a capital: «Родине», «Героев Отечества».
var notSurnames = []string{"родин", "отечеств", "геро", "бог", "росси", "москв", "отц", "господ", "христ", "кремл"}

func surnameLike(lower string) bool {
	for _, stem := range notSurnames {
		if strings.HasPrefix(lower, stem) {
			return false
		}
	}
	return surnameShape.MatchString(lower)
}

var (
	surnameFamily = regexp.MustCompile(`^(.+?(?:ов|ев|ин|ын))(?:а|у|ым|ыми|е|ой|ою|ых)?$`)
	surnameSky    = regexp.MustCompile(`^(.+?[сц]к)(?:ий|ая|ого|ому|им|ом|ую|ой|ие|их)$`)
)

// surnameKey is a surname without its case ending: «Захарову», «Захаровым» →
// «захаров».
func surnameKey(lower string) string {
	if m := surnameSky.FindStringSubmatch(lower); m != nil {
		return m[1]
	}
	if m := surnameFamily.FindStringSubmatch(lower); m != nil {
		return m[1]
	}
	for _, ending := range []string{"ом", "ем", "ым", "ой", "ей", "а", "у", "е", "ю", "я", "и", "ы"} {
		if stem := strings.TrimSuffix(lower, ending); stem != lower && utf8.RuneCountInString(stem) >= 3 {
			return stem
		}
	}
	return lower
}

// nominative says whether a surname is written in the form it is named by.
func nominative(lower, key string) bool {
	return lower == key || strings.HasSuffix(lower, "ский") || strings.HasSuffix(lower, "цкий") || strings.HasSuffix(lower, "ая") && strings.HasPrefix(lower, key)
}

// ---- the mentions ---------------------------------------------------------

type personPart int

const (
	partNone personPart = iota
	partName
	partPatronymic
	partInitial
	partSurname
)

type personMention struct {
	start, end int
	words      []personWord
	kinds      []personPart
	names      []string
}

// key is what joins the mentions of one person: the surname without its
// ending, or the first name and the patronymic, or the first name alone.
func (m personMention) key() string {
	if surname := m.surnameWord(); surname >= 0 {
		return "ф:" + surnameKey(m.words[surname].lower)
	}
	return m.nameKey()
}

func (m personMention) nameKey() string {
	name, patronymic := "", ""
	for i, kind := range m.kinds {
		switch kind {
		case partName:
			if name == "" {
				name = m.names[i]
			}
		case partPatronymic:
			patronymic = patronymicBase.ReplaceAllString(m.words[i].lower, "$1")
		}
	}
	if name == "" && patronymic == "" {
		return ""
	}
	return "и:" + name + "|" + patronymic
}

func (m personMention) surnameWord() int {
	for i, kind := range m.kinds {
		if kind == partSurname {
			return i
		}
	}
	return -1
}

// mentions finds the people in a text. known are the keys of surnames found
// elsewhere: such a word is a surname even at the start of a sentence.
func mentions(text string, known map[string]bool) []personMention {
	book := loadedNames()
	words := personWords(text)
	found := []personMention{}
	for i := 0; i < len(words); {
		w := words[i]
		if !w.capital() && !w.initial() {
			i++
			continue
		}
		// A run of capital words and initials with only spaces between them.
		j := i + 1
		for j < len(words) && (words[j].capital() || words[j].initial()) && joined(text, words[j-1], words[j]) {
			j++
		}
		run := words[i:j]
		titled := i > 0 && joined(text, words[i-1], w) && isTitle(words[i-1].lower)
		if m, ok := runMention(text, run, titled, known, book); ok {
			found = append(found, m)
		}
		i = j
	}
	return found
}

// runMention decides which words of a run of capitals name a person.
func runMention(text string, run []personWord, titled bool, known map[string]bool, book nameBook) (personMention, bool) {
	kinds := make([]personPart, len(run))
	names := make([]string, len(run))
	strong := func(k personPart) bool { return k == partName || k == partPatronymic || k == partInitial }
	for n, w := range run {
		switch {
		case w.initial():
			kinds[n] = partInitial
		case patronymicWord.MatchString(w.lower):
			kinds[n] = partPatronymic
		default:
			if name := book.firstName(w.lower); name != "" && !book.words[name] {
				kinds[n], names[n] = partName, name
			}
		}
	}
	// A name that is also a word («Вера», «Слава») is a name only next to
	// another part of a name or a surname-like word.
	for n, w := range run {
		if kinds[n] != partNone {
			continue
		}
		if name := book.firstName(w.lower); name != "" && book.words[name] {
			near := n > 0 && (strong(kinds[n-1]) || surnameLike(run[n-1].lower)) || n+1 < len(run) && (strong(kinds[n+1]) || surnameLike(run[n+1].lower))
			if near {
				kinds[n], names[n] = partName, name
			}
		}
	}
	first, last := -1, -1
	for n := range run {
		if strong(kinds[n]) {
			if first < 0 {
				first = n
			}
			last = n
		}
	}
	if first >= 0 {
		// Before the first part only a word shaped as a surname («Петров Сергей»),
		// after the last any capital word («Сергей Петров»).
		start := first
		for start > 0 && run[start-1].capital() && (surnameLike(run[start-1].lower) || known[surnameKey(run[start-1].lower)] || titled && start-1 == 0) {
			start--
		}
		end := last + 1
		for end < len(run) && run[end].capital() {
			end++
		}
		for n := start; n < end; n++ {
			if kinds[n] == partNone {
				kinds[n] = partSurname
			}
		}
		if !slices.ContainsFunc(kinds[start:end], func(k personPart) bool { return k == partName || k == partPatronymic || k == partSurname }) {
			return personMention{}, false
		}
		return personMention{run[start].start, run[end-1].end, run[start:end], kinds[start:end], names[start:end]}, true
	}
	// No first name: a surname after a rank or a position, a known surname or a
	// surname-shaped word inside a sentence.
	start := -1
	for n, w := range run {
		if !w.capital() {
			continue
		}
		afterTitle := n == 0 && titled || n > 0 && isTitle(run[n-1].lower)
		if afterTitle || known[surnameKey(w.lower)] || surnameLike(w.lower) && !sentenceStart(text, w) {
			start = n
			break
		}
	}
	if start < 0 {
		return personMention{}, false
	}
	end := start + 1
	for end < len(run) && run[end].capital() && (surnameLike(run[end].lower) || known[surnameKey(run[end].lower)]) {
		end++
	}
	for n := start; n < end; n++ {
		kinds[n] = partSurname
	}
	return personMention{run[start].start, run[end-1].end, run[start:end], kinds[start:end], names[start:end]}, true
}

// ---- the labels -----------------------------------------------------------

// markPeople replaces the people in the texts by labels, numbered by their
// first mention over all the texts in order, and cuts the personal data of
// portraits.go out of the rest.
func markPeople(texts []string) *peopleMarks {
	// The first pass finds the surnames, the second takes them everywhere.
	known := map[string]bool{}
	for _, text := range texts {
		for _, m := range mentions(text, nil) {
			if s := m.surnameWord(); s >= 0 {
				if key := surnameKey(m.words[s].lower); utf8.RuneCountInString(key) >= 3 {
					known[key] = true
				}
			}
		}
	}
	all := make([][]personMention, len(texts))
	// A mention with a surname and a first name with a patronymic joins both
	// keys: «Сергей Иванович» elsewhere is the same «Петров».
	alias := map[string]string{}
	for n, text := range texts {
		all[n] = mentions(text, known)
		for _, m := range all[n] {
			if m.surnameWord() >= 0 {
				if name := m.nameKey(); name != "" && !strings.HasSuffix(name, "|") {
					if _, taken := alias[name]; !taken {
						alias[name] = m.key()
					}
				}
			}
		}
	}
	marks := &peopleMarks{}
	numbers := map[string]int{}
	for n, text := range texts {
		piece := peopleText{}
		at := 0
		for _, m := range all[n] {
			key := m.key()
			if key == "" {
				continue
			}
			if joinedKey, ok := alias[key]; ok {
				key = joinedKey
			}
			number, seen := numbers[key]
			if !seen {
				marks.people = append(marks.people, &markedPerson{})
				number = len(marks.people)
				numbers[key] = number
			}
			person := marks.people[number-1]
			written := text[m.start:m.end]
			person.mentions = append(person.mentions, written)
			if s := m.surnameWord(); s >= 0 {
				word := m.words[s]
				person.surnames = append(person.surnames, word.text)
				if person.surname == "" && nominative(word.lower, surnameKey(word.lower)) {
					person.surname = word.text
				}
			}
			piece = append(piece, cutPersonal(text[at:m.start])...)
			piece = append(piece, peoplePiece{personLabel(number), written, number})
			at = m.end
		}
		piece = append(piece, cutPersonal(text[at:])...)
		marks.texts = append(marks.texts, piece)
	}
	return marks
}

// cutPersonal splits a piece of text into the pieces left as they are and the
// personal data of portraits.go replaced by personalMask.
func cutPersonal(text string) peopleText {
	pieces := peopleText{}
	at := 0
	for _, span := range personalSpans(text) {
		if span.start > at {
			pieces = append(pieces, peoplePiece{text[at:span.start], text[at:span.start], 0})
		}
		pieces = append(pieces, peoplePiece{personalMask, text[span.start:span.end], -1})
		at = span.end
	}
	if at < len(text) {
		pieces = append(pieces, peoplePiece{text[at:], text[at:], 0})
	}
	return pieces
}

// text is the n-th text as it goes to the model.
func (m *peopleMarks) text(n int) string { return m.texts[n].shown() }

// name is how the teacher sees a person: the surname as the author names him,
// or the first name and the patronymic as written, or the first mention.
func (m *peopleMarks) name(number int) string {
	if number < 1 || number > len(m.people) {
		return personLabel(number)
	}
	p := m.people[number-1]
	switch {
	case p.surname != "":
		return p.surname
	case len(p.surnames) > 0:
		return namedForm(p.surnames[0])
	}
	return p.mentions[0]
}

// namedForm turns a surname in another case into the form it is named by:
// «Петрову» → «Петров», «Достоевского» → «Достоевский».
func namedForm(written string) string {
	lower := normalWord(written)
	key := surnameKey(lower)
	if key == lower || !strings.HasPrefix(lower, key) {
		return written
	}
	stem := string([]rune(written)[:utf8.RuneCountInString(key)])
	if surnameSky.MatchString(lower) {
		return stem + "ий"
	}
	return stem
}

// names puts the names back in a text written by the model.
func (m *peopleMarks) names(text string) string {
	return personLabels.ReplaceAllStringFunc(text, func(label string) string {
		number, _ := strconv.Atoi(personLabels.FindStringSubmatch(label)[1])
		return m.name(number)
	})
}

// quote puts back the words of the author in a quote: the quote is found in a
// text as it went to the model and the same piece of the text as written is
// taken, so the case and the mistakes of the author stay. A quote not found
// gets the names as the teacher sees them.
func (m *peopleMarks) quote(text string) string {
	if !personLabels.MatchString(text) && !strings.Contains(text, personalMask) {
		return text
	}
	want, _ := quoteRunes(text)
	if len(want) > 0 {
		for _, piece := range m.texts {
			shown := piece.shown()
			have, at := quoteRunes(shown)
			if i := runesIndex(have, want); i >= 0 {
				last := i + len(want) - 1
				_, size := utf8.DecodeRuneInString(shown[at[last]:])
				return piece.orig(at[i], at[last]+size)
			}
		}
	}
	return m.names(text)
}

// orig returns what the author wrote in place of the shown bytes [from, to):
// a label or a cut piece cut in the middle is taken whole.
func (t peopleText) orig(from, to int) string {
	var out strings.Builder
	at := 0
	for _, p := range t {
		start, end := at, at+len(p.shown)
		at = end
		if end <= from || start >= to {
			continue
		}
		if p.person != 0 {
			out.WriteString(p.orig)
			continue
		}
		out.WriteString(p.orig[max(from, start)-start : min(to, end)-start])
	}
	return out.String()
}

// quoteRunes is a text in the form of quoteForm, rune by rune, with the byte
// at which every rune stands in the text.
func quoteRunes(text string) ([]rune, []int) {
	runes, at := []rune{}, []int{}
	space := false
	for i, r := range text {
		if unicode.IsSpace(r) {
			space = len(runes) > 0
			continue
		}
		if space {
			runes, at = append(runes, ' '), append(at, i)
			space = false
		}
		for _, n := range []rune(quoteSpelling.Replace(strings.ToLower(string(r)))) {
			runes, at = append(runes, n), append(at, i)
		}
	}
	return runes, at
}

func runesIndex(have, want []rune) int {
	for i := 0; i+len(want) <= len(have); i++ {
		if slices.Equal(have[i:i+len(want)], want) {
			return i
		}
	}
	return -1
}

// restore puts the people back in an answer of the model: in a quote the words
// of the author, elsewhere the names.
func (m *peopleMarks) restore(output []byte) []byte {
	if m == nil || len(m.people) == 0 && !strings.Contains(string(output), personalMask) {
		return output
	}
	var value any
	if json.Unmarshal(output, &value) != nil {
		return output
	}
	var walk func(key string, v any) any
	walk = func(key string, v any) any {
		switch x := v.(type) {
		case string:
			if key == "quote" {
				return m.quote(x)
			}
			return m.names(x)
		case []any:
			for i := range x {
				x[i] = walk(key, x[i])
			}
		case map[string]any:
			for k, item := range x {
				x[k] = walk(k, item)
			}
		}
		return v
	}
	data, err := json.Marshal(walk("", value))
	if err != nil {
		return output
	}
	return data
}

// markRows replaces the people in the field «ответ» of the rows under the keys
// of the input, in order, and returns the labels.
func markRows(payload *object, keys ...string) *peopleMarks {
	rows := []*object{}
	for _, key := range keys {
		rows = append(rows, objectList(payload.values[key])...)
	}
	texts := make([]string, len(rows))
	for n, row := range rows {
		texts[n] = objectText(row, "ответ")
	}
	marks := markPeople(texts)
	for n, row := range rows {
		row.set("ответ", marks.text(n))
	}
	return marks
}

// labelProblem finds a label of a person in a text where only a position may
// stand.
func labelProblem(where, text string) string {
	if personLoose.MatchString(text) {
		return fmt.Sprintf("%s: метка человека в тексте — называй его по должности: преподаватель, куратор взвода, начальник цикла; метка — только в teachers.name", where)
	}
	return ""
}
