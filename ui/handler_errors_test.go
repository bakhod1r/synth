package ui

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// deadConn is a ResponseWriter whose body writes fail, the way a client that
// hangs up mid-download makes them fail. httptest's recorder cannot do this,
// and a download is exactly where it matters: the handler streams row by row.
type deadConn struct{ header http.Header }

func (d *deadConn) Header() http.Header {
	if d.header == nil {
		d.header = http.Header{}
	}
	return d.header
}
func (d *deadConn) Write([]byte) (int, error) { return 0, errors.New("connection reset by peer") }
func (d *deadConn) WriteHeader(int)           {}

// A JSONL download writes one row at a time. When the client disappears the
// handler stops rather than encoding every remaining row into a dead socket.
func TestHandleGenerateStopsWhenTheClientHangsUp(t *testing.T) {
	body := `{"name":"data","count":500,"format":"jsonl","fields":{"id":{"kind":"int"}}}`
	r := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(body))
	handleGenerate(&deadConn{}, r) // must return rather than block or panic
}

// The workbench posts a name straight into the generated YAML document. A name
// carrying YAML syntax makes that document unparseable, and the handler has to
// report it as a bad request instead of serving a 500.
func TestParseSpecReportsAnUnparseableGeneratedDocument(t *testing.T) {
	body := `{"name":"broken: name: here","count":1,"fields":{"id":{"kind":"int"}}}`
	r := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(body))
	w := httptest.NewRecorder()
	handleGenerate(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d for a name that breaks the generated YAML", w.Code, http.StatusBadRequest)
	}
}
