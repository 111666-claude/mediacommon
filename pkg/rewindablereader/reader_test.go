package rewindablereader_test

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/rewindablereader"
)

const testMaxRecordedSize = 1 * 1024 * 1024

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

// emptyReadReader performs emptyLeft empty reads before returning data.
type emptyReadReader struct {
	emptyLeft int
	data      []byte
	reads     int
}

func (r *emptyReadReader) Read(buf []byte) (int, error) {
	r.reads++
	if r.emptyLeft > 0 {
		r.emptyLeft--
		return 0, nil
	}
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(buf, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestReaderEmptyReads(t *testing.T) {
	// Anchor case: 100000 empty reads followed by four bytes. Empty reads must
	// not be recorded, so after Rewind the four bytes are returned directly.
	base := &emptyReadReader{
		emptyLeft: 100000,
		data:      []byte{1, 2, 3, 4},
	}
	r := &rewindablereader.Reader{R: base}

	buf := make([]byte, 4)

	// drain the empty reads
	nTotal := 0
	for {
		n, err := r.Read(buf)
		require.NoError(t, err)
		nTotal += n
		if n > 0 {
			require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])
			break
		}
	}
	require.Equal(t, 4, nTotal)
	require.Equal(t, 100001, base.reads)

	// Rewind: no empty reads must be replayed, the data comes in one read.
	r.Rewind()
	// calling Rewind again is idempotent and must not alter the replay state
	r.Rewind()
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])
	// the underlying reader is not touched while recorded data is replayed
	require.Equal(t, 100001, base.reads)

	// once the recorded data is exhausted, reads pass through to the
	// underlying reader instead of looping over stale entries
	n, err = r.Read(buf)
	require.Equal(t, io.EOF, err)
	require.Equal(t, 0, n)
	require.Equal(t, 100002, base.reads)
}

func TestReaderEmptyReadsDifferentBufSize(t *testing.T) {
	base := &emptyReadReader{
		emptyLeft: 10,
		data:      []byte{1, 2, 3, 4},
	}
	r := &rewindablereader.Reader{R: base}

	buf := make([]byte, 1024)
	nTotal := 0
	for {
		n, err := r.Read(buf)
		require.NoError(t, err)
		nTotal += n
		if nTotal == 4 {
			break
		}
	}
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:nTotal])

	r.Rewind()

	// replay with a smaller buffer must split the single recorded entry
	small := make([]byte, 2)
	out := make([]byte, 0, 4)
	for len(out) < 4 {
		n, err := r.Read(small)
		require.NoError(t, err)
		require.Equal(t, 2, n)
		out = append(out, small[:n]...)
	}
	require.Equal(t, []byte{1, 2, 3, 4}, out)
	// still no underlying read during replay
	require.Equal(t, 11, base.reads)
}

// chunkReader returns one byte at a time.
type chunkReader struct {
	data []byte
}

func (r *chunkReader) Read(buf []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	buf[0] = r.data[0]
	r.data = r.data[1:]
	return 1, nil
}

// fillReader fills the whole requested buffer at every read.
type fillReader struct {
	data []byte
}

func (r *fillReader) Read(buf []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(buf, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestReaderMaxRecordedSize(t *testing.T) {
	r := &rewindablereader.Reader{R: &chunkReader{data: make([]byte, testMaxRecordedSize+1)}}

	buf := make([]byte, 1)
	out := make([]byte, 0, testMaxRecordedSize)
	for range testMaxRecordedSize {
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 1, n)
		out = append(out, buf[0])
	}
	require.Len(t, out, testMaxRecordedSize)

	// one more byte exceeds the limit and must not produce a partial cache
	n, err := r.Read(buf)
	require.EqualError(t, err, "max recorded size exceeded")
	require.Equal(t, 0, n)

	// the data recorded before the error is still fully readable after Rewind
	r.Rewind()
	replayed := make([]byte, 0, testMaxRecordedSize)
	for len(replayed) < testMaxRecordedSize {
		n, err = r.Read(buf)
		require.NoError(t, err)
		replayed = append(replayed, buf[:n]...)
	}
	require.Len(t, replayed, testMaxRecordedSize)
	require.Equal(t, out, replayed)
}

func TestReaderMaxRecordedSizeLargeRead(t *testing.T) {
	// a single read that crosses the limit must fail without partial recording
	r := &rewindablereader.Reader{
		R: &fillReader{data: make([]byte, testMaxRecordedSize+64)},
	}

	// fill exactly the limit, one byte per read
	one := make([]byte, 1)
	for range testMaxRecordedSize {
		_, err := r.Read(one)
		require.NoError(t, err)
	}

	// a large read pulls 64 bytes from the underlying reader but must be
	// rejected as a whole, without recording any of them
	big := make([]byte, 64)
	n, err := r.Read(big)
	require.EqualError(t, err, "max recorded size exceeded")
	require.Equal(t, 0, n)

	// rewind still replays exactly the previously recorded MiB
	r.Rewind()
	count := 0
	for {
		n, err := r.Read(big)
		require.NoError(t, err)
		count += n
		if count == testMaxRecordedSize {
			break
		}
	}
}

// nWithErrReader returns data together with a non-EOF error.
type nWithErrReader struct {
	err error
}

func (r *nWithErrReader) Read(buf []byte) (int, error) {
	copy(buf, []byte{1, 2, 3, 4})
	return 4, r.err
}

func TestReaderNWithError(t *testing.T) {
	errTest := errors.New("test error")
	r := &rewindablereader.Reader{R: &nWithErrReader{err: errTest}}

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	require.Equal(t, errTest, err)
	require.Equal(t, 4, n)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	// n>0 reads are recorded even when an error is returned
	r.Rewind()
	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])
}

