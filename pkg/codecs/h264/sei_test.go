package h264

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasRecoveryPointSEI(t *testing.T) {
	t.Run("recovery point", func(t *testing.T) {
		// payloadType=6, payloadSize=2, recovery_frame_cnt=0,
		// exact_match_flag=1, broken_link_flag=0,
		// changing_slice_group_idc=0, RBSP stop + alignment
		ok, err := hasRecoveryPointSEI([]byte{0x06, 0x06, 0x02, 0x00, 0x83, 0x80})
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("recovery point with non-zero frame count", func(t *testing.T) {
		// payloadType=6, payloadSize=2, recovery_frame_cnt=1,
		// exact_match_flag=0, broken_link_flag=1,
		// changing_slice_group_idc=0, RBSP stop + alignment
		ok, err := hasRecoveryPointSEI([]byte{0x06, 0x06, 0x02, 0x01, 0x42, 0x80})
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("recovery point with emulation prevention bytes", func(t *testing.T) {
		ok, err := hasRecoveryPointSEI([]byte{
			0x06, 0x06, 0x02, 0x00, 0x00, 0x03, 0x83, 0x00, 0x03, 0x80,
		})
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("recovery point after another message", func(t *testing.T) {
		// user data (payloadType=5, 4 bytes, the last one being the RBSP
		// stop byte of that payload), then recovery point
		ok, err := hasRecoveryPointSEI([]byte{
			0x06,
			0x05, 0x04, 0xAA, 0xBB, 0xCC, 0x80,
			0x06, 0x02, 0x00, 0x83, 0x80,
		})
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("ordinary SEI without recovery point", func(t *testing.T) {
		// user data (payloadType=5), followed by the RBSP stop byte
		ok, err := hasRecoveryPointSEI([]byte{0x06, 0x05, 0x04, 0xAA, 0xBB, 0xCC, 0x80})
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("empty SEI", func(t *testing.T) {
		// no payload at all: not a recovery point, not an error either,
		// since an SEI NAL unit may contain no message
		ok, err := hasRecoveryPointSEI([]byte{0x06})
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("truncated payload type", func(t *testing.T) {
		// 0xFF introduces a multi-byte payload type, but nothing follows
		_, err := hasRecoveryPointSEI([]byte{0x06, 0xFF})
		require.Error(t, err)
	})

	t.Run("truncated payload size", func(t *testing.T) {
		// payloadType=5, then an unterminated multi-byte payload size
		_, err := hasRecoveryPointSEI([]byte{0x06, 0x05, 0xFF})
		require.Error(t, err)
	})

	t.Run("payload exceeds NAL unit", func(t *testing.T) {
		// payloadType=6, payloadSize=10 but only one payload byte follows
		_, err := hasRecoveryPointSEI([]byte{0x06, 0x06, 0x0A, 0x00})
		require.Error(t, err)
	})

	t.Run("truncated recovery point payload", func(t *testing.T) {
		// payloadType=6, payloadSize=2 but only one payload byte follows,
		// so the recovery point fields cannot be parsed
		_, err := hasRecoveryPointSEI([]byte{0x06, 0x06, 0x02, 0x00})
		require.Error(t, err)
	})
}
