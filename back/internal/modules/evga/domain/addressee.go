package domain

import "strings"

// Подбор кандидатов в адресаты уведомления (спека 025).
//
// Правило задано аналитиком 13.10.2026: «само учреждение и профильный вышестоящий орган».
// Профильный орган берётся ИЗ НАЗВАНИЯ самого учреждения, а не из справочника
// подчинённости: на боевых данных название совпадает с фактическим выбором аудиторов
// в 63.7% случаев, а справочник (там обычно «аппарат акима») — лишь в 40.6%.

// orgForms — организационные формы, которые не участвуют в сопоставлении названий.
// Порядок важен: длинные формы снимаются раньше коротких, иначе «коммунальное
// государственное учреждение» распадётся и оставит мусор.
var orgForms = []string{
	"коммунальное государственное учреждение",
	"государственное учреждение",
	"коммуналдық мемлекеттік мекемесі",
	"коммуналдык мемлекеттик мекемеси",
	"мемлекеттік мекемесі",
	"республиканское государственное учреждение",
	"ргп на пхв",
	"гккп", "кгкп", "ргу", "кгу", "гу", "ао", "тоо",
	"акимата", "әкімдігінің", "акимат",
}

// parentMarkers — маркеры подчинённости в названии учреждения (спека 025 FR-2).
// Русские: профильный орган идёт ОТ маркера до конца строки.
var parentMarkers = []string{
	"управления", "управление",
	"департамента", "департамент",
	"комитета", "комитет",
	"министерства", "министерство",
}

// kzParentMarker — казахский порядок слов обратный: орган стоит ПЕРЕД маркером,
// поэтому берётся фрагмент до маркера включительно.
const kzParentMarker = "басқармасы"

// NormalizeOrgName — название организации для сопоставления: нижний регистр,
// без кавычек, орг-форм и кодового суффикса вида « - 2295013» (спека 025 FR-3).
func NormalizeOrgName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Map(func(r rune) rune {
		switch r {
		case '«', '»', '"', '\'', '`':
			return ' '
		}
		return r
	}, s)
	s = stripCodeSuffix(s)
	for _, f := range orgForms {
		s = strings.ReplaceAll(s, f, " ")
	}
	return strings.Join(strings.Fields(s), " ")
}

// stripCodeSuffix убирает хвост « - 2295013»: в its_cli.title код организации
// дописан к наименованию и мешает сопоставлению.
func stripCodeSuffix(s string) string {
	i := strings.LastIndex(s, " - ")
	if i < 0 {
		return s
	}
	tail := strings.TrimSpace(s[i+3:])
	if len(tail) < 4 {
		return s
	}
	for _, r := range tail {
		if r < '0' || r > '9' {
			return s
		}
	}
	return strings.TrimSpace(s[:i])
}

// ParentOrgFromName — профильный вышестоящий орган из названия учреждения (FR-2).
// Маркер не найден либо фрагмент совпал со всем названием → "" (второго кандидата нет).
func ParentOrgFromName(name string) string {
	low := strings.ToLower(name)

	// казахский вариант: «… білім басқармасы <подразделение>» — орган до маркера включительно
	if i := strings.Index(low, kzParentMarker); i >= 0 {
		cand := strings.TrimSpace(name[:i+len(kzParentMarker)])
		if n := NormalizeOrgName(cand); n != "" && n != NormalizeOrgName(name) {
			return cand
		}
	}

	// русский вариант: берём самый ранний маркер — он отделяет подразделение от органа
	best := -1
	for _, m := range parentMarkers {
		if i := strings.Index(low, m); i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	if best < 0 {
		return ""
	}
	cand := strings.TrimSpace(strings.Trim(name[best:], `"«»' `))
	if n := NormalizeOrgName(cand); n == "" || n == NormalizeOrgName(name) {
		return ""
	}
	return cand
}

// wordPrefixes — значимые слова названия, обрезанные до 5 символов.
// Обрезка гасит расхождение падежей: «управления» и «управление» → «управ».
func wordPrefixes(s string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, w := range strings.Fields(NormalizeOrgName(s)) {
		if len([]rune(w)) < 5 {
			continue
		}
		p := string([]rune(w)[:5])
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// MatchScore — доля значимых слов кандидата, найденных в названии организации (FR-3).
// 0 — не похоже, 1 — все значимые слова кандидата присутствуют.
// Считается от кандидата: адресат в справочнике обычно короче названия учреждения.
func MatchScore(candidate, orgTitle string) float64 {
	cw := wordPrefixes(candidate)
	if len(cw) == 0 {
		return 0
	}
	ow := map[string]bool{}
	for _, p := range wordPrefixes(orgTitle) {
		ow[p] = true
	}
	hit := 0
	for _, p := range cw {
		if ow[p] {
			hit++
		}
	}
	return float64(hit) / float64(len(cw))
}

// AddresseeMatchThreshold — порог уверенности для подсказки (FR-4).
// 0.7 выбран по боевым данным: на нём правило совпадает с фактическим выбором
// аудиторов в 63.7% случаев; ниже порога растут ложные подсказки.
const AddresseeMatchThreshold = 0.7

// AddresseeCandidates — кандидаты в адресаты по названию ГУ (FR-1):
// само учреждение, затем профильный вышестоящий орган (если распознан).
func AddresseeCandidates(senderName string) []string {
	senderName = strings.TrimSpace(senderName)
	if senderName == "" {
		return nil
	}
	out := []string{senderName}
	if p := ParentOrgFromName(senderName); p != "" {
		out = append(out, p)
	}
	return out
}
