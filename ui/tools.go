package ui

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"hash/crc32"
	"net/http"
	"net/url"
	"strings"
)

// Hashing and encoding utilities for the workbench.
//
// These are two different things and the interface must not blur them:
//
//   - A hash is one-way. There is no "decode" for SHA-256, and offering one
//     next to it would teach a falsehood. Hashes here report Reversible: false.
//   - An encoding is two-way. Base64, hex and percent-encoding round-trip, and
//     both directions are offered.
//
// Everything runs in this process on this machine, like the rest of Synth.
// Nothing is sent anywhere.

// defaultIterations is the PBKDF2 work factor used when the caller does not
// choose one. It is a deliberate cost: a fast password hash is a broken one.
const defaultIterations = 600_000

// maxIterations bounds the work a single request can ask for, so the workbench
// cannot be made to hang itself.
const maxIterations = 5_000_000

// maxToolInput bounds the input size. This is a workbench utility, not a file
// processor.
const maxToolInput = 1 << 20 // 1 MiB

type toolRequest struct {
	Op    string `json:"op"`
	Input string `json:"input"`
	// Key is the secret for HMAC.
	Key string `json:"key"`
	// Salt and Iterations apply to password derivation.
	Salt       string `json:"salt"`
	Iterations int    `json:"iterations"`
}

type toolResponse struct {
	Output string `json:"output"`
	// Note carries a caveat the user should see next to the result — that a
	// hash cannot be reversed, or that an algorithm is unfit for passwords.
	Note string `json:"note,omitempty"`
}

// toolInfo describes one operation for the interface.
type toolInfo struct {
	Op    string `json:"op"`
	Group string `json:"group"`
	// Reversible reports whether a matching decode operation exists.
	Reversible bool `json:"reversible"`
	// NeedsKey and NeedsSalt tell the interface which extra inputs to show.
	NeedsKey  bool `json:"needsKey,omitempty"`
	NeedsSalt bool `json:"needsSalt,omitempty"`
	// Warn marks an algorithm that is fine for fixtures and wrong for security.
	Warn string `json:"warn,omitempty"`
}

var toolCatalog = []toolInfo{
	{Op: "md5", Group: "hash", Warn: "broken for security; use it only for fixtures and checksums"},
	{Op: "sha1", Group: "hash", Warn: "broken for security; use it only for fixtures and checksums"},
	{Op: "sha256", Group: "hash"},
	{Op: "sha512", Group: "hash"},
	{Op: "crc32", Group: "hash", Warn: "a checksum, not a hash: trivially forged"},
	{Op: "hmac-sha256", Group: "hash", NeedsKey: true},

	{Op: "pbkdf2-sha256", Group: "password", NeedsSalt: true},

	{Op: "base64", Group: "encoding", Reversible: true},
	{Op: "base64url", Group: "encoding", Reversible: true},
	{Op: "hex", Group: "encoding", Reversible: true},
	{Op: "url", Group: "encoding", Reversible: true},
	{Op: "jwt", Group: "encoding", Reversible: true},
}

func handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, toolCatalog)
		return
	}
	var req toolRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxToolInput+4096)).Decode(&req); err != nil {
		http.Error(w, "cannot read the request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Input) > maxToolInput {
		http.Error(w, "input is too large for the workbench", http.StatusBadRequest)
		return
	}
	res, err := runTool(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, res)
}

