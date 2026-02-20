// Package i18n provides shared multi-language support for all products.
// Language definitions, label infrastructure, and common utilities.
package i18n

import "strings"

// Language represents a supported output language.
type Language string

const (
	LangZH Language = "zh" // Chinese (default)
	LangEN Language = "en" // English
	LangJA Language = "ja" // Japanese
	LangKO Language = "ko" // Korean
	LangDE Language = "de" // German
	LangES Language = "es" // Spanish
)

// AllLanguages is the list of all supported languages.
var AllLanguages = []Language{LangZH, LangEN, LangJA, LangKO, LangDE, LangES}

// LanguageName returns the human-readable display name of a language.
func LanguageName(lang Language) string {
	switch lang {
	case LangZH:
		return "中文"
	case LangEN:
		return "English"
	case LangJA:
		return "日本語"
	case LangKO:
		return "한국어"
	case LangDE:
		return "Deutsch"
	case LangES:
		return "Español"
	default:
		return string(lang)
	}
}

// IsValidLanguage checks if a language code is supported.
func IsValidLanguage(lang string) bool {
	for _, l := range AllLanguages {
		if Language(lang) == l {
			return true
		}
	}
	return false
}

// ParseLanguages splits a comma-separated language string and validate each.
func ParseLanguages(s string) []Language {
	parts := strings.Split(s, ",")
	var langs []Language
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if IsValidLanguage(p) {
			langs = append(langs, Language(p))
		}
	}
	if len(langs) == 0 {
		return []Language{LangZH}
	}
	return langs
}
