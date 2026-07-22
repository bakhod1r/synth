package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestFormatFromExtIgnoresCompressionSuffix(t *testing.T) {
	tests := map[string]string{
		"users.csv":       "csv",
		"users.csv.gz":    "csv",
		"users.jsonl":     "jsonl",
		"users.jsonl.gz":  "jsonl",
		"users.jsonl.zst": "jsonl",
		"users.sql.zst":   "sql",
		"users.parquet":   "parquet",
		// Not a compression suffix, so it decides the format itself
		// (and falls through to the csv default).
		"users.txt": "csv",
		"":          "csv",
	}
	for path, want := range tests {
		if got := formatFromExt(path); got != want {
			t.Errorf("formatFromExt(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestCompressionFromExt(t *testing.T) {
	tests := map[string]string{
		"users.csv":       "",
		"users.csv.gz":    "gz",
		"users.jsonl.zst": "zst",
		"users.gz":        "gz",
		"users.txt":       "",
	}
	for path, want := range tests {
		if got := compressionFromExt(path); got != want {
			t.Errorf("compressionFromExt(%q) = %q, want %q", path, got, want)
		}
	}
}

// The whole contract: what lands on disk, once decompressed, is exactly what
// an uncompressed run would have written.
func TestOpenSinkRoundTrip(t *testing.T) {
	payload := []byte("id,name\n1,Aleksandr\n2,Алексаендр\n")
	for _, ext := range []string{"", ".gz", ".zst"} {
		t.Run("ext"+ext, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "users.csv"+ext)
			w, err := openSink(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(payload); err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}

			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got := decompress(t, ext, raw)
			if !bytes.Equal(got, payload) {
				t.Errorf("round trip = %q, want %q", got, payload)
			}
			if ext != "" && bytes.Equal(raw, payload) {
				t.Errorf("%s file was written uncompressed", ext)
			}
		})
	}
}

// Close must report a compressor flush failure rather than leaving a truncated
// file behind: a half-written 40GB export that reports success is worse than
// one that fails loudly.
func TestOpenSinkClosesCompressorBeforeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.csv.gz")
	w, err := openSink(path)
	if err != nil {
		t.Fatal(err)
	}
	// Enough to be buffered rather than flushed by chance.
	if _, err := w.Write(bytes.Repeat([]byte("x"), 1<<16)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(decompress(t, ".gz", raw)) != 1<<16 {
		t.Error("compressor was not flushed before the file closed")
	}
}

func decompress(t *testing.T, ext string, raw []byte) []byte {
	t.Helper()
	switch ext {
	case ".gz":
		r, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		return out
	case ".zst":
		r, err := zstd.NewReader(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		return out
	default:
		return raw
	}
}
