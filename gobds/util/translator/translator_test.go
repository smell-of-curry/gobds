package translator

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/resource"
)

const testManifest = `{
	"format_version": 2,
	"header": {
		"name": "test",
		"description": "test",
		"uuid": "358dc189-8d9e-4fdb-b4e1-f8e0fb855f9a",
		"version": [1, 0, 0],
		"min_engine_version": [1, 21, 0]
	},
	"modules": [{
		"type": "resources",
		"uuid": "388de949-18e5-4228-a447-f63237aa45a2",
		"version": [1, 0, 0]
	}]
}`

func packWithFiles(t *testing.T, files map[string]string) *resource.Pack {
	t.Helper()
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	pack, err := resource.Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	return pack
}

func TestSetupMergesOverlappingKeys(t *testing.T) {
	const lang = "zz_MERGE"
	first := packWithFiles(t, map[string]string{
		"manifest.json":        testManifest,
		"texts/languages.json": `["zz_MERGE"]`,
		"texts/zz_MERGE.lang":  "shared=from-first\nonlyFirst=1\n",
	})
	second := packWithFiles(t, map[string]string{
		"manifest.json":        testManifest,
		"texts/languages.json": `["zz_MERGE"]`,
		"texts/zz_MERGE.lang":  "shared=from-second\nonlySecond=2\n",
	})
	if err := Setup(first); err != nil {
		t.Fatal(err)
	}
	if err := Setup(second); err != nil {
		t.Fatal(err)
	}
	got, ok := TranslationFor(lang)
	if !ok {
		t.Fatal("missing merged language")
	}
	if got["shared"] != "from-second" || got["onlyFirst"] != "1" || got["onlySecond"] != "2" {
		t.Fatalf("merged map = %#v", got)
	}
}

func TestSetupSkipsPackWithoutTexts(t *testing.T) {
	pack := packWithFiles(t, map[string]string{"manifest.json": testManifest})
	if err := Setup(pack); err != nil {
		t.Fatalf("pack with no texts/: %v", err)
	}
}
