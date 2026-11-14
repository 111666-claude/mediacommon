package pmp4

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mp4/codecs"
)

// faultyReadSeeker wraps a ReadSeeker and allows to inject
// seek failures and short reads.
type faultyReadSeeker struct {
	rs io.ReadSeeker

	seekErr  error // when non-nil, every Seek fails with this error
	maxBytes int   // total bytes served by Read before returning io.EOF; -1 = unlimited
	served   int
}

func (f *faultyReadSeeker) Read(p []byte) (int, error) {
	if f.maxBytes >= 0 {
		remaining := f.maxBytes - f.served
		if remaining <= 0 {
			return 0, io.EOF
		}
		if len(p) > remaining {
			p = p[:remaining]
		}
	}

	n, err := f.rs.Read(p)
	f.served += n
	return n, err
}

func (f *faultyReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if f.seekErr != nil {
		return 0, f.seekErr
	}
	return f.rs.Seek(offset, whence)
}

func testLazyPayloadPresentation() Presentation {
	return Presentation{
		Tracks: []*Track{
			{
				ID:        1,
				TimeScale: 90000,
				Codec: &codecs.H264{
					SPS: []byte{ // 1920x1080 baseline
						0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02,
						0x27, 0xe5, 0x84, 0x00, 0x00, 0x03, 0x00, 0x04,
						0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9, 0x20,
					},
					PPS: []byte{0x08, 0x06, 0x07, 0x08},
				},
				Samples: []*Sample{
					{
						Duration:    90000,
						PayloadSize: 4,
						GetPayload: func() ([]byte, error) {
							return []byte{1, 2, 3, 4}, nil
						},
					},
					{
						Duration:    90000,
						PayloadSize: 2,
						GetPayload: func() ([]byte, error) {
							return []byte{5, 6}, nil
						},
					},
				},
			},
			{
				ID:        2,
				TimeScale: 48000,
				Codec: &codecs.MPEG4Audio{
					Config: mpeg4audio.AudioSpecificConfig{
						Type:          2,
						SampleRate:    44100,
						ChannelConfig: 2,
						ChannelCount:  2, //nolint:staticcheck
					},
				},
				Samples: []*Sample{
					{
						Duration:    48000,
						PayloadSize: 3,
						GetPayload: func() ([]byte, error) {
							return []byte{7, 8, 9}, nil
						},
					},
					{
						Duration:    48000,
						PayloadSize: 5,
						GetPayload: func() ([]byte, error) {
							return []byte{10, 11, 12, 13, 14}, nil
						},
					},
				},
			},
		},
	}
}

func unmarshalLazyPresentation(t *testing.T, rs io.ReadSeeker) *Presentation {
	t.Helper()

	var p Presentation
	err := p.Unmarshal(rs)
	require.NoError(t, err)
	require.Len(t, p.Tracks, 2)
	require.Len(t, p.Tracks[0].Samples, 2)
	require.Len(t, p.Tracks[1].Samples, 2)

	return &p
}

func TestPresentationLazyPayloadSeekError(t *testing.T) {
	var buf bytes.Buffer
	err := testLazyPayloadPresentation().Marshal(&buf)
	require.NoError(t, err)

	frs := &faultyReadSeeker{rs: bytes.NewReader(buf.Bytes()), maxBytes: -1}
	p := unmarshalLazyPresentation(t, frs)

	// make the underlying ReadSeeker refuse seeks
	seekErr := errors.New("seek error")
	frs.seekErr = seekErr

	for _, track := range p.Tracks {
		for _, sample := range track.Samples {
			pl, err2 := sample.GetPayload()
			require.ErrorIs(t, err2, seekErr)
			require.Nil(t, pl)
		}
	}

	// after seeks work again, payloads must be read from the declared offsets
	frs.seekErr = nil

	expected := map[int][][]byte{
		1: {{1, 2, 3, 4}, {5, 6}},
		2: {{7, 8, 9}, {10, 11, 12, 13, 14}},
	}

	for _, track := range p.Tracks {
		for i, sample := range track.Samples {
			pl, err2 := sample.GetPayload()
			require.NoError(t, err2)
			require.Equal(t, expected[track.ID][i], pl)
		}
	}
}

func TestPresentationLazyPayloadReadConsistency(t *testing.T) {
	var buf bytes.Buffer
	err := testLazyPayloadPresentation().Marshal(&buf)
	require.NoError(t, err)

	p := unmarshalLazyPresentation(t, bytes.NewReader(buf.Bytes()))

	expected := map[int][][]byte{
		1: {{1, 2, 3, 4}, {5, 6}},
		2: {{7, 8, 9}, {10, 11, 12, 13, 14}},
	}

	// read samples of both tracks in an interleaved order,
	// then read previous samples again: results must not change.
	type sampleRef struct {
		trackID int
		index   int
	}

	trackByID := map[int]*Track{}
	for _, track := range p.Tracks {
		trackByID[track.ID] = track
	}

	for _, ref := range []sampleRef{
		{1, 0}, {2, 0}, {1, 1}, {2, 1}, {1, 0}, {2, 1}, {1, 1}, {2, 0},
	} {
		sample := trackByID[ref.trackID].Samples[ref.index]
		pl, err2 := sample.GetPayload()
		require.NoError(t, err2)
		require.Equal(t, expected[ref.trackID][ref.index], pl)
		require.Len(t, pl, int(sample.PayloadSize))
	}
}

func TestPresentationLazyPayloadShortRead(t *testing.T) {
	var buf bytes.Buffer
	err := testLazyPayloadPresentation().Marshal(&buf)
	require.NoError(t, err)

	frs := &faultyReadSeeker{rs: bytes.NewReader(buf.Bytes()), maxBytes: -1}
	p := unmarshalLazyPresentation(t, frs)

	// a short read (less bytes than PayloadSize, then EOF) must result
	// in an error, not in a truncated payload returned as success.
	for _, track := range p.Tracks {
		for _, sample := range track.Samples {
			frs.maxBytes = int(sample.PayloadSize) - 1
			frs.served = 0

			pl, err2 := sample.GetPayload()
			require.ErrorIs(t, err2, io.ErrUnexpectedEOF)
			require.Nil(t, pl)
		}
	}

	// an immediate EOF must result in an error too.
	frs.maxBytes = 0
	frs.served = 0

	pl, err2 := p.Tracks[0].Samples[0].GetPayload()
	require.ErrorIs(t, err2, io.EOF)
	require.Nil(t, pl)

	// reading past the end of the file must fail as well.
	frs.maxBytes = -1
	frs.rs = bytes.NewReader(buf.Bytes()[:buf.Len()-1])

	pl, err2 = p.Tracks[1].Samples[1].GetPayload()
	require.ErrorIs(t, err2, io.ErrUnexpectedEOF)
	require.Nil(t, pl)
}
