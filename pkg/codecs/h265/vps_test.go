package h265_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h265"
)

func TestVPSUnmarshal(t *testing.T) {
	t.Run("id 0", func(t *testing.T) {
		var v h265.VPS
		err := v.Unmarshal([]byte{
			0x41, 0x00, 0x0c, 0x01, 0xff, 0xff, 0x01, 0x00,
			0x00, 0x00, 0x00, 0xb3, 0x80, 0x00, 0x00, 0x00,
			0x00, 0x1e, 0x7c,
		})
		require.NoError(t, err)
		require.Equal(t, uint8(0), v.ID)
	})

	t.Run("id 1", func(t *testing.T) {
		var v h265.VPS
		err := v.Unmarshal([]byte{
			0x41, 0x00, 0x1c, 0x01, 0xff, 0xff, 0x01, 0x00,
			0x00, 0x00, 0x00, 0xb3, 0x80, 0x00, 0x00, 0x00,
			0x00, 0x1e, 0x7c,
		})
		require.NoError(t, err)
		require.Equal(t, uint8(1), v.ID)
	})

	t.Run("not a VPS", func(t *testing.T) {
		var v h265.VPS
		err := v.Unmarshal([]byte{0x43, 0x00, 0x01})
		require.Error(t, err)
	})

	t.Run("too short", func(t *testing.T) {
		var v h265.VPS
		err := v.Unmarshal([]byte{0x41})
		require.Error(t, err)
	})
}
