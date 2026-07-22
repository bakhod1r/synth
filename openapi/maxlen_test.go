package openapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaxLengthBecomesMaxlen(t *testing.T) {
	spec := `
openapi: 3.0.0
info: {title: t, version: "1"}
paths:
  /users:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                nickname: {type: string, maxLength: 12}
                bio: {type: string}
`
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	sch, err := s.Schema("POST", "/users")
	if err != nil {
		t.Fatal(err)
	}
	if got := sch.FieldByName("nickname").Params["maxlen"]; got != "12" {
		t.Errorf("nickname: maxlen = %q, want %q", got, "12")
	}
	if got := sch.FieldByName("bio").Params["maxlen"]; got != "" {
		t.Errorf("bio: maxlen = %q, want empty", got)
	}
}
