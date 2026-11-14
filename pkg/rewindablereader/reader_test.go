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
	require.ErrorIs(t, err, rewindablereader.ErrMaxRecordedSizeExceeded)
	require.EqualError(t, err, "max recorded size exceeded")
	require.Zero(t, n)
}

func TestReaderCustomMaxRecordedSize(t *testing.T) {
	// data up to the custom limit is recorded and can be rewinded
	r := &rewindablereader.Reader{
		R:               bytes.NewReader(make([]byte, 2048)),
		MaxRecordedSize: 2048,
	}

	buf := make([]byte, 2048)
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 2048, n)

	r.Rewind()

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Len(t, out, 2048)

	// exceeding the custom limit returns the same diagnostic error
	r = &rewindablereader.Reader{
		R:               bytes.NewReader(make([]byte, 2049)),
		MaxRecordedSize: 2048,
	}

	buf = make([]byte, 2049)
	n, err = r.Read(buf)
	require.ErrorIs(t, err, rewindablereader.ErrMaxRecordedSizeExceeded)
	require.EqualError(t, err, "max recorded size exceeded")
	require.Zero(t, n)
}

func TestReaderMaxRecordedSizeNoTruncationOrReplay(t *testing.T) {
	// a read that crosses the limit delivers nothing: no partial bytes are
	// recorded, so nothing can be silently truncated or delivered twice
	src := make([]byte, 2048)
	for i := range src {
		src[i] = byte(i + 1)
	}

	r := &rewindablereader.Reader{
		R:               bytes.NewReader(src),
		MaxRecordedSize: 1024,
	}

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 1024, n)

	// the offending read returns zero bytes and keeps failing
	buf = make([]byte, 8)
	n, err = r.Read(buf)
	require.ErrorIs(t, err, rewindablereader.ErrMaxRecordedSizeExceeded)
	require.Zero(t, n)

	n, err = r.Read(buf)
	require.ErrorIs(t, err, rewindablereader.ErrMaxRecordedSizeExceeded)
	require.Zero(t, n)

	// rewind replays exactly the bytes recorded before the error, in order
	r.Rewind()
	replayed := make([]byte, 1024)
	n, err = io.ReadFull(r, replayed)
	require.NoError(t, err)
	require.Equal(t, 1024, n)
	require.Equal(t, src[:1024], replayed)

	// the next read falls through to the underlying reader instead of
	// looping over recorded data: it must return fresh bytes, not the
	// recorded ones again (the failed read already advanced the underlying
	// reader by len(buf), which is the pre-existing reader behavior)
	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 8, n)
	require.Equal(t, src[1024+16:1024+24], buf)
}

func TestReaderZeroMaxRecordedSizeUsesDefault(t *testing.T) {
	for _, maxSize := range []int{0, -1} {
		r := &rewindablereader.Reader{
			R:               bytes.NewReader(make([]byte, rewindablereader.MaxRecordedSize+1)),
			MaxRecordedSize: maxSize,
		}

		buf := make([]byte, rewindablereader.MaxRecordedSize)
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, rewindablereader.MaxRecordedSize, n)

		n, err = r.Read(buf)
		require.ErrorIs(t, err, rewindablereader.ErrMaxRecordedSizeExceeded)
		require.Zero(t, n)
	}
}

func TestReaderRewindConsistency(t *testing.T) {
	// larger than the default limit, with a content pattern that makes any
	// duplicated, missing or reordered byte detectable
	size := 2 * 1024 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte((i * 7) >> 3)
	}

	r := &rewindablereader.Reader{
		R:               bytes.NewReader(data),
		MaxRecordedSize: size,
	}

	// read through using small and irregular buffer sizes
	first := make([]byte, 0, size)
	bufSizes := []int{1, 7, 188, 4096, 65537}
	read := 0
	for read < size {
		buf := make([]byte, bufSizes[read%len(bufSizes)])
		n, err := r.Read(buf)
		require.NoError(t, err)
		first = append(first, buf[:n]...)
		read += n
	}
	require.Equal(t, data, first)

	r.Rewind()

	// after rewind the same bytes must be delivered again, in the same order
	second, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, data, second)

	// once the recorded data is exhausted the underlying reader is at EOF
	n, err := r.Read(make([]byte, 16))
	require.ErrorIs(t, err, io.EOF)
	require.Zero(t, n)
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
