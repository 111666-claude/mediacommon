package rewindablereader_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/rewindablereader"
)

type dummyReader2 struct {
	i int
}

func (r *dummyReader2) Read(buf []byte) (int, error) {
	r.i++
	switch r.i {
	case 1:
		copy(buf, []byte{1, 2, 3, 4})
		return 4, nil

	case 2:
		copy(buf, []byte{5, 6})
		return 2, nil

	case 3:
		copy(buf, []byte{7, 8})
		return 2, nil
	}
	panic("should not happen")
}

func TestReader(t *testing.T) {
	r := &rewindablereader.Reader{R: &dummyReader2{}}

	for i := range 2 {
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

		n, err = r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, []byte{5, 6}, buf[:n])

		if i == 0 {
			r.Rewind()
		} else {
			n, err = r.Read(buf)
			require.NoError(t, err)
			require.Equal(t, []byte{7, 8}, buf[:n])
		}
	}
}

func TestReaderDifferentBufSize(t *testing.T) {
	r := &rewindablereader.Reader{R: &dummyReader2{}}

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	r.Rewind()

	buf = make([]byte, 2)

	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2}, buf[:n])

	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{3, 4}, buf[:n])
}

// emptyReadsReader returns (0, nil) a given number of times,
// then returns payload, then EOF.
type emptyReadsReader struct {
	emptyReads int
	payload    []byte
	pos        int
}

func (r *emptyReadsReader) Read(buf []byte) (int, error) {
	if r.emptyReads > 0 {
		r.emptyReads--
		return 0, nil
	}

	if r.pos < len(r.payload) {
		n := copy(buf, r.payload[r.pos:])
		r.pos += n
		return n, nil
	}

	return 0, io.EOF
}

func TestReaderEmptyReads(t *testing.T) {
	r := &rewindablereader.Reader{R: &emptyReadsReader{
		emptyReads: 100000,
		payload:    []byte{1, 2, 3, 4},
	}}

	buf := make([]byte, 1024)

	// empty reads must not alter the recorded content.
	for range 100000 {
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 0, n)
	}

	for i := range 2 {
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

		if i == 0 {
			r.Rewind()
		}
	}
}

func TestReaderMaxRecordedSize(t *testing.T) {
	r := &rewindablereader.Reader{R: &emptyReadsReader{
		payload: make([]byte, 1024*1024+1),
	}}

	buf := make([]byte, 1024*1024)
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 1024*1024, n)

	_, err = r.Read(buf)
	require.EqualError(t, err, "max recorded size exceeded")

	// data recorded before the error must still be readable after a rewind.
	r.Rewind()

	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 1024*1024, n)
}

func TestReaderEOF(t *testing.T) {
	r := &rewindablereader.Reader{R: &emptyReadsReader{
		payload: []byte{1, 2},
	}}

	buf := make([]byte, 1024)

	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2}, buf[:n])

	_, err = r.Read(buf)
	require.Equal(t, io.EOF, err)

	r.Rewind()

	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2}, buf[:n])

	// after the recorded data is consumed, reads go to the
	// underlying reader again, which returns EOF.
	_, err = r.Read(buf)
	require.Equal(t, io.EOF, err)
}
