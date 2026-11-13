// Package rewindablereader contains a rewindable reader.
package rewindablereader

import (
	"errors"
	"io"
)

const (
	defaultMaxSize = 1 * 1024 * 1024
)

// Reader is a reader that can be rewinded.
type Reader struct {
	R       io.Reader
	MaxSize int

	entries  [][]byte
	size     int
	rewound  bool
	entryIdx int
	entryOff int
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (int, error) {
	if !r.rewound {
		if r.MaxSize == 0 {
			r.MaxSize = defaultMaxSize
		}

		n, err := r.R.Read(p)

		if (r.size + n) > r.MaxSize {
			return 0, errors.New("max recorded size exceeded")
		}

		// empty reads must not alter the recorded data, its size or the
		// boundaries between recorded entries.
		if n > 0 {
			entry := make([]byte, n)
			copy(entry, p[:n])
			r.entries = append(r.entries, entry)
			r.size += n
		}

		return n, err
	}

	// replay recorded data. The position is kept separately from the entries,
	// so that Rewind can be called any number of times without losing data.
	if r.entryIdx < len(r.entries) {
		entry := r.entries[r.entryIdx][r.entryOff:]
		n := copy(p, entry)

		if n == len(entry) {
			r.entryIdx++
			r.entryOff = 0
		} else {
			r.entryOff += n
		}

		return n, nil
	}

	return r.R.Read(p)
}

// Rewind rewinds the reader, causing recorded data to be read again.
// It can be called any number of times. Once recorded data has been
// replayed, reads continue from the underlying reader, which is never
// re-read implicitly.
func (r *Reader) Rewind() {
	r.rewound = true
	r.entryIdx = 0
	r.entryOff = 0
}
