package language

import "strings"

type Language uint8

const (
	LanguageUndefined Language = iota
	LanguageEnglish
	LanguageSpanish
)

func (l Language) String() string {
	switch l {
	case LanguageEnglish:
		return "English"
	case LanguageSpanish:
		return "Spanish"
	default:
		return "Undefined"
	}
}

func (l Language) Parse(s string) Language {
	switch strings.ToLower(s) {
	case "english", "en":
		return LanguageEnglish
	case "spanish", "es":
		return LanguageSpanish
	default:
		return LanguageUndefined
	}
}

func (l Language) IsValid() bool {
	return l == LanguageEnglish || l == LanguageSpanish
}
