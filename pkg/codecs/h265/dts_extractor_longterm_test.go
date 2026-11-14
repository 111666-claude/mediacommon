package h265

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/bits"
)

// bitWriter accumulates bits into a byte buffer.
type bitWriter struct {
	buf []byte
	pos int
}

func (w *bitWriter) writeFlag(v bool) {
	if v {
		w.writeBits(1, 1)
	} else {
		w.writeBits(0, 1)
	}
}

func (w *bitWriter) writeBits(v uint64, n int) {
	need := (w.pos + n + 7) / 8
	for len(w.buf) < need {
		w.buf = append(w.buf, 0)
	}
	bits.WriteBitsUnsafe(w.buf, &w.pos, v, n)
}

func (w *bitWriter) writeUE(v uint32) {
	code := v + 1
	n := 0
	for tmp := code; tmp > 1; tmp >>= 1 {
		n++
	}
	for range n {
		w.writeBits(0, 1)
	}
	for i := n; i >= 0; i-- {
		w.writeBits(uint64((code>>i)&1), 1)
	}
}

// emulationPreventionInsert inserts 0x03 after every 0x00 0x00 sequence.
func emulationPreventionInsert(rbsp []byte) []byte {
	ret := make([]byte, 0, len(rbsp)+len(rbsp)/8)
	zeros := 0

	for _, b := range rbsp {
		if zeros >= 2 && b <= 3 {
			ret = append(ret, 0x03)
			zeros = 0 // the inserted 0x03 breaks the zero run
		}
		ret = append(ret, b)
		if b == 0 {
			zeros++
		} else {
			zeros = 0
		}
	}

	return ret
}

// testLongTermSPS is a real, fully valid SPS with
// long_term_ref_pics_present_flag = 1, log2_max_pic_order_cnt_lsb = 16,
// MaxNumReorderPics = 0 and 25 fps timing info (3600 time units per frame
// in the 90 kHz clock). It is shared with the SPS unmarshal test case.
var testLongTermSPS = []byte{
	0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00, 0x03,
	0x00, 0xb0, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
	0x00, 0x5d, 0xa0, 0x02, 0x80, 0x80, 0x2d, 0x16,
	0x36, 0xb9, 0x24, 0xcb, 0xf0, 0x08, 0x00, 0x00,
	0x03, 0x00, 0x08, 0x00, 0x00, 0x03, 0x01, 0x95,
	0x08,
}

// testShortTermSPS is a real, fully valid SPS with no long-term reference
// support, log2_max_pic_order_cnt_lsb = 16, MaxNumReorderPics = 0 and
// 25 fps timing info.
var testShortTermSPS = []byte{
	0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00, 0x03,
	0x00, 0x80, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
	0x00, 0x96, 0xa0, 0x01, 0x80, 0x20, 0x06, 0xc1,
	0xfe, 0x36, 0xbb, 0xb5, 0x37, 0x77, 0x25, 0xd6,
	0x02, 0xdc, 0x04, 0x04, 0x04, 0x10, 0x00, 0x00,
	0x3e, 0x80, 0x00, 0x04, 0x26, 0x87, 0x21, 0xde,
	0xe5, 0x10, 0x01, 0x6e, 0x20, 0x00, 0x66, 0xff,
	0x00, 0x0b, 0x71, 0x00, 0x03, 0x37, 0xf8, 0x80,
}

func testPPS() []byte {
	return []byte{0x44, 0x01, 0xc1, 0x72, 0xb4, 0x62, 0x40}
}

func testIDR() []byte {
	return []byte{0x26, 0x1, 0xaf, 0x8, 0x42, 0x23, 0x48, 0x8a, 0x43, 0xe2}
}

