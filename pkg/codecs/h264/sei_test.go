package h264

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSEIUnmarshal(t *testing.T) {
	for _, ca := range []struct {
		name string
		nalu []byte
		want bool
	}{
		{
			"recovery point",
			// type=6, size=1, recovery_frame_cnt=0, rbsp_trailing_bits
			[]byte{0x06, 0x06, 0x01, 0x80},
			true,
		},
		{
			"recovery point with non-zero recovery_frame_cnt and flags",
			// type=6, size=2, recovery_frame_cnt=1, exact=1, broken=0, idc=2
			[]byte{0x06, 0x06, 0x02, 0x2c, 0x80},
			true,
		},
		{
			"buffering period and pic timing only",
			[]byte{0x06, 0x01, 0x02, 0x04, 0x24, 0x80},
			false,
		},
		{
			"pic timing followed by recovery point",
			// type=1 size=2 (04 24), type=6 size=1
			// (recovery_frame_cnt=0, golomb byte 0x80),
			// ending byte aligned without explicit trailing bits
			[]byte{0x06, 0x01, 0x02, 0x04, 0x24, 0x06, 0x01, 0x80},
			true,
		},
		{
			"pic timing followed by recovery point with trailing bits",
			// type=1 size=2 (04 24), type=6 size=1 (0x80),
			// one byte of alignment_zero_bits, then rbsp_trailing_bits
			[]byte{0x06, 0x01, 0x02, 0x04, 0x24, 0x06, 0x01, 0x80, 0x00, 0x80},
			true,
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var sei SEI
			err := sei.Unmarshal(ca.nalu)
			require.NoError(t, err)
			require.Equal(t, ca.want, sei.RecoveryPoint)
		})
	}
}

func TestSEIUnmarshalExtended(t *testing.T) {
	// payload type = 6 (recovery point),
	// payload size = 0xff + 0x02 = 257, exercising the
	// multi-byte payloadSize escape encoding
	nalu := []byte{0x06, 0x06, 0xff, 0x02}
	// recovery_frame_cnt=0 (golomb 1), exact=1, broken=0, idc=0
	nalu = append(nalu, 0xc0)
	// 256 more payload bytes, then rbsp_trailing_bits
	for i := 0; i < 256; i++ {
		nalu = append(nalu, 0x00)
	}
	nalu = append(nalu, 0x80)

	var sei SEI
	err := sei.Unmarshal(nalu)
	require.NoError(t, err)
	require.True(t, sei.RecoveryPoint)
}

func TestSEIUnmarshalErrors(t *testing.T) {
	for _, ca := range []struct {
		name string
		nalu []byte
	}{
		{
			"empty",
			[]byte{},
		},
		{
			"not an SEI",
			[]byte{0x41, 0x9a, 0x21},
		},
		{
			"declared payload bigger than NALU",
			// type=6, size=5, only one payload byte
			[]byte{0x06, 0x06, 0x05, 0x00, 0x80},
		},
		{
			"truncated recovery point payload",
			// type=6 size=4 but only two payload bytes, then trailing
			[]byte{0x06, 0x06, 0x04, 0x2c, 0x80, 0x80},
		},
		{
			"truncated recovery point header",
			// type=6 size=1, payload starts but is empty
			[]byte{0x06, 0x06, 0x01},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var sei SEI
			err := sei.Unmarshal(ca.nalu)
			require.Error(t, err)
			require.False(t, sei.RecoveryPoint)
		})
	}
}

func TestSEIUnmarshalTruncatedNonRecovery(t *testing.T) {
	for _, ca := range []struct {
		name string
		nalu []byte
	}{
		{
			// real-world SEI from the mbs_only_flag=0 DTS fixture:
			// a complete buffering period payload followed by a
			// single padding byte that does not form a full payload
			"buffering period with padding byte",
			[]byte{0x6, 0x0, 0x6, 0x85, 0x7e, 0x40, 0x0, 0x0, 0x10, 0x1},
		},
		{
			// pic timing payload whose declared size exceeds the NALU
			"truncated pic timing payload",
			[]byte{0x06, 0x01, 0x05, 0x04, 0x24, 0x80},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var sei SEI
			err := sei.Unmarshal(ca.nalu)
			require.NoError(t, err)
			require.False(t, sei.RecoveryPoint)
		})
	}
}

func TestSEIUnmarshalEmulationPrevention(t *testing.T) {
	// recovery point payload containing a 0x00 0x00 0x03 sequence:
	// type=6, size=4, payload = 2c 00 00 00, trailing
	rbsp := []byte{0x06, 0x06, 0x04, 0x2c, 0x00, 0x00, 0x00, 0x80}
	// insert an emulation prevention byte after 00 00
	nalu := []byte{0x06, 0x06, 0x04, 0x2c, 0x00, 0x00, 0x03, 0x00, 0x80}
	require.Equal(t, rbsp, EmulationPreventionRemove(nalu))

	var sei SEI
	err := sei.Unmarshal(nalu)
	require.NoError(t, err)
	require.True(t, sei.RecoveryPoint)
}
