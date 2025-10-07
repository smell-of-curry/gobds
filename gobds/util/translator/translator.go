package translator

import (
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// MappedTranslations ...
type MappedTranslations map[string]map[string]string

var (
	translations  = make(MappedTranslations)
	translationMu sync.RWMutex
)

// TranslationFor ...
func TranslationFor(lang string) (map[string]string, bool) {
	translationMu.RLock()
	defer translationMu.RUnlock()
	t, ok := translations[lang]
	if !ok {
		return nil, ok
	}
	return t, ok
}

// SetTranslationFor ...
func SetTranslationFor(lang string, t map[string]string) {
	translationMu.Lock()
	defer translationMu.Unlock()
	translations[lang] = t
}

// Setup ...
func Setup(rp *resource.Pack) error {
	languages, exists := SupportedLanguages(rp)
	if exists != nil {
		return exists
	}
	for _, l := range languages {
		mapped, err := TranslationMapFor(rp, l)
		if err != nil {
			return err
		}
		SetTranslationFor(l, mapped)
	}
	return nil
}
