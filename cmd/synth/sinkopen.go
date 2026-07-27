package main

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// openSink opens the output destination named by path, wrapping it in a
// compressor when the filename asks for one.
//
// Compression is decided here and nowhere else, so every format gets it for
// free — the encoders keep writing to a plain io.Writer and never learn that
// gzip exists. The filename carries both facts: the inner extension is the
// format, the outer one is the compression.
//
//	users.csv        csv,   uncompressed
//	users.csv.gz     csv,   gzip
//	users.jsonl.zst  jsonl, zstd
//
// An empty path means stdout, which is never compressed — a terminal full of
// binary helps nobody, and a pipe can be fed through gzip by the shell.
//
// The returned Close flushes the compressor before closing the file and
// reports either failure. That matters at the sizes this exists for: a
// truncated multi-gigabyte export that reports success is worse than one that
// fails loudly.
func openSink(path string) (io.WriteCloser, error) {
	return openSinkMode(path, false)
}

// openSinkMode opens the sink, appending to the file rather than truncating it
// when appendMode is set. Appending never applies to stdout.
func openSinkMode(path string, appendMode bool) (io.WriteCloser, error) {
	if path == "" {
		return nopCloser{os.Stdout}, nil
	}
	f, err := openFile(path, appendMode)
	if err != nil {
		return nil, err
	}
	switch compressionFromExt(path) {
	case "gz":
		return &sink{w: gzip.NewWriter(f), f: f}, nil
	case "zst":
		z, err := zstd.NewWriter(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		return &sink{w: z, f: f}, nil
	default:
		return f, nil
	}
}

// openFile creates the output file, or opens it for appending when appendMode
// is set. A concatenated gzip or zstd stream is still valid — decoders read the
// members in sequence — so appending works through the compressor too.
func openFile(path string, appendMode bool) (*os.File, error) {
	if appendMode {
		return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	}
	return os.Create(path)
}

// sink pairs a compressor with the file underneath it.
type sink struct {
	w io.WriteCloser
	f *os.File
}

func (s *sink) Write(p []byte) (int, error) { return s.w.Write(p) }

func (s *sink) Close() error {
	// The compressor closes first: closing it is what flushes its final block.
	// The file is closed either way, so a compressor failure does not also
	// leak a descriptor.
	err := s.w.Close()
	if cerr := s.f.Close(); err == nil {
		err = cerr
	}
	return err
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

// compressionFromExt returns the compression named by the path's final
// extension, or "" if it names none.
func compressionFromExt(path string) string {
	switch strings.TrimPrefix(filepath.Ext(path), ".") {
	case "gz", "gzip":
		return "gz"
	case "zst", "zstd":
		return "zst"
	default:
		return ""
	}
}

// stripCompressionExt removes a trailing compression extension so the format
// can be read from the one beneath it.
func stripCompressionExt(path string) string {
	if compressionFromExt(path) != "" {
		return strings.TrimSuffix(path, filepath.Ext(path))
	}
	return path
}
