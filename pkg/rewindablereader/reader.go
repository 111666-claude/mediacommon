// Package rewindablereader contains a rewindable reader.
package rewindablereader

import (
	"errors"
	"io"
)

// MaxRecordedSize is the maximum amount of data that is recorded
// by a Reader when the Reader.MaxRecordedSize field is not set explicitly.
const MaxRecordedSize = 1 * 1024 * 1024

// ErrMaxRecordedSizeExceeded is returned when the amount of data read
// before Rewind exceeds the configured maximum.
var ErrMaxRecordedSizeExceeded = errors.New("max recorded size exceeded")

// Reader is a reader that can be (and must be) rewinded once.
type Reader struct {
	R io.Reader

	// MaxRecordedSize is the maximum amount of data that can be recorded
	// before Rewind is called. A value of 0 (or a negative value) means
	// that the package-level MaxRecordedSize default is used.
	MaxRecordedSize int

	entries  [][]byte
	size     int
	rewinded bool
}

func (r *Reader) maxRecordedSize() int {
	if r.MaxRecordedSize > 0 {
		return r.MaxRecordedSize
	}
	return MaxRecordedSize
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (int, error) {
	if !r.rewinded {
		n, err := r.R.Read(p)

		if (r.size + n) > r.maxRecordedSize() {
			return 0, ErrMaxRecordedSizeExceeded
		}

		if n > 0 {
			entry := make([]byte, n)
			copy(entry, p[:n])
			r.entries = append(r.entries, entry)
			r.size += n
		}
		return n, err
	}

	if r.entries != nil {
		entry := r.entries[0]
		n := copy(p, entry)

		if n != len(entry) {
			r.entries[0] = entry[n:]
		} else {
			r.entries = r.entries[1:]

			if len(r.entries) == 0 {
				r.entries = nil // release entries from memory
			}
		}

		return n, nil
	}

	return r.R.Read(p)
}

// Rewind rewinds the reader. This can be (and must be) called once.
func (r *Reader) Rewind() {
	r.rewinded = true
}
