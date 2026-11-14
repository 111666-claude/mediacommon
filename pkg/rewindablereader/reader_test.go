package rewindablereader_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/rewindablereader"
)

type emptyThenReader struct {
	empty int
	data  io.Reader
}

func (r *emptyThenReader) Read(buf []byte) (int, error) {
	if r.empty > 0 {
		r.empty--
		return 0, nil
	}

	return r.data.Read(buf)
}

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

func TestReaderEmptyReads(t *testing.T) {
	r := &rewindablereader.Reader{R: &emptyThenReader{
		empty: 100000,
		data:  bytes.NewReader([]byte{1, 2, 3, 4}),
	}}

	buf := make([]byte, 4)
	for range 100000 {
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Zero(t, n)
	}

	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	n, err = r.Read(buf)
	require.ErrorIs(t, err, io.EOF)
	require.Zero(t, n)

	r.Rewind()
	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	n, err = r.Read(buf)
	require.ErrorIs(t, err, io.EOF)
	require.Zero(t, n)
}

func TestReaderMaxRecordedSize(t *testing.T) {
	r := &rewindablereader.Reader{R: bytes.NewReader(make([]byte, 1024*1024+1))}

	buf := make([]byte, 1024*1024)
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, len(buf), n)

	n, err = r.Read(buf)
	require.EqualError(t, err, "max recorded size exceeded")
	require.Zero(t, n)
}

func TestReaderCustomMaxRecordedSize(t *testing.T) {
	r := &rewindablereader.Reader{
		R:               bytes.NewReader(make([]byte, 10)),
		MaxRecordedSize: 8,
	}

	buf := make([]byte, 4)

	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)

	// exactly at the custom limit: still allowed
	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)

	// past the custom limit: error
	n, err = r.Read(buf)
	require.EqualError(t, err, "max recorded size exceeded")
	require.Zero(t, n)
}

func TestReaderRewindConsistency(t *testing.T) {
	data := make([]byte, 2*1024*1024)
	for i := range data {
		data[i] = byte(i)
	}

	r := &rewindablereader.Reader{
		R:               bytes.NewReader(data),
		MaxRecordedSize: 3 * 1024 * 1024,
	}

	// record the first 1.5MB
	recorded := make([]byte, 1536*1024)
	_, err := io.ReadFull(r, recorded)
	require.NoError(t, err)
	require.Equal(t, data[:len(recorded)], recorded)

	r.Rewind()

	// data read after the rewind must be identical to the original input
	all := make([]byte, len(data))
	_, err = io.ReadFull(r, all)
	require.NoError(t, err)
	require.Equal(t, data, all)
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
