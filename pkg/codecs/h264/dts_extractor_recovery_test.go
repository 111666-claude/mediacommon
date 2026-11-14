package h264

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// SPS and slices reused by the recovery-point tests. The slices are
// built so that their pic_order_cnt_lsb is known (0, 4, 6, 12, ...)
// and they exercise both P and B (reordered) pictures.
var (
	recoveryTestSPS = []byte{
		0x67, 0x64, 0x00, 0x28, 0xac, 0xd9, 0x40, 0x78,
		0x02, 0x27, 0xe5, 0x84, 0x00, 0x00, 0x03, 0x00,
		0x04, 0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60,
		0xc6, 0x58,
	}
	// nal_unit_type=6 recovery point SEI: payload type 6, size 1,
	// recovery_frame_cnt=0, followed by rbsp_trailing_bits.
	recoveryTestSEI = []byte{0x06, 0x06, 0x01, 0x80}
	// a recovery point SEI whose declared payload does not fit in
	// the NALU, used to check error handling.
	recoveryTestBadSEI = []byte{0x06, 0x06, 0x05, 0x00, 0x80}
	// a non-recovery (pic timing) SEI
	recoveryTestOtherSEI = []byte{0x06, 0x01, 0x02, 0x04, 0x24, 0x80}

	recoveryTestPOC0  = []byte{0x41, 0x9a, 0x00, 0x6c, 0x45, 0xff} // non-IDR, POC 0
	recoveryTestPoc4  = []byte{0x41, 0x9a, 0x42, 0x3c, 0x21, 0x93} // non-IDR, POC 4
	recoveryTestPoc6  = []byte{0x41, 0x9a, 0x63, 0x49, 0xe1, 0x0f} // non-IDR, POC 6
	recoveryTestPoc12 = []byte{0x41, 0x9a, 0x86, 0x49, 0xe1, 0x0f} // non-IDR, POC 12
	recoveryTestPoc10 = []byte{0x41, 0x9e, 0xa5, 0x42, 0x7f, 0xf9} // non-IDR, POC 10
	recoveryTestPoc8  = []byte{0x01, 0x9e, 0xc4, 0x69, 0x13, 0xff} // non-IDR, POC 8
	recoveryTestPoc16 = []byte{0x41, 0x9a, 0xc8, 0x4b, 0xa8, 0x42} // non-IDR, POC 16
	recoveryTestIDR   = []byte{0x65, 0x88, 0x84, 0x00, 0x33, 0xff} // IDR, POC 0
)

func TestDTSExtractorRecoveryPoint(t *testing.T) {
	// a stream starting with SPS + recovery point SEI + a non-IDR
	// picture, followed by reordered non-IDRs and then an IDR.
	sequence := []sample{
		{[][]byte{recoveryTestSPS, recoveryTestSEI, recoveryTestPOC0}, 56890, 56890},
		{[][]byte{recoveryTestPoc4}, 58390, 59890},
		{[][]byte{recoveryTestPoc6}, 60640, 62890},
		{[][]byte{recoveryTestPoc12}, 60730, 74890},
		{[][]byte{recoveryTestPoc10}, 60820, 71890},
		{[][]byte{recoveryTestPoc8}, 64855, 68890},
		{[][]byte{recoveryTestPoc16}, 68062, 80890},
		{[][]byte{recoveryTestIDR}, 72338, 80890},
	}

	ex := NewDTSExtractor()
	for _, sa := range sequence {
		dts, err := ex.Extract(sa.au, sa.pts)
		require.NoError(t, err)
		require.Equal(t, sa.dts, dts)
	}

	// every computed DTS must be monotonically increasing and
	// never greater than its PTS
	exMonotonic := NewDTSExtractor()
	var prev int64
	for i, sa := range sequence {
		dts, err := exMonotonic.Extract(sa.au, sa.pts)
		require.NoError(t, err)
		require.LessOrEqual(t, dts, sa.pts)
		if i > 0 {
			require.Greater(t, dts, prev)
		}
		prev = dts
	}
}

