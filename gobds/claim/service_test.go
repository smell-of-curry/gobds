package claim

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/smell-of-curry/gobds/gobds/service"
)

func TestServiceUsesAdvertisedLastModifiedFor304(t *testing.T) {
	const modified = "Tue, 14 Jul 2026 12:00:00 GMT"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if requests == 2 {
			if got := request.Header.Get("if-modified-since"); got != modified {
				t.Errorf("If-Modified-Since = %q", got)
			}
			writer.WriteHeader(http.StatusNotModified)
			return
		}
		writer.Header().Set("Last-Modified", modified)
		_, _ = writer.Write([]byte(`[{"_key":"one","data":{"claimId":"one","playerXUID":"owner","location":{"dimension":"minecraft:overworld","pos1":{"x":0,"z":0},"pos2":{"x":1,"z":1}}}}]`))
	}))
	defer server.Close()

	client := NewService(service.Config{Enabled: true, URL: server.URL}, slog.Default())
	first, err := client.FetchClaims()
	if err != nil || len(first.Claims) != 1 || first.NotModified {
		t.Fatalf("unexpected initial fetch: result=%+v err=%v", first, err)
	}
	second, err := client.FetchClaims()
	if err != nil || !second.NotModified {
		t.Fatalf("unexpected revalidation: result=%+v err=%v", second, err)
	}
}