// eofReader returns EOF directly, with no data.
type eofReader struct {
	reads int
}

func (r *eofReader) Read(_ []byte) (int, error) {
	r.reads++
	return 0, io.EOF
}

func TestReaderDirectEOF(t *testing.T) {
	base := &eofReader{}
	r := &rewindablereader.Reader{R: base}

	buf := make([]byte, 4)
	n, err := r.Read(buf)
	require.Equal(t, io.EOF, err)
	require.Equal(t, 0, n)

	// EOF with zero bytes is not recorded; after Rewind the EOF comes from the
	// underlying reader again, and repeated queries stay idempotent.
	r.Rewind()
	n, err = r.Read(buf)
	require.Equal(t, io.EOF, err)
	require.Equal(t, 0, n)

	r.Rewind()
	n, err = r.Read(buf)
	require.Equal(t, io.EOF, err)
	require.Equal(t, 0, n)

	require.Equal(t, 3, base.reads)
}

// dataEOFReader returns data together with io.EOF.
type dataEOFReader struct {
	data []byte
}

func (r *dataEOFReader) Read(buf []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(buf, r.data)
	r.data = r.data[n:]
	return n, io.EOF
}

func TestReaderEOFAfterData(t *testing.T) {
	r := &rewindablereader.Reader{R: &dataEOFReader{data: []byte{1, 2, 3, 4}}}

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	require.Equal(t, io.EOF, err)
	require.Equal(t, 4, n)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	// recorded data is replayed without an error; EOF only surfaces once the
	// recorded data is exhausted and the underlying reader is queried again.
	r.Rewind()
	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, []byte{1, 2, 3, 4}, buf[:n])

	n, err = r.Read(buf)
	require.Equal(t, io.EOF, err)
	require.Equal(t, 0, n)
}

func TestReaderLinearWork(t *testing.T) {
	// 6000 empty reads followed by 6000 one-byte chunks: recording must stay
	// linear in the amount of data, not in the number of Read calls.
	const count = 6000
	base := &emptyReadReader{
		emptyLeft: count,
		data:      make([]byte, count),
	}
	for i := range base.data {
		base.data[i] = byte(i)
	}
	r := &rewindablereader.Reader{R: base}

	buf := make([]byte, 1)
	got := make([]byte, 0, count)
	empty := 0
	for len(got) < count {
		n, err := r.Read(buf)
		require.NoError(t, err)
		if n == 0 {
			empty++
			continue
		}
		got = append(got, buf[0])
	}
	require.Equal(t, count, empty)
	require.Len(t, got, count)

	// after Rewind every recorded byte is replayed with no empty reads and no
	// access to the underlying reader
	r.Rewind()
	replayed := make([]byte, 0, count)
	emptyAfterRewind := 0
	for len(replayed) < count {
		n, err := r.Read(buf)
		require.NoError(t, err)
		if n == 0 {
			emptyAfterRewind++
			continue
		}
		replayed = append(replayed, buf[0])
	}
	require.Equal(t, 0, emptyAfterRewind)
	require.Equal(t, got, replayed)
	require.Equal(t, count*2, base.reads)
}

func BenchmarkReaderEmptyReads(b *testing.B) {
	base := &emptyReadReader{
		emptyLeft: b.N,
		data:      []byte{1, 2, 3, 4},
	}
	r := &rewindablereader.Reader{R: base}

	buf := make([]byte, 4)
	for {
		n, _ := r.Read(buf)
		if n > 0 {
			break
		}
	}
}