func TestDTSExtractorRecoveryPointNALUOrdering(t *testing.T) {
	// the ordering of SPS, SEI and slice NALUs within the access
	// unit must not change the result.
	variants := [][][]byte{
		{recoveryTestSPS, recoveryTestSEI, recoveryTestPOC0},
		{recoveryTestPOC0, recoveryTestSEI, recoveryTestSPS},
		{recoveryTestSEI, recoveryTestSPS, recoveryTestPOC0},
		{recoveryTestSEI, recoveryTestPOC0, recoveryTestSPS},
		{recoveryTestSPS, recoveryTestPOC0, recoveryTestSEI},
	}

	for _, au := range variants {
		ex := NewDTSExtractor()
		dts, err := ex.Extract(au, 12345)
		require.NoError(t, err)
		require.Equal(t, int64(12345), dts)
	}
}

func TestDTSExtractorRecoveryPointAfterImageIgnored(t *testing.T) {
	ex := NewDTSExtractor()

	// an IDR establishes the random access point
	dts, err := ex.Extract([][]byte{recoveryTestSPS, recoveryTestIDR}, 1000)
	require.NoError(t, err)
	require.Equal(t, int64(1000), dts)

	// a later recovery point must not reset the reference state;
	// the non-IDR is processed as an ordinary continuation frame
	dts, err = ex.Extract([][]byte{recoveryTestSEI, recoveryTestPoc4}, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(1500), dts)
}

func TestDTSExtractorRecoveryPointThenIDR(t *testing.T) {
	// an IDR in the same access unit as the recovery point takes
	// precedence and the recovery flag is consumed.
	ex := NewDTSExtractor()
	dts, err := ex.Extract([][]byte{recoveryTestSPS, recoveryTestSEI, recoveryTestIDR}, 1000)
	require.NoError(t, err)
	require.Equal(t, int64(1000), dts)

	// the following non-IDR is a regular frame, not a new baseline
	dts, err = ex.Extract([][]byte{recoveryTestPoc4}, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(1500), dts)
}

func TestDTSExtractorDuplicateRecoveryPoint(t *testing.T) {
	// two recovery point SEIs in the same access unit are harmless
	ex := NewDTSExtractor()
	dts, err := ex.Extract(
		[][]byte{recoveryTestSPS, recoveryTestSEI, recoveryTestSEI, recoveryTestPOC0},
		3000,
	)
	require.NoError(t, err)
	require.Equal(t, int64(3000), dts)

	// a recovery point flag must not leak into the next access unit
	dts, err = ex.Extract([][]byte{recoveryTestPoc4}, 6000)
	require.NoError(t, err)
	require.Equal(t, int64(4500), dts)
}

func TestDTSExtractorNoRandomAccess(t *testing.T) {
	ex := NewDTSExtractor()

	// a non-IDR without an IDR or recovery point is rejected
	_, err := ex.Extract([][]byte{recoveryTestSPS, recoveryTestPOC0}, 1000)
	require.EqualError(t, err, "random access frame not received yet")

	// a non-recovery SEI does not provide a random access point
	_, err = ex.Extract([][]byte{recoveryTestSPS, recoveryTestOtherSEI, recoveryTestPOC0}, 1000)
	require.EqualError(t, err, "random access frame not received yet")
}

func TestDTSExtractorStandaloneSEIAU(t *testing.T) {
	ex := NewDTSExtractor()

	// an access unit made of parameter sets / SEIs only cannot
	// produce a DTS, but must not arm any flag for the next unit
	_, err := ex.Extract([][]byte{recoveryTestSPS, recoveryTestSEI}, 1000)
	require.EqualError(t, err, "access unit doesn't contain an IDR or non-IDR NALU")

	// the recovery point was not consumed, so it still applies to
	// the first image that follows it.
	dts, err := ex.Extract([][]byte{recoveryTestSEI, recoveryTestPOC0}, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(2000), dts)
}

func TestDTSExtractorEmptyAUAndNALU(t *testing.T) {
	ex := NewDTSExtractor()

	_, err := ex.Extract([][]byte{}, 1000)
	require.EqualError(t, err, "access unit doesn't contain an IDR or non-IDR NALU")

	_, err = ex.Extract([][]byte{{}}, 1000)
	require.EqualError(t, err, "empty NALU")
}

