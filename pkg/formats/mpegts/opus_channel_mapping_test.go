package mpegts

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/opus"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/codecs"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/substructs"
)

func TestOpusMultichannelRoundTrip(t *testing.T) {
	// Build an explicit 5.1 OpusHead-style mapping and convert it into the
	// descriptor that the writer puts in the PMT.
	mapping, err := opus.NewChannelMapping(4, 2, []uint8{0, 4, 1, 2, 3, 5})
	require.NoError(t, err)

	desc, err := substructs.NewOpusAudioDescriptorFromChannelMapping(mapping)
	require.NoError(t, err)
	require.Equal(t, uint8(0x81), desc.ChannelConfigCode)

	track := &Track{
		Codec: &codecs.Opus{
			Desc:         desc,
			ChannelCount: 6, //nolint:staticcheck
		},
	}

	var buf bytes.Buffer
	w := NewWriter(&buf, []*Track{track})

	err = w.WriteOpus(track, 10*90000, [][]byte{{1, 2, 3}, {4}})
	require.NoError(t, err)

	err = w.WriteOpus(track, 12*90000, [][]byte{{5, 6}})
	require.NoError(t, err)

	r := &Reader{R: bytes.NewReader(buf.Bytes())}
	err = r.Initialize()
	require.NoError(t, err)

	tracks := r.Tracks()
	require.Len(t, tracks, 1)

	readCodec, ok := tracks[0].Codec.(*codecs.Opus)
	require.True(t, ok)
	require.Equal(t, 6, readCodec.Desc.ChannelCount())

	readMapping, err := readCodec.Desc.GetChannelMapping()
	require.NoError(t, err)
	require.Equal(t, mapping, readMapping)

	gotSamples := 0
	r.OnDataOpus(tracks[0], func(_ int64, frames [][]byte) error {
		require.NotEmpty(t, frames)
		gotSamples++
		return nil
	})

	for {
		err = r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, 2, gotSamples)
}

func TestOpusMonoStereoRoundTrip(t *testing.T) {
	for _, ca := range []struct {
		name string
		h    *opus.IDHeader
		code uint8
	}{
		{
			"mono",
			&opus.IDHeader{Version: 1, ChannelCount: 1, ChannelMappingFamily: 0},
			0x01,
		},
		{
			"stereo",
			&opus.IDHeader{Version: 1, ChannelCount: 2, ChannelMappingFamily: 0},
			0x02,
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			desc, err := substructs.NewOpusAudioDescriptorFromIDHeader(ca.h)
			require.NoError(t, err)
			require.Equal(t, ca.code, desc.ChannelConfigCode)

			// A predefined code round-trips into the same family 0 mapping.
			m, err := desc.GetChannelMapping()
			require.NoError(t, err)
			require.Equal(t, uint8(0), m.Family)
			require.Equal(t, int(ca.h.ChannelCount), len(m.Layout))
		})
	}
}
