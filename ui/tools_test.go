package ui_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func tool(t *testing.T, body string) (map[string]string, int) {
	t.Helper()
	rec := post(t, "/api/tools", body)
	if rec.Code != 200 {
		return map[string]string{"error": rec.Body.String()}, rec.Code
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("bad response: %v (%s)", err, rec.Body)
	}
	return out, rec.Code
}

// The catalog drives the interface, so it must say which operations reverse.
func TestToolCatalogMarksReversibility(t *testing.T) {
	rec := get(t, "/api/tools")
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	var tools []struct {
		Op         string `json:"op"`
		Group      string `json:"group"`
		Reversible bool   `json:"reversible"`
	}
	json.Unmarshal(rec.Body.Bytes(), &tools)

	byOp := map[string]bool{}
	group := map[string]string{}
	for _, tl := range tools {
		byOp[tl.Op] = tl.Reversible
		group[tl.Op] = tl.Group
	}
	for _, op := range []string{"sha256", "md5", "pbkdf2-sha256", "hmac-sha256"} {
		if byOp[op] {
			t.Errorf("%s is one-way but the catalog says it reverses", op)
		}
	}
	for _, op := range []string{"base64", "hex", "url", "jwt"} {
		if !byOp[op] {
			t.Errorf("%s round-trips but the catalog says it does not", op)
		}
	}
	if group["pbkdf2-sha256"] != "password" {
		t.Errorf("pbkdf2 should be grouped as a password function, got %q", group["pbkdf2-sha256"])
	}
}

// Hashes must match an independent computation, not merely look like hex.
func TestSHA256MatchesStdlib(t *testing.T) {
	out, _ := tool(t, `{"op":"sha256","input":"synth"}`)
	sum := sha256.Sum256([]byte("synth"))
	if want := hex.EncodeToString(sum[:]); out["output"] != want {
		t.Fatalf("sha256 = %q, want %q", out["output"], want)
	}
}

// Asking to decode a hash must be refused with an explanation. Silently
// returning something would teach that hashes are reversible.
func TestHashCannotBeDecoded(t *testing.T) {
	for _, op := range []string{"sha256", "md5", "pbkdf2-sha256"} {
		out, code := tool(t, `{"op":"`+op+`-decode","input":"abc"}`)
		if code != 400 {
			t.Errorf("%s-decode returned %d, want 400", op, code)
		}
		if !strings.Contains(out["error"], "one-way") {
			t.Errorf("%s-decode error does not explain why: %s", op, out["error"])
		}
	}
}

// Broken algorithms stay available for fixtures, but must carry the warning.
func TestBrokenHashesAreLabelled(t *testing.T) {
	for _, op := range []string{"md5", "sha1"} {
		out, _ := tool(t, `{"op":"`+op+`","input":"x"}`)
		if !strings.Contains(strings.ToLower(out["note"]), "broken") {
			t.Errorf("%s result carries no warning: %q", op, out["note"])
		}
	}
}

