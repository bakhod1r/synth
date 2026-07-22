package pgcopy

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

var wantHeader = append(
	[]byte("PGCOPY\n\377\r\n\x00"), // 11-byte signature
	0, 0, 0, 0,                     // flags: no OIDs
	0, 0, 0, 0, // header extension length: none
)

func TestBinaryHeaderAndTrailer(t *testing.T) {
	var buf bytes.Buffer
	w := NewBinary(&buf, []string{"a"})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got := buf.Bytes()
	if !bytes.Equal(got[:19], wantHeader) {
		t.Errorf("header = %v, want %v", got[:19], wantHeader)
	}
	if want := []byte{0xFF, 0xFF}; !bytes.Equal(got[19:], want) {
		t.Errorf("trailer = %v, want %v", got[19:], want)
	}
}

func TestBinaryRowEncoding(t *testing.T) {
	var buf bytes.Buffer
	w := NewBinary(&buf, []string{"s", "i", "f", "b", "t", "n"})
	err := w.WriteRow(map[string]any{
		"s": "hi",
		"i": int64(1),
		"f": 1.5,
		"b": true,
		"t": time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), // the PG epoch: 0
		"n": nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	body := buf.Bytes()[len(wantHeader) : buf.Len()-2]

	var want []byte
	want = binary.BigEndian.AppendUint16(want, 6) // field count
	want = binary.BigEndian.AppendUint32(want, 2)
	want = append(want, "hi"...)
	want = binary.BigEndian.AppendUint32(want, 8)
	want = binary.BigEndian.AppendUint64(want, 1)
	want = binary.BigEndian.AppendUint32(want, 8)
	want = binary.BigEndian.AppendUint64(want, math.Float64bits(1.5))
	want = binary.BigEndian.AppendUint32(want, 1)
	want = append(want, 1)
	want = binary.BigEndian.AppendUint32(want, 8)
	want = binary.BigEndian.AppendUint64(want, 0)
	want = binary.BigEndian.AppendUint32(want, math.MaxUint32) // -1: NULL

	if !bytes.Equal(body, want) {
		t.Errorf("row = % x\nwant  = % x", body, want)
	}
}

// Postgres counts microseconds from 2000-01-01, not from the Unix epoch. Get
// this wrong and every timestamp lands 30 years off — plausible enough to
// survive review.
func TestBinaryTimestampEpoch(t *testing.T) {
	tests := []struct {
		when time.Time
		want int64
	}{
		{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), 0},
		{time.Date(2000, 1, 1, 0, 0, 1, 0, time.UTC), 1_000_000},
		{time.Date(1999, 12, 31, 23, 59, 59, 0, time.UTC), -1_000_000},
		{time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), -946684800_000_000},
		{time.Date(2000, 1, 1, 0, 0, 0, 1000, time.UTC), 1}, // 1µs
	}
	for _, tt := range tests {
		if got := pgTimestamp(tt.when); got != tt.want {
			t.Errorf("pgTimestamp(%s) = %d, want %d", tt.when, got, tt.want)
		}
	}
}

// A timestamp with an offset must convert to the same instant as its UTC
// equivalent, not to the wall-clock reading.
func TestBinaryTimestampNormalisesZone(t *testing.T) {
	zone := time.FixedZone("UTC+5", 5*3600)
	local := time.Date(2026, 7, 22, 15, 0, 0, 0, zone)
	utc := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	if pgTimestamp(local) != pgTimestamp(utc) {
		t.Errorf("zone offset was not normalised: %d != %d",
			pgTimestamp(local), pgTimestamp(utc))
	}
}

// Binary COPY carries no type names: Postgres reads each field's bytes as
// whatever the target column is. A value the writer cannot encode must fail
// here rather than produce bytes the server misreads as a valid value.
func TestBinaryRejectsUnsupportedType(t *testing.T) {
	var buf bytes.Buffer
	w := NewBinary(&buf, []string{"v"})
	err := w.WriteRow(map[string]any{"v": []string{"a", "b"}})
	if err == nil {
		t.Fatal("expected an error for an unsupported type")
	}
}

func TestBinaryMissingKeyIsNull(t *testing.T) {
	var buf bytes.Buffer
	w := NewBinary(&buf, []string{"a"})
	if err := w.WriteRow(map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	body := buf.Bytes()[len(wantHeader) : buf.Len()-2]
	want := []byte{0, 1, 0xFF, 0xFF, 0xFF, 0xFF}
	if !bytes.Equal(body, want) {
		t.Errorf("row = % x, want % x", body, want)
	}
}
