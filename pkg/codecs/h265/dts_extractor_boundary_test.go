package h265

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fixtures derived from a valid long-term reference stream:
// the SPS has long_term_ref_pics_present_flag = 1, no reordering
// (MaxNumReorderPics = 0) and a 30 Hz timing info.
var ltrVPS = []byte{
	0x40, 0x01, 0x0c, 0x03, 0xff, 0xff, 0x01, 0x60,
	0x00, 0x00, 0x03, 0x00, 0x80, 0x00, 0x00, 0x03,
	0x00, 0x00, 0x03, 0x00, 0x78, 0x00, 0x00, 0xbd,
	0xe0, 0x48,
}

var ltrSPS = []byte{
	0x42, 0x01, 0x03, 0x01, 0x60, 0x00, 0x00, 0x03,
	0x00, 0x80, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
	0x00, 0x78, 0x00, 0x00, 0xa0, 0x03, 0xc0, 0x80,
	0x10, 0xe7, 0xf8, 0xdb, 0xdf, 0x76, 0xa6, 0xee,
	0xe4, 0xbf, 0x60, 0x2d, 0xc0, 0x40, 0x40, 0x41,
	0x00, 0x00, 0x03, 0x03, 0xe8, 0x00, 0x00, 0x75,
	0x30, 0x72, 0x1d, 0xee, 0x51, 0x00, 0x72, 0x20,
	0x00, 0x80, 0x64, 0x00, 0xe4, 0x40, 0x01, 0x00,
	0xc8, 0x40, 0x1c, 0x88, 0x00, 0x20, 0x19, 0x00,
	0x39, 0x10, 0x00, 0x40, 0x32, 0x20,
}

var ltrPPS = []byte{
	0x44, 0x01, 0xc1, 0x72, 0xb0, 0x9c, 0x14, 0x0a,
	0x66, 0x40,
}

var ltrIDR = []byte{
	0x26, 0x01, 0xaf, 0x09, 0x70, 0xf3, 0xbb, 0xc2,
	0x9c, 0x58, 0xd5, 0x7f, 0x3c, 0x57, 0x68, 0x33,
	0x83, 0xf7, 0x50, 0x87, 0x4c, 0x19, 0x40, 0x9f,
	0x2c, 0xd7, 0x73, 0x70, 0x6d, 0x15, 0x3a, 0xa0,
}

// first slice of a P-frame whose short-term RPS is empty and which only
// references long-term reference pictures.
var ltrTRAIL = []byte{
	0x02, 0x03, 0xd0, 0x00, 0x09, 0xa0, 0x00, 0x0b,
	0xa1, 0xce, 0xfc, 0x18, 0xfc, 0x02, 0x30, 0x08,
	0x5e, 0x80, 0x36, 0xd3, 0x80, 0x00, 0x19, 0xbc,
	0xec, 0x0f, 0xad, 0x40, 0x00, 0x00, 0x03, 0x00,
}

// SPS/PPS of a stream with B-frames (MaxNumReorderPics = 2), used to build
// compact synthetic slice headers that exercise the RPS branch logic.
var bframeSPS = []byte{
	0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00, 0x03,
	0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
	0x00, 0x78, 0xa0, 0x03, 0xc0, 0x80, 0x11, 0x07,
	0xcb, 0x96, 0x56, 0x64, 0xa4, 0xc2, 0xf0, 0x16,
	0x80, 0x80, 0x00, 0x00, 0x03, 0x00, 0x80, 0x00,
	0x00, 0x0f, 0x04,
}

var bframePPS = []byte{0x44, 0x01, 0xc0, 0x73, 0xc1, 0x89}

// packBits builds a byte slice from an MSB-first bit string, zero-padding
// the final byte.
func packBits(bits string) []byte {
	for len(bits)%8 != 0 {
		bits += "0"
	}

	out := make([]byte, len(bits)/8)
	for i := range out {
		var b byte
		for j := 0; j < 8; j++ {
			b = (b << 1) | (bits[i*8+j] - '0')
		}
		out[i] = b
	}
	return out
}

// buildSliceNALU builds a TRAIL_R slice NALU with an inline short-term RPS.
// sliceType is the H.265 slice_type syntax (0 = B, 1 = P), numNeg/numPos are
// the RPS list sizes and every reference entry uses delta = 1 and
// used_by_curr_pic = 1.
func buildSliceNALU(sliceType uint32, numNeg, numPos uint32) []byte {
	ue := func(v uint32) string {
		code := v + 1
		n := 0
		for x := code; x > 0; x >>= 1 {
			n++
		}
		s := strings.Repeat("0", n-1)
		s += strconv.FormatUint(uint64(code), 2)
		return s
	}

	negBody := strings.Repeat(ue(0)+"1", int(numNeg)) // delta minus1 = 0, used = 1
	posBody := strings.Repeat(ue(0)+"1", int(numPos))

	// 8 reserved bits (the extractor starts at bit 8), then:
	// first_slice_segment_in_pic_flag, pps_id, slice_type, poc_lsb(8),
	// short_term_ref_pic_set_sps_flag = 0, RPS.
	bits := "00000000" + "1" + ue(0) + ue(sliceType) + "00000000" + "0" +
		ue(numNeg) + negBody + ue(numPos) + posBody

	nalu := append([]byte{0x02}, packBits(bits)...) // 0x02 = TRAIL_R
	for len(nalu) <= maxBytesToGetPOC {
		nalu = append(nalu, 0x00)
	}
	return nalu
}

