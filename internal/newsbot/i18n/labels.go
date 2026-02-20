package i18n

// Re-export shared types from pkg/i18n for backward compatibility.
// Existing newsbot code continues to import internal/newsbot/i18n.

import (
	shared "github.com/RobinCoderZhao/devkit-suite/pkg/i18n"
)

// Language is an alias for the shared Language type.
type Language = shared.Language

// Re-export language constants.
const (
	LangZH = shared.LangZH
	LangEN = shared.LangEN
	LangJA = shared.LangJA
	LangKO = shared.LangKO
	LangDE = shared.LangDE
	LangES = shared.LangES
)

// AllLanguages re-exports the shared list.
var AllLanguages = shared.AllLanguages

// Labels is an alias for NewsLabels from the shared package.
type Labels = shared.NewsLabels

// Delegated functions.
func LanguageName(lang Language) string  { return shared.LanguageName(lang) }
func IsValidLanguage(lang string) bool   { return shared.IsValidLanguage(lang) }
func ParseLanguages(s string) []Language { return shared.ParseLanguages(s) }
func GetLabels(lang Language) Labels     { return shared.GetNewsLabels(lang) }
