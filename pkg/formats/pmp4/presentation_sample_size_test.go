package pmp4

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/formats/mp4/codecs"
)

// buildPresentationBytes marshals a single-track presentation containing the
// given payloads, so that the resulting MP4 is well formed end to end.
func buildPresentationBytes(t *testing.T, payloads [][]byte) []byte {
	t.Helper()

	samples := make([]*Sample, len(payloads))
	for i, payload := range payloads {
		samples[i] = &Sample{
			Duration:    90000,
			PayloadSize: uint32(len(payload)),
			GetPayload: func() ([]byte, error) {
				return payload, nil
			},
		}
	}

	p := Presentation{
		Tracks: []*Track{{
			ID:        1,
			TimeScale: 90000,
			Codec: &codecs.Opus{
				ChannelCount: 2,
			},
			Samples: samples,
		}},
	}

	var buf bytes.Buffer
	err := p.Marshal(&buf)
	require.NoError(t, err)

	return buf.Bytes()
}

// patchStszEntrySize overwrites the size of the sample at sampleIndex inside
// the stsz box with the given value. The stsz layout written by this package
// is: box header, full box header, SampleSize(4), SampleCount(4), then one
// 32-bit entry per sample.
func patchStszEntrySize(t *testing.T, enc []byte, sampleIndex int, size uint32) []byte {
	t.Helper()

	out := append([]byte(nil), enc...)

	idx := bytes.Index(out, []byte{'s', 't', 's', 'z'})
	require.NotEqual(t, -1, idx)

	entryOffset := idx + 4 /* box type */ + 4 /* full box */ + 4 /* SampleSize */ +
		4 /* SampleCount */ + sampleIndex*4
	require.LessOrEqual(t, entryOffset+4, len(out))

	out[entryOffset] = byte(size >> 24)
	out[entryOffset+1] = byte(size >> 16)
	out[entryOffset+2] = byte(size >> 8)
	out[entryOffset+3] = byte(size)

	return out
}

func TestPresentationUnmarshalZeroSampleSize(t *testing.T) {
	// a stsz entry corrupted to zero must be rejected instead of producing
	// an empty sample and shifting the following chunk offsets
	enc := patchStszEntrySize(
		t,
		buildPresentationBytes(t, [][]byte{{0x01, 0x02}, {0x03, 0x04}}),
		1,
		0,
	)

	var p Presentation
	err := p.Unmarshal(bytes.NewReader(enc))
	require.EqualError(t, err, "invalid sample size")
	require.Empty(t, p.Tracks)
}

func TestPresentationUnmarshalTruncatedSampleSize(t *testing.T) {
	// a declared size larger than the remaining mdat bytes keeps failing
	// when the payload is read (io.ReadFull returns unexpected EOF); the
	// zero-size rejection must not shadow this truncation behavior
	enc := patchStszEntrySize(
		t,
		buildPresentationBytes(t, [][]byte{{0x01, 0x02}}),
		0,
		1000,
	)

	var p Presentation
	err := p.Unmarshal(bytes.NewReader(enc))
	require.NoError(t, err)
	require.Len(t, p.Tracks, 1)
	require.Len(t, p.Tracks[0].Samples, 1)

	_, err = p.Tracks[0].Samples[0].GetPayload()
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestPresentationUnmarshalValidSampleSizes(t *testing.T) {
	for _, ca := range []struct {
		name     string
		payloads [][]byte
	}{
		{
			"single sample",
			[][]byte{{0x01, 0x02}},
		},
		{
			"multiple samples",
			[][]byte{{0x01, 0x02}, {0x03, 0x04}, {0x05}},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			enc := buildPresentationBytes(t, ca.payloads)

			var p Presentation
			err := p.Unmarshal(bytes.NewReader(enc))
			require.NoError(t, err)
			require.Len(t, p.Tracks, 1)
			require.Len(t, p.Tracks[0].Samples, len(ca.payloads))

			for i, payload := range ca.payloads {
				sample := p.Tracks[0].Samples[i]
				require.Equal(t, uint32(len(payload)), sample.PayloadSize)
				var got []byte
				got, err = sample.GetPayload()
				require.NoError(t, err)
				require.Equal(t, payload, got)
			}
		})
	}
}

func TestPresentationUnmarshalValidMultipleTracks(t *testing.T) {
	payloads := [][]byte{{0x01, 0x02}, {0x03, 0x04}}

	p := Presentation{
		Tracks: []*Track{
			{
				ID:        1,
				TimeScale: 90000,
				Codec:     &codecs.Opus{ChannelCount: 2},
				Samples: []*Sample{{
					Duration:    90000,
					PayloadSize: 2,
					GetPayload: func() ([]byte, error) {
						return payloads[0], nil
					},
				}},
			},
			{
				ID:        2,
				TimeScale: 90000,
				Codec:     &codecs.Opus{ChannelCount: 2},
				Samples: []*Sample{{
					Duration:    90000,
					PayloadSize: 2,
					GetPayload: func() ([]byte, error) {
						return payloads[1], nil
					},
				}},
			},
		},
	}

	var buf bytes.Buffer
	err := p.Marshal(&buf)
	require.NoError(t, err)

	var decoded Presentation
	err = decoded.Unmarshal(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Len(t, decoded.Tracks, 2)

	for trackIndex, payload := range payloads {
		require.Len(t, decoded.Tracks[trackIndex].Samples, 1)
		sample := decoded.Tracks[trackIndex].Samples[0]
		require.Equal(t, uint32(len(payload)), sample.PayloadSize)
		var got []byte
		got, err = sample.GetPayload()
		require.NoError(t, err)
		require.Equal(t, payload, got)
	}
}
