package fmp4

import (
	"testing"

	amp4 "github.com/abema/go-mp4"
	"github.com/stretchr/testify/require"

	imp4 "github.com/bluenviron/mediacommon/v2/internal/mp4"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/fmp4/seekablebuffer"
)

// buildPartWithExplicitSizes encodes a single-track part whose trun carries
// explicit sample sizes, like Part.Marshal does.
func buildPartWithExplicitSizes(t *testing.T, sizes []uint32, payload []byte) []byte {
	t.Helper()

	var buf seekablebuffer.Buffer
	w := &imp4.Writer{W: &buf}
	w.Initialize()

	moofOffset, err := w.WriteBoxStart(&amp4.Moof{}) // <moof>
	require.NoError(t, err)

	_, err = w.WriteBox(&amp4.Mfhd{ // <mfhd/>
		SequenceNumber: 1,
	})
	require.NoError(t, err)

	_, err = w.WriteBoxStart(&amp4.Traf{}) // <traf>
	require.NoError(t, err)

	_, err = w.WriteBox(&amp4.Tfhd{ // <tfhd/>
		FullBox: amp4.FullBox{
			Flags: [3]byte{0x02, 0x00, 0x00},
		},
		TrackID: 1,
	})
	require.NoError(t, err)

	_, err = w.WriteBox(&amp4.Tfdt{ // <tfdt/>
		FullBox: amp4.FullBox{
			Version: 1,
		},
		BaseMediaDecodeTimeV1: 0,
	})
	require.NoError(t, err)

	trunFlags := trunFlagDataOffsetPreset |
		trunFlagSampleDurationPresent |
		trunFlagSampleSizePresent

	trun := &amp4.Trun{ // <trun/>
		FullBox: amp4.FullBox{
			Version: 1,
			Flags:   [3]byte{0, byte(trunFlags >> 8), byte(trunFlags)},
		},
		SampleCount: uint32(len(sizes)),
	}
	for _, size := range sizes {
		trun.Entries = append(trun.Entries, amp4.TrunEntry{
			SampleDuration: 30,
			SampleSize:     size,
		})
	}

	trunOffset, err := w.WriteBox(trun)
	require.NoError(t, err)

	require.NoError(t, w.WriteBoxEnd()) // </traf>
	require.NoError(t, w.WriteBoxEnd()) // </moof>

	mdatOffset, err := w.WriteBox(&amp4.Mdat{Data: payload}) // <mdat/>
	require.NoError(t, err)

	trun.DataOffset = int32(mdatOffset - moofOffset + 8)
	require.NoError(t, w.RewriteBox(trunOffset, trun))

	return buf.Bytes()
}

// buildPartWithDefaultSize encodes a single-track part in which the trun does
// not contain sample sizes and the size comes from the tfhd
// DefaultSampleSize field instead.
func buildPartWithDefaultSize(t *testing.T, defaultSize uint32, payload []byte) []byte {
	t.Helper()

	var buf seekablebuffer.Buffer
	w := &imp4.Writer{W: &buf}
	w.Initialize()

	moofOffset, err := w.WriteBoxStart(&amp4.Moof{}) // <moof>
	require.NoError(t, err)

	_, err = w.WriteBox(&amp4.Mfhd{ // <mfhd/>
		SequenceNumber: 1,
	})
	require.NoError(t, err)

	_, err = w.WriteBoxStart(&amp4.Traf{}) // <traf>
	require.NoError(t, err)

	_, err = w.WriteBox(&amp4.Tfhd{ // <tfhd/>
		FullBox: amp4.FullBox{
			Flags: [3]byte{0x02, 0x00, amp4.TfhdDefaultSampleSizePresent},
		},
		TrackID:           1,
		DefaultSampleSize: defaultSize,
	})
	require.NoError(t, err)

	_, err = w.WriteBox(&amp4.Tfdt{ // <tfdt/>
		FullBox: amp4.FullBox{
			Version: 1,
		},
		BaseMediaDecodeTimeV1: 0,
	})
	require.NoError(t, err)

	trunFlags := trunFlagDataOffsetPreset | trunFlagSampleDurationPresent

	trun := &amp4.Trun{ // <trun/>
		FullBox: amp4.FullBox{
			Version: 1,
			Flags:   [3]byte{0, byte(trunFlags >> 8), byte(trunFlags)},
		},
		SampleCount: 1,
		Entries: []amp4.TrunEntry{{
			SampleDuration: 30,
		}},
	}

	trunOffset, err := w.WriteBox(trun)
	require.NoError(t, err)

	require.NoError(t, w.WriteBoxEnd()) // </traf>
	require.NoError(t, w.WriteBoxEnd()) // </moof>

	mdatOffset, err := w.WriteBox(&amp4.Mdat{Data: payload}) // <mdat/>
	require.NoError(t, err)

	trun.DataOffset = int32(mdatOffset - moofOffset + 8)
	require.NoError(t, w.RewriteBox(trunOffset, trun))

	return buf.Bytes()
}

func TestPartsUnmarshalZeroExplicitSampleSize(t *testing.T) {
	// one valid sample of 2 bytes, followed by a sample whose explicit
	// trun size has been corrupted to zero
	enc := buildPartWithExplicitSizes(t, []uint32{2, 0}, []byte{0x01, 0x02})

	var parts Parts
	err := parts.Unmarshal(enc)
	require.EqualError(t, err, "invalid sample size")
	require.Empty(t, parts)
}

func TestPartsUnmarshalZeroDefaultSampleSize(t *testing.T) {
	// the trun carries no sample size; the tfhd default size is zero
	enc := buildPartWithDefaultSize(t, 0, []byte{0x01, 0x02})

	var parts Parts
	err := parts.Unmarshal(enc)
	require.EqualError(t, err, "invalid sample size")
	require.Empty(t, parts)
}

func TestPartsUnmarshalTruncatedSampleSize(t *testing.T) {
	// declared size is larger than the bytes left in the mdat payload
	enc := buildPartWithExplicitSizes(t, []uint32{100}, []byte{0x01, 0x02})

	var parts Parts
	err := parts.Unmarshal(enc)
	require.EqualError(t, err, "invalid sample size")
	require.Empty(t, parts)
}

func TestPartsUnmarshalValidSampleSizes(t *testing.T) {
	// single and multi-sample parts keep being decoded with the original
	// offsets and payloads
	for _, ca := range []struct {
		name    string
		sizes   []uint32
		payload []byte
	}{
		{"single sample", []uint32{2}, []byte{0x01, 0x02}},
		{"multiple samples", []uint32{2, 3, 1}, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}},
	} {
		t.Run(ca.name, func(t *testing.T) {
			enc := buildPartWithExplicitSizes(t, ca.sizes, ca.payload)

			var parts Parts
			err := parts.Unmarshal(enc)
			require.NoError(t, err)
			require.Len(t, parts, 1)
			require.Len(t, parts[0].Tracks, 1)
			require.Len(t, parts[0].Tracks[0].Samples, len(ca.sizes))

			pos := 0
			for i, size := range ca.sizes {
				sample := parts[0].Tracks[0].Samples[i]
				require.Equal(t, size, uint32(len(sample.Payload)))
				require.Equal(t, ca.payload[pos:pos+int(size)], sample.Payload)
				pos += int(size)
			}
		})
	}
}
