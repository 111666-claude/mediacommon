package h264_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
)

func TestPPSUnmarshal(t *testing.T) {
	t.Run("id 0, sps 0", func(t *testing.T) {
		var p h264.PPS
		err := p.Unmarshal([]byte{0x68, 0xce, 0x3c})
		require.NoError(t, err)
		require.Equal(t, h264.PPS{ID: 0, SPSID: 0}, p)
	})

	t.Run("id 1, sps 1", func(t *testing.T) {
		var p h264.PPS
		err := p.Unmarshal([]byte{0x68, 0x48, 0xe3, 0xc0})
		require.NoError(t, err)
		require.Equal(t, h264.PPS{ID: 1, SPSID: 1}, p)
	})

	t.Run("id 2, sps 0", func(t *testing.T) {
		var p h264.PPS
		err := p.Unmarshal([]byte{0x68, 0x73, 0x8f, 0x00})
		require.NoError(t, err)
		require.Equal(t, h264.PPS{ID: 2, SPSID: 0}, p)
	})

	t.Run("not a PPS", func(t *testing.T) {
		var p h264.PPS
		err := p.Unmarshal([]byte{0x67, 0x42, 0x00, 0x1e})
		require.Error(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		var p h264.PPS
		err := p.Unmarshal(nil)
		require.Error(t, err)
	})
}
