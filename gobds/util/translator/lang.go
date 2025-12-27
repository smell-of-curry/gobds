// Package translator provides language translation utilities for the GoBDS proxy.
package translator

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/go-jose/go-jose/v4/json"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/smell-of-curry/gobds/gobds/util"
)

// SupportedLanguages ...
func SupportedLanguages(rp *resource.Pack) (supportedLanguages []string, err error) {
	raw, err := rp.ReadFile("texts/languages.json")
	if err != nil {
		return nil, fmt.Errorf("error while reading languages.json: %w", err)
	}

	supportedLanguagesJSON, err := util.ParseCommentedJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("error while parsing languages.json: %w", err)
	}

	err = json.Unmarshal(supportedLanguagesJSON, &supportedLanguages)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshaling languages.json: %w", err)
	}
	return supportedLanguages, nil
}

// TranslationMapFor ...
func TranslationMapFor(rp *resource.Pack, language string) (langMap map[string]string, err error) {
	raw, err := rp.ReadFile("texts/" + language + ".lang")
	if err != nil {
		return nil, fmt.Errorf("error while reading language file: %w", err)
	}

	langMap = make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("error while parsing line in .lang file: %s", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		langMap[key] = value
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("error while scanning .lang file: %w", err)
	}
	return langMap, nil
}