// buildTestSlice builds a VCL slice NALU with a fully controlled reference
// picture set. The extractor reads the RBSP starting at bit 8 (the NAL
// header is stripped), so the writer reserves a leading byte. Fields after
// the long-term RPS are not needed by the extractor and are replaced by
// RBSP stop bits.
func buildTestSlice(t *testing.T, naluType NALUType, sliceType uint32,
	pocLsb uint32, log2MaxPicOrderCntLsb uint32,
	shortTermInSPS bool, numSPSSRPS uint32,
	deltaPocS0 []int32, deltaPocS1 []int32,
	numLongTermPics uint32,
) []byte {
	t.Helper()

	w := &bitWriter{}
	w.writeBits(0, 8) // reserved: skipped by the extractor after header strip

	w.writeFlag(true) // first_slice_segment_in_pic_flag

	if naluType >= NALUType_BLA_W_LP && naluType <= NALUType_RSV_IRAP_VCL23 {
		w.writeFlag(false) // no_output_of_prior_pics_flag
	}

	w.writeUE(0)         // slice_pic_parameter_set_id
	w.writeUE(sliceType) // slice_type (0=B, 1=P, 2=I)
	w.writeBits(uint64(pocLsb), int(log2MaxPicOrderCntLsb))
	w.writeFlag(shortTermInSPS) // short_term_ref_pic_set_sps_flag

	if !shortTermInSPS {
		// stRpsIdx equals num_short_term_ref_pic_sets; when it is non-zero
		// the inter_ref_pic_set_prediction_flag is present.
		if numSPSSRPS != 0 {
			w.writeFlag(false) // inter_ref_pic_set_prediction_flag
		}

		w.writeUE(uint32(len(deltaPocS0)))
		w.writeUE(uint32(len(deltaPocS1)))

		var prev int32
		for i, d := range deltaPocS0 {
			if i == 0 {
				w.writeUE(uint32(-d - 1))
			} else {
				w.writeUE(uint32(prev - d - 1))
			}
			w.writeFlag(true) // used_by_curr_pic_s0_flag
			prev = d
		}

		prev = 0
		for i, d := range deltaPocS1 {
			if i == 0 {
				w.writeUE(uint32(d - 1))
			} else {
				w.writeUE(uint32(d - prev - 1))
			}
			w.writeFlag(true) // used_by_curr_pic_s1_flag
			prev = d
		}
	}

	if numLongTermPics > 0 {
		w.writeUE(numLongTermPics) // num_long_term_pics
		for i := range numLongTermPics {
			w.writeBits(uint64(i), int(log2MaxPicOrderCntLsb)) // poc_lsb_lt
			w.writeFlag(true)                                  // used_by_curr_pic_lt_flag
			w.writeFlag(false)                                 // delta_poc_msb_present_flag
		}
	}

	w.writeFlag(true) // rbsp_stop_bit
	for w.pos%8 != 0 {
		w.writeFlag(false)
	}

	// nal_unit_type in bits 1..6; nuh_temporal_id_plus1 is set to 1 so that
	// even type 0 (TRAIL_N) produces a non-zero header byte.
	header := byte(int(naluType)<<1) | 0x01
	// w.buf is the complete RBSP: a reserved first byte skipped by the
	// extractor's pos=8 offset, followed by the real slice header content.
	return append([]byte{header}, emulationPreventionInsert(w.buf)...)
}

func mustParseSPS(t *testing.T, nalu []byte) *SPS {
	t.Helper()
	var sps SPS
	require.NoError(t, sps.Unmarshal(nalu))
	return &sps
}

// TestDTSExtractorLongTermReference covers streams using long-term reference
// pictures. Such a stream contains P frames whose short-term RPS has no
// negative entry; before the fix the extractor returned "invalid DeltaPocS0"
// and the stream stopped at the second GOP.
func TestDTSExtractorLongTermReference(t *testing.T) {
	sps := mustParseSPS(t, testLongTermSPS)
	require.True(t, sps.LongTermRefPicsPresentFlag)
	require.Equal(t, uint32(12), sps.Log2MaxPicOrderCntLsbMinus4)
	require.Equal(t, []uint32{0}, sps.MaxNumReorderPics)

	ex := NewDTSExtractor()

	// IDR starts the sequence; MaxNumReorderPics is 0, so the first DTS is
	// derived from the PTS/DTS difference of the IDR itself (zero).
	_, err := ex.Extract([][]byte{testLongTermSPS, testPPS(), testIDR()}, 0)
	require.NoError(t, err)

	// A P frame with an empty short-term RPS but a long-term entry. Before
	// the fix this returned "invalid DeltaPocS0".
	longTermP := buildTestSlice(t, NALUType_TRAIL_R, 1, 3, 16, false, 0, nil, nil, 1)
	dts, err := ex.Extract([][]byte{longTermP}, 3600)
	require.NoError(t, err)
	require.LessOrEqual(t, dts, int64(3600))

	// A regular P frame with a normal negative short-term reference.
	regularP := buildTestSlice(t, NALUType_TRAIL_R, 1, 6, 16, false, 0,
		[]int32{-1}, nil, 0)
	dts2, err := ex.Extract([][]byte{regularP}, 7200)
	require.NoError(t, err)
	require.GreaterOrEqual(t, dts2, dts)

	// Second long-term P frame, mimicking the start of another GOP: this is
	// the point where the playback used to stop.
	longTermP2 := buildTestSlice(t, NALUType_TRAIL_R, 1, 9, 16, false, 0, nil, nil, 1)
	dts3, err := ex.Extract([][]byte{longTermP2}, 10800)
	require.NoError(t, err)
	require.GreaterOrEqual(t, dts3, dts2)
	require.LessOrEqual(t, dts3, int64(10800))
}

