package translator

import (
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// MappedTranslations ...
type MappedTranslations map[string]map[string]string

var (
	translations  MappedTranslations
	translationMu sync.RWMutex
)

func init() {
	translations = make(MappedTranslations)
}

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
func Setup(rp *resource.Pack) (err error) {
	languages, err := SupportedLanguages(rp)
	if err != nil {
		return err
	}
	for _, l := range languages {
		mapped, err := TranslationMapFor(rp, l)
		if err != nil {
			return err
		}
		SetTranslationFor(l, mapped)
	}
	return
}
