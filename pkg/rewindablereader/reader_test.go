package rewindablereader_test

import (
	"bytes"
	"io"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/rewindablereader"
)

// emptyThenReader returns n empty reads before delegating to data.
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

// emptyThenChunksReader returns n empty reads, then count chunks of data.
type emptyThenChunksReader struct {
	empty  int
	chunk  []byte
	count  int
	offset int
}

func (r *emptyThenChunksReader) Read(buf []byte) (int, error) {
	if r.empty > 0 {
		r.empty--
		return 0, nil
	}

	if r.count == 0 {
		return 0, io.EOF
	}

	n := copy(buf, r.chunk[r.offset:])
	r.offset += n
	if r.offset == len(r.chunk) {
		r.offset = 0
		r.count--
	}
	return n, nil
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

// countingReader counts how many times the underlying reader is called.
type countingReader struct {
	reads int
	data  io.Reader
}

func (r *countingReader) Read(buf []byte) (int, error) {
	r.reads++
	return r.data.Read(buf)
}

// dataWithError returns n bytes together with an error.
type dataWithError struct {
	data []byte
	err  error
}

func (r *dataWithError) Read(buf []byte) (int, error) {
	n := copy(buf, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 {
		return n, r.err
	}
	return n, nil
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

func TestReaderEmptyReadsDoNotAffectReplay(t *testing.T) {
	// empty reads interleaved with data must not produce empty entries:
	// a replay with smaller buffers must return data straight away.
	r := &rewindablereader.Reader{R: &emptyThenChunksReader{
		empty: 100000,
		chunk: []byte{1, 2, 3, 4},
		count: 1,
	}}

	buf := make([]byte, 4)
	for range 100000 {
		_, err := r.Read(buf)
		require.NoError(t, err)
	}

	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	r.Rewind()

	small := make([]byte, 1)
	for _, expected := range []byte{1, 2, 3, 4} {
		n, err = r.Read(small)
		require.NoError(t, err)
		require.Equal(t, []byte{expected}, small[:n])
	}
}

func TestReaderEmptyReadsAreCheap(t *testing.T) {
	// reading empty data must not allocate (and therefore must not record
	// empty entries), regardless of how many empty reads occur.
	r := &rewindablereader.Reader{R: &emptyThenReader{empty: 1 << 30, data: bytes.NewReader(nil)}}
	buf := make([]byte, 1024)

	allocs := testing.AllocsPerRun(100, func() {
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Zero(t, n)
	})
	require.Zero(t, allocs)
}

func TestReaderEmptyReadsLinearity(t *testing.T) {
	const count = 6000

	r := &rewindablereader.Reader{R: &emptyThenChunksReader{
		empty: count,
		chunk: []byte{1, 2, 3, 4},
		count: count,
	}}

	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	buf := make([]byte, 4)
	for range count {
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Zero(t, n)
	}

	expected := make([]byte, 0, count*4)
	for range count {
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 4, n)
		expected = append(expected, buf[:n]...)
	}

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	// a linear implementation allocates about the size of the recorded data.
	// a quadratic one (e.g. re-copying the whole cache on each read) would
	// allocate orders of magnitude more.
	require.Less(t, after.TotalAlloc-before.TotalAlloc, uint64(count*4*100))

	r.Rewind()

	replayed := make([]byte, 0, len(expected))
	for len(replayed) < len(expected) {
		n, err := r.Read(buf)
		require.NoError(t, err)
		replayed = append(replayed, buf[:n]...)
	}
	require.Equal(t, expected, replayed)
}

func TestReaderMaxRecordedSize(t *testing.T) {
	r := &rewindablereader.Reader{R: bytes.NewReader(make([]byte, 1024*1024+1))}

	buf := make([]byte, 1024*1024)
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, len(buf), n)

	n, err = r.Read(make([]byte, 1))
	require.EqualError(t, err, "max recorded size exceeded")
	require.Zero(t, n)

	// data recorded before the error is still available after a rewind.
	r.Rewind()

	replayed := 0
	for {
		n, err = r.Read(buf)
		replayed += n
		if err != nil {
			break
		}
	}
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, 1024*1024, replayed)
}

func TestReaderMaxRecordedSizeNotPartiallyStored(t *testing.T) {
	// when the limit is exceeded, the overflowing chunk must not be recorded,
	// not even partially.
	r := &rewindablereader.Reader{
		R:       bytes.NewReader([]byte{1, 2, 3, 4, 5, 6}),
		MaxSize: 4,
	}

	buf := make([]byte, 4)
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	n, err = r.Read(buf)
	require.EqualError(t, err, "max recorded size exceeded")
	require.Zero(t, n)

	r.Rewind()

	small := make([]byte, 2)
	n, err = r.Read(small)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2}, small[:n])

	n, err = r.Read(small)
	require.NoError(t, err)
	require.Equal(t, []byte{3, 4}, small[:n])

	n, err = r.Read(small)
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

func TestReaderReadWithError(t *testing.T) {
	r := &rewindablereader.Reader{R: &dataWithError{
		data: []byte{1, 2, 3, 4},
		err:  io.ErrUnexpectedEOF,
	}}

	buf := make([]byte, 4)
	n, err := r.Read(buf)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	r.Rewind()
	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])
}

func TestReaderEOF(t *testing.T) {
	cr := &countingReader{data: bytes.NewReader(nil)}
	r := &rewindablereader.Reader{R: cr}

	buf := make([]byte, 4)
	n, err := r.Read(buf)
	require.ErrorIs(t, err, io.EOF)
	require.Zero(t, n)

	r.Rewind()
	n, err = r.Read(buf)
	require.ErrorIs(t, err, io.EOF)
	require.Zero(t, n)
}

func TestReaderRepeatedRewind(t *testing.T) {
	cr := &countingReader{data: bytes.NewReader([]byte{1, 2, 3, 4})}
	r := &rewindablereader.Reader{R: cr}

	buf := make([]byte, 4)
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	for range 3 {
		r.Rewind()

		n, err = r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

		// the underlying reader is not read again while replaying.
		require.Equal(t, 1, cr.reads)
	}

	// rewind is idempotent even without reads in between.
	r.Rewind()
	r.Rewind()
	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])
	require.Equal(t, 1, cr.reads)
}

func TestReaderPartialReplayThenRewind(t *testing.T) {
	r := &rewindablereader.Reader{R: &dummyReader2{}}

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{5, 6}, buf[:n])

	small := make([]byte, 2)

	// consume only the first entry of the replay, then rewind again.
	r.Rewind()
	n, err = r.Read(small)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2}, small[:n])

	r.Rewind()
	n, err = r.Read(small)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2}, small[:n])

	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{3, 4}, buf[:n])

	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{5, 6}, buf[:n])

	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{7, 8}, buf[:n])
}