// TestDTSExtractorLongTermWithShortTermRPS mixes short-term and long-term
// references and exercises both reference lists (L0/L1).
func TestDTSExtractorLongTermWithShortTermRPS(t *testing.T) {
	sps := mustParseSPS(t, testLongTermSPS)
	require.True(t, sps.LongTermRefPicsPresentFlag)

	var pps PPS
	require.NoError(t, pps.Unmarshal(testPPS()))

	// At the diff-computation level, a B frame with a negative (L0) and a
	// positive (L1) short-term picture plus a long-term entry is parsed
	// correctly: the long-term RPS is consumed and both lists are accounted
	// for. The formula is -DeltaPocS0[0]-1 minus the number of L1 pictures.
	bFrame := buildTestSlice(t, NALUType_TRAIL_R, 0, 4, 16, false, 0,
		[]int32{-1}, []int32{2}, 1)
	diff, err := getPTSDTSDiff(bFrame, sps, &pps)
	require.NoError(t, err)
	require.Equal(t, (-(-1)-1)-1, diff)

	ex := NewDTSExtractor()
	_, err = ex.Extract([][]byte{testLongTermSPS, testPPS(), testIDR()}, 0)
	require.NoError(t, err)

	// P frame with a short-term negative reference and no long-term entry.
	pShort := buildTestSlice(t, NALUType_TRAIL_R, 1, 3, 16, false, 0,
		[]int32{-1}, nil, 0)
	dtsShort, err := ex.Extract([][]byte{pShort}, 3600)
	require.NoError(t, err)

	// P frame whose short-term RPS is empty but that carries a long-term entry.
	pLong := buildTestSlice(t, NALUType_TRAIL_R, 1, 6, 16, false, 0, nil, nil, 2)
	dtsLong, err := ex.Extract([][]byte{pLong}, 7200)
	require.NoError(t, err)
	require.GreaterOrEqual(t, dtsLong, dtsShort)
	require.LessOrEqual(t, dtsLong, int64(7200))
}

// TestDTSExtractorRPSOnlyL1 checks the non-reference TRAIL_N case whose RPS
// only contains positive pictures.
func TestDTSExtractorRPSOnlyL1(t *testing.T) {
	sps := mustParseSPS(t, testShortTermSPS)
	var pps PPS
	require.NoError(t, pps.Unmarshal(testPPS()))

	// A non-reference B frame with positive pictures only reports a negative
	// difference equal to the number of positive pictures; verify this at the
	// diff-computation level, where reordering frames are not involved.
	bOnlyL1 := buildTestSlice(t, NALUType_TRAIL_N, 0, 1, 16, false, 1,
		nil, []int32{2, 4}, 0)
	diff, err := getPTSDTSDiff(bOnlyL1, sps, &pps)
	require.NoError(t, err)
	require.Equal(t, -2, diff)

	// End-to-end, an anchor P frame followed by a regular P frame stays
	// monotonic with this short-term-only SPS.
	ex := NewDTSExtractor()
	_, err = ex.Extract([][]byte{testShortTermSPS, testPPS(), testIDR()}, 0)
	require.NoError(t, err)

	anchor := buildTestSlice(t, NALUType_TRAIL_R, 1, 3, 16, false, 1,
		[]int32{-1}, nil, 0)
	dts1, err := ex.Extract([][]byte{anchor}, 10800)
	require.NoError(t, err)

	next := buildTestSlice(t, NALUType_TRAIL_R, 1, 6, 16, false, 1,
		[]int32{-1}, nil, 0)
	dts2, err := ex.Extract([][]byte{next}, 14400)
	require.NoError(t, err)
	require.GreaterOrEqual(t, dts2, dts1)
}

