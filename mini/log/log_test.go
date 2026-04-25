package log

import (
	"errors"
	"testing"
)

func TestAppendAssignsSequentialOffsets(t *testing.T) {
	l := New()
	for i, v := range [][]byte{[]byte("a"), []byte("b"), []byte("c")} {
		off, err := l.Append(v)
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		if off != uint64(i) {
			t.Fatalf("offset: want %d, got %d", i, off)
		}
	}
}

func TestReadReturnsAppendedValue(t *testing.T) {
	l := New()
	_, _ = l.Append([]byte("first"))
	_, _ = l.Append([]byte("second"))

	rec, err := l.Read(1)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(rec.Value) != "second" {
		t.Fatalf("value: want %q, got %q", "second", rec.Value)
	}
	if rec.Offset != 1 {
		t.Fatalf("offset: want 1, got %d", rec.Offset)
	}
}

func TestReadPastEndReturnsOutOfRange(t *testing.T) {
	l := New()
	_, _ = l.Append([]byte("only"))

	if _, err := l.Read(99); !errors.Is(err, ErrOffsetOutOfRange) {
		t.Fatalf("want ErrOffsetOutOfRange, got %v", err)
	}
}
