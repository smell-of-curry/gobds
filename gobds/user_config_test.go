package gobds

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// minimalPackZip builds the smallest valid .mcpack archive: a zip with a
// manifest.json at its root.
func minimalPackZip(t *testing.T) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f, err := w.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest := `{
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
	if _, err = f.Write([]byte(manifest)); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestReadURLPackHasNoDownloadURL guards against regressing to
// resource.ReadURL: a URL-sourced pack must NOT carry a CDN DownloadURL, or
// the listener advertises it in TexturePackInfo and clients with a stale
// cached version try (and fail) the client-side HTTPS download instead of the
// reliable RakNet chunk path.
func TestReadURLPackHasNoDownloadURL(t *testing.T) {
	data := minimalPackZip(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	pack, err := readURLPack(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if pack.DownloadURL() != "" {
		t.Fatalf("URL pack must be served over RakNet, got DownloadURL %q", pack.DownloadURL())
	}
	if pack.Len() != len(data) {
		t.Fatalf("pack content length mismatch: got %d, want %d", pack.Len(), len(data))
	}
}

func offerNames(offers []packOffer) []string {
	names := make([]string, len(offers))
	for i, offer := range offers {
		names[i] = offer.id
	}
	return names
}

func TestFilterPackOffers(t *testing.T) {
	offers := []packOffer{
		{id: "en"},
		{id: "de", filter: packFilter{Language: "de_DE"}},
		{id: "3d", filter: packFilter{MinMemoryTier: 2}},
		{id: "de-3d", filter: packFilter{MinMemoryTier: 2, Language: "de_DE"}},
		{id: "core"},
		{id: "legacy-url"},
		{id: "legacy-path"},
	}

	tests := []struct {
		name string
		tier int
		lang string
		want []string
	}{
		{
			name: "tier below min drops 3d, dash language matches, order kept",
			tier: 1,
			lang: "de-DE",
			want: []string{"en", "de", "core", "legacy-url", "legacy-path"},
		},
		{
			name: "tier equal and case-insensitive language keeps 3d packs",
			tier: 2,
			lang: "DE_de",
			want: []string{"en", "de", "3d", "de-3d", "core", "legacy-url", "legacy-path"},
		},
		{
			name: "other language drops language packs, high tier keeps 3d",
			tier: 4,
			lang: "en_US",
			want: []string{"en", "3d", "core", "legacy-url", "legacy-path"},
		},
		{
			name: "tier zero and empty language keeps only unfiltered, including legacy",
			tier: 0,
			lang: "",
			want: []string{"en", "core", "legacy-url", "legacy-path"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := offerNames(filterPackOffers(offers, tt.tier, tt.lang))
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPackMatchesEdges(t *testing.T) {
	if !packMatches(packFilter{}, 0, "en_US") {
		t.Fatal("zero filter must match tier 0")
	}
	if packMatches(packFilter{MinMemoryTier: 2}, 1, "en_US") {
		t.Fatal("tier 1 must not satisfy min 2")
	}
	if !packMatches(packFilter{MinMemoryTier: 2}, 2, "en_US") {
		t.Fatal("tier 2 must satisfy min 2")
	}
	if !packMatches(packFilter{Language: "pt-BR"}, 0, "pt_br") {
		t.Fatal("dash and underscore, different case, must match")
	}
	if packMatches(packFilter{MinMemoryTier: 2, Language: "ru_RU"}, 2, "en_US") {
		t.Fatal("language mismatch must fail even when tier matches")
	}
	if !packMatches(packFilter{MinMemoryTier: 2, Language: "ru_RU"}, 4, "ru-RU") {
		t.Fatal("tier above min and normalized language must match")
	}
}