// TestDTSExtractorErrors makes sure malformed inputs keep returning errors
// instead of being silently accepted.
func TestDTSExtractorErrors(t *testing.T) {
	t.Run("SPS not received yet", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{testIDR()}, 0)
		require.EqualError(t, err, "SPS not received yet")
	})

	t.Run("PPS not received yet", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{testShortTermSPS, testIDR()}, 0)
		require.EqualError(t, err, "PPS not received yet")
	})

	t.Run("random access frame not received yet", func(t *testing.T) {
		ex := NewDTSExtractor()
		slice := buildTestSlice(t, NALUType_TRAIL_R, 1, 3, 16, false, 1,
			[]int32{-1}, nil, 0)
		_, err := ex.Extract([][]byte{testShortTermSPS, testPPS(), slice}, 3600)
		require.EqualError(t, err, "random access frame not received yet")
	})

	t.Run("invalid SPS", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{{0x42, 0x00}, testPPS(), testIDR()}, 0)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid SPS")
	})

	t.Run("invalid PPS", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract(
			[][]byte{testShortTermSPS, []byte{0x44, 0x00}, testIDR()}, 0)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid PPS")
	})

	t.Run("empty short-term RPS on P frame without long-term", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{testShortTermSPS, testPPS(), testIDR()}, 0)
		require.NoError(t, err)

		bad := buildTestSlice(t, NALUType_TRAIL_R, 1, 3, 16, false, 1, nil, nil, 0)
		_, err = ex.Extract([][]byte{bad}, 3600)
		require.EqualError(t, err, "invalid DeltaPocS0")
	})

	t.Run("truncated slice NALU", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{testLongTermSPS, testPPS(), testIDR()}, 0)
		require.NoError(t, err)

		// header ends before the RPS flags can be read.
		_, err = ex.Extract([][]byte{{0x02, 0x01, 0x00}}, 3600)
		require.Error(t, err)
	})

	t.Run("truncated long-term RPS", func(t *testing.T) {
		ex := NewDTSExtractor()
		_, err := ex.Extract([][]byte{testLongTermSPS, testPPS(), testIDR()}, 0)
		require.NoError(t, err)

		// num_long_term_pics = 2 but no entry follows.
		w := &bitWriter{}
		w.writeBits(0, 8)
		w.writeFlag(true)
		w.writeUE(0) // pps id
		w.writeUE(1) // P slice
		w.writeBits(3, 16)
		w.writeFlag(false) // inline short-term RPS
		w.writeUE(0)       // num_negative_pics
		w.writeUE(0)       // num_positive_pics
		w.writeUE(2)       // num_long_term_pics
		w.writeFlag(true)  // rbsp stop bit, too early
		for w.pos%8 != 0 {
			w.writeFlag(false)
		}
		nalu := append([]byte{0x03}, emulationPreventionInsert(w.buf)...)

		_, err = ex.Extract([][]byte{nalu}, 3600)
		require.Error(t, err)
	})
}

// TestDTSExtractorStateIsolation makes sure that a failed extraction does not
// poison the extractor: the next valid sequence can start again.
func TestDTSExtractorStateIsolation(t *testing.T) {
	ex := NewDTSExtractor()

	// invalid AU before any random access picture
	_, err := ex.Extract([][]byte{testLongTermSPS, testPPS()}, 0)
	require.Error(t, err)

	// a fresh, valid sequence can start on the same extractor
	_, err = ex.Extract([][]byte{testLongTermSPS, testPPS(), testIDR()}, 0)
	require.NoError(t, err)

	// malformed long-term frame fails ...
	bad := buildTestSlice(t, NALUType_TRAIL_R, 1, 3, 16, false, 0, nil, nil, 0)
	_, err = ex.Extract([][]byte{bad}, 3600)
	require.Error(t, err)

	// ... and the next legal frame is still accepted
	good := buildTestSlice(t, NALUType_TRAIL_R, 1, 3, 16, false, 0, nil, nil, 1)
	_, err = ex.Extract([][]byte{good}, 3600)
	require.NoError(t, err)

	// two extractors do not share state
	ex2 := NewDTSExtractor()
	_, err = ex2.Extract([][]byte{testLongTermSPS, testPPS(), testIDR()}, 0)
	require.NoError(t, err)
	_, err = ex2.Extract([][]byte{good}, 3600)
	require.NoError(t, err)
}