func TestDTSExtractorErrorsAndBoundaries(t *testing.T) {
	t.Run("SPS not received yet", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{ltrIDR}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "SPS not received yet")
	})

	t.Run("PPS not received yet", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{ltrVPS, ltrSPS, ltrIDR}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "PPS not received yet")
	})

	t.Run("random access frame not received yet", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{ltrVPS, ltrSPS, ltrPPS, ltrTRAIL}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "random access frame not received yet")
	})

	t.Run("access unit without VCL NALU", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{ltrVPS, ltrSPS, ltrPPS}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "random access frame not received yet")

		// once a random access picture unlocked the stream, an AU with no
		// VCL NALU must still be rejected.
		_, err = ex.Extract([][]byte{ltrVPS, ltrSPS, ltrPPS, ltrIDR}, 0)
		require.NoError(t, err)
		_, err = ex.Extract([][]byte{ltrVPS, ltrSPS, ltrPPS}, 3000)
		require.Error(t, err)
		require.Contains(t, err.Error(), "doesn't contain an IDR or non-IDR NALU")
	})

	t.Run("corrupted SPS is rejected", func(t *testing.T) {
		ex := NewDTSExtractor()
		// keep only the NAL unit header: the SPS body cannot be parsed.
		badSPS := append([]byte{}, ltrSPS[:2]...)
		_, err := ex.Extract([][]byte{ltrVPS, badSPS, ltrPPS, ltrIDR}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid SPS")
	})

	t.Run("corrupted PPS is rejected", func(t *testing.T) {
		ex := NewDTSExtractor()
		// keep only the NAL unit header: the PPS body cannot be parsed.
		badPPS := append([]byte{}, ltrPPS[:2]...)
		_, err := ex.Extract([][]byte{ltrVPS, ltrSPS, badPPS, ltrIDR}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid PPS")
	})

	t.Run("truncated slice NALU is rejected", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{ltrVPS, ltrSPS, ltrPPS, ltrIDR}, 0)
		require.NoError(t, err)

		truncated := append([]byte{}, ltrTRAIL[:3]...)
		_, err = ex.Extract([][]byte{truncated}, 3000)
		require.Error(t, err)
	})

	t.Run("failed AU does not corrupt state, next valid sequence restarts", func(t *testing.T) {
		ex := NewDTSExtractor()

		// a failing AU (no random access picture yet)
		_, err := ex.Extract([][]byte{ltrTRAIL}, 0)
		require.Error(t, err)

		// a complete, valid sequence can start from scratch
		dts, err := ex.Extract([][]byte{ltrVPS, ltrSPS, ltrPPS, ltrIDR}, 0)
		require.NoError(t, err)
		require.Equal(t, int64(0), dts)

		// long-term P-frame: empty short-term RPS must not be an error
		dts, err = ex.Extract([][]byte{ltrTRAIL}, 3000)
		require.NoError(t, err)
		require.Equal(t, int64(3000), dts)
	})

	t.Run("new SPS resets stream state", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{ltrVPS, ltrSPS, ltrPPS, ltrIDR}, 0)
		require.NoError(t, err)
		_, err = ex.Extract([][]byte{ltrTRAIL}, 3000)
		require.NoError(t, err)

		// re-sending the parameter sets and an IDR restarts a fresh sequence
		dts, err := ex.Extract([][]byte{ltrVPS, ltrSPS, ltrPPS, ltrIDR}, 6000)
		require.NoError(t, err)
		require.Equal(t, int64(6000), dts)
	})
}

func TestDTSExtractorLongTermMonotonic(t *testing.T) {
	ex := NewDTSExtractor()

	// key frame followed by a run of pictures that only use long-term
	// references: DTS must be monotonic, equal to PTS (no reordering) and
	// spaced by exactly one frame duration (3000 at 30 Hz).
	pts := []int64{0, 3000, 6000, 9000, 12000, 15000}
	want := []int64{0, 3000, 6000, 9000, 12000, 15000}

	var prev int64
	for i, p := range pts {
		var au [][]byte
		if i == 0 {
			au = [][]byte{ltrVPS, ltrSPS, ltrPPS, ltrIDR}
		} else {
			au = [][]byte{ltrTRAIL}
		}

		dts, err := ex.Extract(au, p)
		require.NoError(t, err)
		require.Equal(t, want[i], dts, "AU %d", i)
		require.LessOrEqual(t, dts, p, "DTS must not exceed PTS at AU %d", i)

		if i > 0 {
			require.Greater(t, dts, prev, "DTS must be strictly increasing at AU %d", i)
			require.Equal(t, int64(3000), dts-prev, "DTS delta must stay constant at AU %d", i)
		}
		prev = dts
	}
}

func TestDTSExtractorRPSBranches(t *testing.T) {
	var spsp SPS
	require.NoError(t, spsp.Unmarshal(bframeSPS))

	var ppsp PPS
	require.NoError(t, ppsp.Unmarshal(bframePPS))

	cases := []struct {
		name     string
		nalu     []byte
		wantDiff int
	}{
		{
			// B-frame with an empty S0 list and a single L1 entry: valid for
			// long-term-only references, distance comes from S1 only.
			"B-frame, empty S0, L1 only",
			buildSliceNALU(0, 0, 1),
			-1,
		},
		{
			// P-frame with an empty short-term RPS (long-term references only).
			"P-frame, empty RPS",
			buildSliceNALU(1, 0, 0),
			0,
		},
		{
			// legacy behavior must be unchanged when S0 is present.
			"P-frame, S0 only",
			buildSliceNALU(1, 1, 0),
			0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diff, err := getPTSDTSDiff(c.nalu, &spsp, &ppsp)
			require.NoError(t, err)
			require.Equal(t, c.wantDiff, diff)
		})
	}
}