// runTool performs one operation. Decode variants are named "<op>-decode";
// everything else encodes or hashes.
func runTool(req toolRequest) (toolResponse, error) {
	op, decode := strings.CutSuffix(req.Op, "-decode")
	op = strings.TrimSuffix(op, "-encode")

	if decode && !reversible(op) {
		return toolResponse{}, fmt.Errorf("%s is a one-way hash: it cannot be decoded, "+
			"only recomputed and compared", op)
	}

	switch op {
	case "md5":
		return hashed(md5.New(), req.Input, "MD5 is broken for security; fine for a fixture or a checksum"), nil
	case "sha1":
		return hashed(sha1.New(), req.Input, "SHA-1 is broken for security; fine for a fixture or a checksum"), nil
	case "sha256":
		return hashed(sha256.New(), req.Input, ""), nil
	case "sha512":
		return hashed(sha512.New(), req.Input, ""), nil
	case "crc32":
		return toolResponse{
			Output: fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(req.Input))),
			Note:   "CRC32 detects accidental corruption, not tampering",
		}, nil

	case "hmac-sha256":
		if req.Key == "" {
			return toolResponse{}, fmt.Errorf("hmac-sha256 needs a key")
		}
		m := hmac.New(sha256.New, []byte(req.Key))
		m.Write([]byte(req.Input))
		return toolResponse{Output: hex.EncodeToString(m.Sum(nil))}, nil

	case "pbkdf2-sha256":
		return passwordHash(req)

	case "base64":
		return codec(base64.StdEncoding, req.Input, decode)
	case "base64url":
		return codec(base64.RawURLEncoding, req.Input, decode)
	case "hex":
		if decode {
			b, err := hex.DecodeString(strings.TrimSpace(req.Input))
			if err != nil {
				return toolResponse{}, fmt.Errorf("not valid hex: %w", err)
			}
			return toolResponse{Output: string(b)}, nil
		}
		return toolResponse{Output: hex.EncodeToString([]byte(req.Input))}, nil
	case "url":
		if decode {
			s, err := url.QueryUnescape(req.Input)
			if err != nil {
				return toolResponse{}, fmt.Errorf("not valid percent-encoding: %w", err)
			}
			return toolResponse{Output: s}, nil
		}
		return toolResponse{Output: url.QueryEscape(req.Input)}, nil

	case "jwt":
		if !decode {
			return toolResponse{}, fmt.Errorf("this workbench decodes JWTs but does not " +
				"sign them: signing needs your real key, which does not belong here")
		}
		return decodeJWT(req.Input)
	}
	return toolResponse{}, fmt.Errorf("unknown operation %q", req.Op)
}

func reversible(op string) bool {
	for _, t := range toolCatalog {
		if t.Op == op {
			return t.Reversible
		}
	}
	return false
}

func hashed(h hash.Hash, input, note string) toolResponse {
	h.Write([]byte(input))
	return toolResponse{Output: hex.EncodeToString(h.Sum(nil)), Note: note}
}

func codec(enc *base64.Encoding, input string, decode bool) (toolResponse, error) {
	if decode {
		b, err := enc.DecodeString(strings.TrimSpace(input))
		if err != nil {
			return toolResponse{}, fmt.Errorf("not valid base64: %w", err)
		}
		return toolResponse{Output: string(b)}, nil
	}
	return toolResponse{Output: enc.EncodeToString([]byte(input))}, nil
}

// passwordHash derives a PBKDF2-HMAC-SHA256 hash and formats it the way a
// password column actually stores one: algorithm, cost and salt alongside the
// digest, so the value is self-describing and verifiable later.
func passwordHash(req toolRequest) (toolResponse, error) {
	iter := req.Iterations
	if iter <= 0 {
		iter = defaultIterations
	}
	if iter > maxIterations {
		return toolResponse{}, fmt.Errorf("%d iterations is beyond what this "+
			"workbench will run in one request (max %d)", iter, maxIterations)
	}
	salt := req.Salt
	if salt == "" {
		return toolResponse{}, fmt.Errorf("pbkdf2 needs a salt — reusing one salt " +
			"for every password is the mistake the salt exists to prevent")
	}
	key, err := pbkdf2.Key(sha256.New, req.Input, []byte(salt), iter, 32)
	if err != nil {
		return toolResponse{}, err
	}
	return toolResponse{
		Output: fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iter,
			base64.RawStdEncoding.EncodeToString([]byte(salt)),
			base64.RawStdEncoding.EncodeToString(key)),
		Note: "one-way: verify by re-deriving with the same salt and cost",
	}, nil
}

// decodeJWT splits a token and decodes its header and payload. The signature is
// left alone: checking it needs the issuer's key, and a workbench that pretended
// to verify without one would be worse than one that does not try.
func decodeJWT(token string) (toolResponse, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) < 2 {
		return toolResponse{}, fmt.Errorf("not a JWT: expected header.payload.signature")
	}
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return toolResponse{}, fmt.Errorf("header is not valid base64url: %w", err)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return toolResponse{}, fmt.Errorf("payload is not valid base64url: %w", err)
	}
	return toolResponse{
		Output: indent(header) + "\n" + indent(payload),
		Note:   "signature not verified — that needs the issuer's key",
	}, nil
}

func indent(raw []byte) string {
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return string(raw)
	}
	pretty, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(pretty)
}