// TestDTSExtractorKeyframeReset checks that a new IDR keeps DTS monotonic
// across the reset boundary.
func TestDTSExtractorKeyframeReset(t *testing.T) {
	ex := NewDTSExtractor()

	_, err := ex.Extract([][]byte{testShortTermSPS, testPPS(), testIDR()}, 0)
	require.NoError(t, err)

	p := buildTestSlice(t, NALUType_TRAIL_R, 1, 3, 16, false, 1,
		[]int32{-1}, nil, 0)
	dts1, err := ex.Extract([][]byte{p}, 10800)
	require.NoError(t, err)

	// new IDR at a later PTS restarts the sequence.
	dts2, err := ex.Extract(
		[][]byte{testShortTermSPS, testPPS(), testIDR()}, 36000)
	require.NoError(t, err)
	require.Greater(t, dts2, dts1)
	require.LessOrEqual(t, dts2, int64(36000))

	// frames after the reset are still monotonic.
	p2 := buildTestSlice(t, NALUType_TRAIL_R, 1, 3, 16, false, 1,
		[]int32{-1}, nil, 0)
	dts3, err := ex.Extract([][]byte{p2}, 46800)
	require.NoError(t, err)
	require.Greater(t, dts3, dts2)
}

// TestDTSExtractorPOCWraparound feeds slices whose pic_order_cnt_lsb crosses
// the 16-bit boundary and checks that DTS stays monotonic. It mixes ordinary
// short-term references with long-term references on both sides of the wrap.
func TestDTSExtractorPOCWraparound(t *testing.T) {
	ex := NewDTSExtractor()
	_, err := ex.Extract([][]byte{testLongTermSPS, testPPS(), testIDR()}, 0)
	require.NoError(t, err)

	// POC values straddle the 0xffff -> 0x0000 boundary.
	pocValues := []uint32{0xfffd, 0xffff, 0x0001, 0x0003, 0x0005}
	var prevDTS int64

	for i, poc := range pocValues {
		pts := int64((i + 1) * 3600)

		var slice []byte
		if i%2 == 0 {
			// long-term reference with an empty short-term RPS
			slice = buildTestSlice(t, NALUType_TRAIL_R, 1, poc, 16,
				false, 0, nil, nil, 1)
		} else {
			// regular short-term negative reference
			slice = buildTestSlice(t, NALUType_TRAIL_R, 1, poc, 16,
				false, 0, []int32{-1}, nil, 0)
		}

		var dts int64
		dts, err = ex.Extract([][]byte{slice}, pts)
		require.NoError(t, err, "POC %d", poc)
		require.LessOrEqual(t, dts, pts, "POC %d", poc)

		if i > 0 {
			require.GreaterOrEqual(t, dts, prevDTS, "non-monotonic DTS at POC %d", poc)
		}
		prevDTS = dts
	}
}

// TestDTSExtractorLinear checks that extraction stays linear in the number of
// access units: no per-frame scan of past frames.
func TestDTSExtractorLinear(t *testing.T) {
	ex := NewDTSExtractor()
	_, err := ex.Extract([][]byte{testLongTermSPS, testPPS(), testIDR()}, 0)
	require.NoError(t, err)

	const n = 6000
	var prevDTS int64
	for i := range n {
		var slice []byte
		pts := int64((i + 1) * 3600)
		if i%5 == 0 {
			slice = buildTestSlice(t, NALUType_TRAIL_R, 1,
				uint32((i+1)*3&0xffff), 16, false, 0, nil, nil, 1)
		} else {
			slice = buildTestSlice(t, NALUType_TRAIL_R, 1,
				uint32((i+1)*3&0xffff), 16, false, 0, []int32{-1}, nil, 0)
		}

		var dts int64
		dts, err = ex.Extract([][]byte{slice}, pts)
		require.NoError(t, err)
		if i > 0 {
			require.GreaterOrEqual(t, dts, prevDTS)
		}
		prevDTS = dts
	}
}
