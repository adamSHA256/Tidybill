package i18n

import "fmt"

type Lang string

const (
	CS Lang = "cs"
	SK Lang = "sk"
	EN Lang = "en"
)

var currentLang Lang = CS

var messages = map[Lang]map[string]string{
	CS: messagesCS,
	SK: messagesSK,
	EN: messagesEN,
}

func SetLang(lang Lang) {
	if _, ok := messages[lang]; ok {
		currentLang = lang
	}
}

func GetLang() Lang {
	return currentLang
}

// T returns a translated string for the given key.
func T(key string) string {
	if msg, ok := messages[currentLang][key]; ok {
		return msg
	}
	// Fallback to Czech
	if msg, ok := messages[CS][key]; ok {
		return msg
	}
	return key
}

// Tf returns a formatted translated string.
func Tf(key string, args ...interface{}) string {
	return fmt.Sprintf(T(key), args...)
}

// AvailableLanguages returns all supported language codes.
func AvailableLanguages() []Lang {
	return []Lang{CS, SK, EN}
}
