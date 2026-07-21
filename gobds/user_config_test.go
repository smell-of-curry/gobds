package gobds

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
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
