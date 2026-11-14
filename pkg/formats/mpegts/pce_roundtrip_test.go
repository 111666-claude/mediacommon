package mpegts

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/codecs"
)

func TestMPEG4AudioChannelConfigZero1000Frames(t *testing.T) {
	track := &Track{
		Codec: &codecs.MPEG4Audio{
			Config: mpeg4audio.AudioSpecificConfig{
				// AAC-LC, 48 kHz, channelConfiguration = 0: the channel
				// layout is carried in-band, not described by the ADTS header.
				Type:         2,
				SampleRate:   48000,
				ChannelCount: 0,
			},
		},
	}

	// 1000 frames, each carrying a distinct access unit
	frames := make([][]byte, 1000)
	adtsPkts := make(mpeg4audio.ADTSPackets, 1000)
	for i := range 1000 {
		au := []byte{byte(i >> 8), byte(i), 0xAA, 0xBB}
		frames[i] = au
		adtsPkts[i] = &mpeg4audio.ADTSPacket{
			Type:         mpeg4audio.ObjectTypeAACLC,
			SampleRate:   48000,
			ChannelCount: 0,
			AU:           au,
		}
	}

	// the raw ADTS stream must keep channelConfiguration = 0 on every frame
	rawADTS, err := adtsPkts.Marshal()
	require.NoError(t, err)

	var decodedADTS mpeg4audio.ADTSPackets
	require.NoError(t, decodedADTS.Unmarshal(rawADTS))
	require.Len(t, decodedADTS, 1000)
	for i, pkt := range decodedADTS {
		require.Equal(t, 0, pkt.ChannelCount, "frame %d", i)
		require.Equal(t, frames[i], pkt.AU, "frame %d", i)
	}

	reEncoded, err := decodedADTS.Marshal()
	require.NoError(t, err)
	require.Equal(t, rawADTS, reEncoded)

	// mux the 1000 frames into MPEG-TS
	var buf bytes.Buffer
	w := NewWriter(&buf, []*Track{track})
	for i := range 1000 {
		err := w.WriteMPEG4Audio(track, int64(i)*1024, [][]byte{frames[i]})
		require.NoError(t, err)
	}

	// demux them back: no frame may be rejected and the in-band channel
	// meaning must be preserved
	r, err := NewReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)

	audioTrack := r.Tracks()[0]
	audioCodec, ok := audioTrack.Codec.(*codecs.MPEG4Audio)
	require.True(t, ok)
	require.Equal(t, 0, audioCodec.ChannelCount)

	count := 0
	r.OnDataMPEG4Audio(audioTrack, func(_ int64, aus [][]byte) error {
		require.Len(t, aus, 1)
		require.Equal(t, frames[count], aus[0], "frame %d", count)
		count++
		return nil
	})

	for {
		err = r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.Equal(t, 1000, count)
}
