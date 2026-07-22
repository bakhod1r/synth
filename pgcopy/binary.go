package pgcopy

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"
)

// The file starts with a fixed signature, a flags word and a header-extension
// length. \377 in the signature is a byte no text encoding produces, so a text
// file fed to a binary COPY is rejected immediately rather than parsed into
// nonsense.
var binarySignature = []byte("PGCOPY\n\377\r\n\x00")

// pgEpoch is 2000-01-01 UTC. Postgres counts timestamps from there, not from
// the Unix epoch.
var pgEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// BinaryWriter writes the COPY binary format — the fastest thing Postgres will
// read, since the server does no parsing at all.
//
// The speed has a price: the file carries no type names, so each field's bytes
// are interpreted as whatever the target column happens to be. A mismatch is
// not a bad row, it is a rejected file or, worse, a value silently misread.
// This is why the CLI writes a matching CREATE TABLE alongside the data — see
// ddl.go, which derives its column types from the same table this encoder
// uses.
type BinaryWriter struct {
	w    *bufio.Writer
	cols []string
	err  error
}

// NewBinary returns a writer emitting the columns in the order given, which is
// the column order the target table must have.
func NewBinary(w io.Writer, cols []string) *BinaryWriter {
	b := &BinaryWriter{w: bufio.NewWriter(w), cols: cols}
	b.write(binarySignature)
	b.writeUint32(0) // flags: row OIDs not included
	b.writeUint32(0) // header extension area: empty
	return b
}

func (b *BinaryWriter) WriteRow(row map[string]any) error {
	if b.err != nil {
		return b.err
	}
	b.writeUint16(uint16(len(b.cols)))
	for _, c := range b.cols {
		v, ok := row[c]
		if !ok || v == nil {
			b.writeUint32(math.MaxUint32) // length -1 means NULL
			continue
		}
		enc, err := encodeBinary(v)
		if err != nil {
			b.err = fmt.Errorf("pgcopy: column %q: %w", c, err)
			return b.err
		}
		b.writeUint32(uint32(len(enc)))
		b.write(enc)
	}
	return b.err
}

// Close writes the trailer and flushes. The trailer is a field count of -1,
// which is how Postgres tells a complete file from a truncated one.
func (b *BinaryWriter) Close() error {
	if b.err != nil {
		return b.err
	}
	b.writeUint16(math.MaxUint16)
	if b.err != nil {
		return b.err
	}
	return b.w.Flush()
}

// encodeBinary renders one value in Postgres's binary wire format.
//
// The five Go types here are every type Synth's 250-odd kinds resolve to, so
// the encoder keys off the value rather than the semantic kind. Anything else
// is an error: binary COPY has no way to say "this field is something you do
// not understand", and guessing produces bytes the server reads as a valid
// value of the wrong type.
func encodeBinary(v any) ([]byte, error) {
	switch x := v.(type) {
	case string:
		return []byte(x), nil
	case []byte:
		return x, nil
	case bool:
		if x {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	case int:
		return be64(uint64(int64(x))), nil
	case int32:
		return be64(uint64(int64(x))), nil
	case int64:
		return be64(uint64(x)), nil
	case float64:
		return be64(math.Float64bits(x)), nil
	case float32:
		return be64(math.Float64bits(float64(x))), nil
	case time.Time:
		return be64(uint64(pgTimestamp(x))), nil
	default:
		return nil, fmt.Errorf("cannot encode %T in binary COPY (use --format pgcopy)", v)
	}
}

// pgTimestamp converts an instant to microseconds since 2000-01-01 UTC.
//
// Using the Unix epoch instead puts every row thirty years out — wrong in a
// way that still looks like a date, so it survives a glance at the output.
func pgTimestamp(t time.Time) int64 {
	d := t.UTC().Sub(pgEpoch)
	return int64(d / time.Microsecond)
}

func be64(v uint64) []byte {
	return binary.BigEndian.AppendUint64(nil, v)
}

// The write helpers hold the first error and then do nothing, so the encoding
// above reads as a sequence of writes rather than a chain of error checks.

func (b *BinaryWriter) write(p []byte) {
	if b.err == nil {
		_, b.err = b.w.Write(p)
	}
}

func (b *BinaryWriter) writeUint16(v uint16) {
	b.write(binary.BigEndian.AppendUint16(nil, v))
}

func (b *BinaryWriter) writeUint32(v uint32) {
	b.write(binary.BigEndian.AppendUint32(nil, v))
}
