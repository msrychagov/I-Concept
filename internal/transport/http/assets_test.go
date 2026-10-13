package httptransport

import (
	"io/fs"
	"strings"
	"testing"
	"vucdiagnostic/web"
)

func TestEmbeddedFrontendAssets(t *testing.T) {
	a := testApp(t)
	for _, route := range []string{"/", "/teacher", "/student", "/survey", "/autobiography"} {
		requireCode(t, callAccount(t, a, route, nil, nil), 200)
	}
	if err := fs.WalkDir(web.Files, "static", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		response := callAccount(t, a, "/assets/"+strings.TrimPrefix(path, "static/"), nil, nil)
		requireCode(t, response, 200)
		if response.Body.Len() == 0 {
			t.Fatalf("empty asset: %s", path)
		}
		if response.Header().Get("Content-Type") == "" {
			t.Fatalf("missing MIME type: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	requireCode(t, callAccount(t, a, "/assets/bank.json", nil, nil), 401)
	requireCode(t, callAccount(t, a, "/assets/unknown.js", nil, nil), 404)
}
