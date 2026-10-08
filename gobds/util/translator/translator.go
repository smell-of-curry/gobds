package translator

import (
	"strings"
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

// SetTranslationFor merges t into the map already stored for lang.
// A key present in both maps keeps the value from t (later pack wins).
func SetTranslationFor(lang string, t map[string]string) {
	translationMu.Lock()
	defer translationMu.Unlock()
	dst := translations[lang]
	if dst == nil {
		dst = make(map[string]string, len(t))
		translations[lang] = dst
	}
	for k, v := range t {
		dst[k] = v
	}
}

// Setup loads every language file in the pack into the shared translation
// map. A pack with no texts/ directory (audio, 3d, and other asset-only
// packs) is skipped. Split texts packs list every language in
// languages.json but ship one .lang each, so missing .lang files are skipped.
func Setup(rp *resource.Pack) error {
	languages, err := SupportedLanguages(rp)
	if err != nil {
		if strings.Contains(err.Error(), "not found in the resource pack") {
			return nil
		}
		return err
	}
	for _, l := range languages {
		mapped, err := TranslationMapFor(rp, l)
		if err != nil {
			if strings.Contains(err.Error(), "not found in the resource pack") {
				continue
			}
			return err
		}
		SetTranslationFor(l, mapped)
	}
	return nil
}
