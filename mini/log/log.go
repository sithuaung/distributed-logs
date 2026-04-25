// Package log is a tiny in-memory append-only log.
//
// It mirrors the role of internal/log in the parent project, but stripped
// down to the essentials so the *idea* is visible:
//
//   - records go in at the end (Append) and get an offset
//   - records can be read back by offset (Read)
//   - records are never updated or deleted
//
// Things deliberately left out compared to internal/log:
//   - on-disk segments + mmap'd index (see internal/log/{segment,store,index}.go)
//   - protobuf record schema (uses raw []byte instead)
//   - Raft replication (see internal/log/distributed.go)
package log

import (
	"errors"
	"sync"
)

// Record is one entry in the log. Value is opaque bytes — the log
// doesn't care if it's JSON, an image, or a protobuf payment event.
type Record struct {
	Value  []byte `json:"value"`
	Offset uint64 `json:"offset"`
}

// Log is the append-only sequence. A sync.RWMutex is enough here
// because everything lives in a slice; the parent project needs much
// more bookkeeping because data is split across segment files on disk.
type Log struct {
	mu      sync.RWMutex
	records []Record
}

func New() *Log {
	return &Log{}
}

// ErrOffsetOutOfRange tells a consumer it has read past the end of
// the log. In a streaming consumer this is the signal to wait for
// more data, not a real error.
var ErrOffsetOutOfRange = errors.New("offset out of range")

// Append adds value to the end of the log and returns its offset.
// Offsets start at 0 and increase by 1 per record — they are how
// consumers track their position in the stream.
func (l *Log) Append(value []byte) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec := Record{
		Value:  value,
		Offset: uint64(len(l.records)),
	}
	l.records = append(l.records, rec)
	return rec.Offset, nil
}

// Read returns the record at offset, or ErrOffsetOutOfRange if there
// isn't one yet. Older records are never overwritten, so a slow
// consumer can always catch up from where it left off.
func (l *Log) Read(offset uint64) (Record, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if offset >= uint64(len(l.records)) {
		return Record{}, ErrOffsetOutOfRange
	}
	return l.records[offset], nil
}