// Encodings must round-trip exactly, including non-ASCII.
func TestEncodingsRoundTrip(t *testing.T) {
	const input = "Salom, dunyo! — ok/uz?x=1 №"
	for _, op := range []string{"base64", "base64url", "hex", "url"} {
		enc, code := tool(t, `{"op":"`+op+`-encode","input":`+quote(input)+`}`)
		if code != 200 {
			t.Fatalf("%s encode failed: %v", op, enc)
		}
		dec, code := tool(t, `{"op":"`+op+`-decode","input":`+quote(enc["output"])+`}`)
		if code != 200 {
			t.Fatalf("%s decode failed: %v", op, dec)
		}
		if dec["output"] != input {
			t.Errorf("%s round trip changed the value: %q", op, dec["output"])
		}
	}
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// A password hash must be self-describing and reproducible, and a different
// salt must give a different digest.
func TestPasswordHashIsSaltedAndReproducible(t *testing.T) {
	a, code := tool(t, `{"op":"pbkdf2-sha256","input":"hunter2","salt":"s1","iterations":1000}`)
	if code != 200 {
		t.Fatalf("failed: %v", a)
	}
	if !strings.HasPrefix(a["output"], "pbkdf2-sha256$1000$") {
		t.Fatalf("hash is not self-describing: %q", a["output"])
	}
	b, _ := tool(t, `{"op":"pbkdf2-sha256","input":"hunter2","salt":"s1","iterations":1000}`)
	if a["output"] != b["output"] {
		t.Fatal("same password, salt and cost gave different hashes")
	}
	c, _ := tool(t, `{"op":"pbkdf2-sha256","input":"hunter2","salt":"s2","iterations":1000}`)
	if a["output"] == c["output"] {
		t.Fatal("changing the salt did not change the hash — the salt is being ignored")
	}
}

// A missing salt is the mistake salts exist to prevent, so it must be refused.
func TestPasswordHashRequiresSalt(t *testing.T) {
	out, code := tool(t, `{"op":"pbkdf2-sha256","input":"hunter2","iterations":1000}`)
	if code != 400 {
		t.Fatalf("status %d, want 400", code)
	}
	if !strings.Contains(out["error"], "salt") {
		t.Fatalf("error does not mention the salt: %s", out["error"])
	}
}

// An absurd work factor must be refused rather than hanging the workbench.
func TestPasswordHashBoundsIterations(t *testing.T) {
	_, code := tool(t, `{"op":"pbkdf2-sha256","input":"x","salt":"s","iterations":999999999}`)
	if code != 400 {
		t.Fatalf("status %d, want 400", code)
	}
}

// HMAC without a key is meaningless and must not quietly become a plain hash.
func TestHMACRequiresKey(t *testing.T) {
	_, code := tool(t, `{"op":"hmac-sha256","input":"x"}`)
	if code != 400 {
		t.Fatalf("status %d, want 400", code)
	}
	out, code := tool(t, `{"op":"hmac-sha256","input":"x","key":"k"}`)
	if code != 200 {
		t.Fatalf("keyed hmac failed: %v", out)
	}
	other, _ := tool(t, `{"op":"hmac-sha256","input":"x","key":"different"}`)
	if out["output"] == other["output"] {
		t.Fatal("changing the key did not change the mac")
	}
}

// JWT decoding must show the claims and must not claim the signature is valid.
func TestJWTDecodeDoesNotClaimVerification(t *testing.T) {
	// {"alg":"HS256"} . {"sub":"1234","name":"Ali"} . signature
	const token = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0IiwibmFtZSI6IkFsaSJ9.c2ln"
	out, code := tool(t, `{"op":"jwt-decode","input":"`+token+`"}`)
	if code != 200 {
		t.Fatalf("failed: %v", out)
	}
	if !strings.Contains(out["output"], "Ali") {
		t.Fatalf("claims not decoded: %q", out["output"])
	}
	if !strings.Contains(out["note"], "not verified") {
		t.Fatalf("decoding must not imply the signature was checked: %q", out["note"])
	}
}

// Signing a JWT needs a real key, which does not belong in a fake-data
// workbench; the refusal must say so.
func TestJWTEncodeIsRefused(t *testing.T) {
	out, code := tool(t, `{"op":"jwt-encode","input":"{}"}`)
	if code != 400 {
		t.Fatalf("status %d, want 400", code)
	}
	if !strings.Contains(out["error"], "sign") {
		t.Fatalf("refusal is unclear: %s", out["error"])
	}
}

// Malformed input must produce a message, not a panic or a wrong answer.
func TestToolsRejectBadInput(t *testing.T) {
	for _, body := range []string{
		`{"op":"hex-decode","input":"zzz"}`,
		`{"op":"base64-decode","input":"!!!!"}`,
		`{"op":"jwt-decode","input":"not-a-token"}`,
		`{"op":"telepathy","input":"x"}`,
	} {
		if _, code := tool(t, body); code != 400 {
			t.Errorf("%s returned %d, want 400", body, code)
		}
	}
}
