package front

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

// The page is served and talks to the api.
func TestHandler_ServesThePage(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.Contains(string(body), "/api/games") {
		t.Fatalf("page: %d", resp.StatusCode)
	}
}
