package i18n

import "strings"

// CountryToLanguage maps an ISO 3166-1 alpha-2 country code to a supported language.
// Returns LangEN and false if no specific mapping exists.
func CountryToLanguage(countryCode string) (Language, bool) {
	lang, ok := countryLangMap[strings.ToUpper(countryCode)]
	return lang, ok
}

var countryLangMap = map[string]Language{
	// Chinese
	"CN": LangZH, "TW": LangZH, "HK": LangZH, "MO": LangZH, "SG": LangZH,
	// Japanese
	"JP": LangJA,
	// Korean
	"KR": LangKO,
	// German
	"DE": LangDE, "AT": LangDE, "CH": LangDE, "LI": LangDE,
	// Spanish
	"ES": LangES, "MX": LangES, "AR": LangES, "CO": LangES,
	"CL": LangES, "PE": LangES, "VE": LangES, "EC": LangES,
	"UY": LangES, "PY": LangES, "BO": LangES, "CU": LangES,
	// English (explicit, also the default fallback)
	"US": LangEN, "GB": LangEN, "AU": LangEN, "CA": LangEN,
	"NZ": LangEN, "IE": LangEN, "IN": LangEN, "ZA": LangEN,
	"PH": LangEN, "NG": LangEN, "KE": LangEN,
}