func TestDTSExtractorInvalidRecoverySEI(t *testing.T) {
	ex := NewDTSExtractor()

	// a malformed recovery point SEI returns an error, not a panic
	_, err := ex.Extract([][]byte{recoveryTestSPS, recoveryTestBadSEI, recoveryTestPOC0}, 1000)
	require.Error(t, err)

	// the extractor is still usable with a valid stream afterwards
	ex2 := NewDTSExtractor()
	dts, err := ex2.Extract([][]byte{recoveryTestSPS, recoveryTestIDR}, 1000)
	require.NoError(t, err)
	require.Equal(t, int64(1000), dts)
}

func TestDTSExtractorRecoveryPointMultipleSlices(t *testing.T) {
	// only one image NALU participates in the computation; an IDR
	// takes precedence even if a non-IDR is present first.
	ex := NewDTSExtractor()
	dts, err := ex.Extract(
		[][]byte{recoveryTestSPS, recoveryTestSEI, recoveryTestPOC0, recoveryTestIDR},
		4000,
	)
	require.NoError(t, err)
	require.Equal(t, int64(4000), dts)
}

func TestDTSExtractorRecoveryPointDoesNotPanic(t *testing.T) {
	// a broad sweep of malformed inputs must return errors instead
	// of panicking.
	inputs := [][][]byte{
		{recoveryTestSPS, {0x06}},
		{recoveryTestSPS, {0x06, 0x06}},
		{recoveryTestSPS, {0x06, 0x06, 0x01}},
		{recoveryTestSPS, recoveryTestBadSEI},
		{{0x06, 0x06, 0x01, 0x80}, recoveryTestPOC0}, // SEI before SPS
	}

	for _, au := range inputs {
		ex := NewDTSExtractor()
		func() {
			defer func() {
				require.Nil(t, recover())
			}()
			_, _ = ex.Extract(au, 1000)
		}()
	}
}

func TestDTSExtractorRecoveryPointAfterSPSChange(t *testing.T) {
	// a new SPS resets the random access state; a recovery point
	// received afterwards, before any image of the new sequence,
	// must be able to establish a fresh random access point.
	sps2 := []byte{
		0x67, 0x42, 0xc0, 0x1e, 0x8c, 0x8d, 0x40, 0x50,
		0x17, 0xfc, 0xb0, 0x0f, 0x08, 0x84, 0x6a,
	}

	ex := NewDTSExtractor()

	// first sequence starts with an IDR
	dts, err := ex.Extract([][]byte{recoveryTestSPS, recoveryTestIDR}, 1000)
	require.NoError(t, err)
	require.Equal(t, int64(1000), dts)

	// an ordinary non-IDR marks an image as already seen
	_, err = ex.Extract([][]byte{recoveryTestPoc4}, 2000)
	require.NoError(t, err)

	// the SPS change resets imageSeen; the recovery point that
	// follows is therefore accepted for the new sequence.
	dts, err = ex.Extract([][]byte{sps2, recoveryTestSEI, recoveryTestPOC0}, 3000)
	require.NoError(t, err)
	require.Equal(t, int64(3000), dts)
}

func TestDTSExtractorRecoveryPointPicOrderCntType2(t *testing.T) {
	// with pic_order_cnt_type=2 the extractor works in passthrough
	// mode and DTS always equals PTS; the recovery point must not
	// break that behavior.
	sps := []byte{
		0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02,
		0x27, 0xe5, 0x84, 0x00, 0x00, 0x03, 0x00, 0x04,
		0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9, 0x20,
	}

	ex := NewDTSExtractor()
	dts, err := ex.Extract([][]byte{sps, recoveryTestSEI, recoveryTestPOC0}, 1000)
	require.NoError(t, err)
	require.Equal(t, int64(1000), dts)

	dts, err = ex.Extract([][]byte{recoveryTestPoc4}, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(2000), dts)
}
