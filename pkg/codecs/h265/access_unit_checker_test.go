package h265

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Fixed byte sequences used by the tests below.
// They are taken from a real H265 stream and are left untouched, so that
// the tests exercise the actual NALU layout (2-byte header + payload).
var (
	auCkVPS = []byte{
		0x40, 0x01, 0x0c, 0x01, 0xff, 0xff, 0x01, 0x60,
		0x00, 0x00, 0x03, 0x00, 0x90, 0x00, 0x00, 0x03,
		0x00, 0x00, 0x03, 0x00, 0x78, 0x99, 0x98, 0x09,
	}

	auCkSPS = []byte{
		0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00, 0x03,
		0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
		0x00, 0x78, 0xa0, 0x03, 0xc0, 0x80, 0x10, 0xe5,
		0x96, 0x66, 0x69, 0x24, 0xca, 0xe0, 0x10, 0x00,
		0x00, 0x03, 0x00, 0x10, 0x00, 0x00, 0x03, 0x01,
		0xe0, 0x80,
	}

	// same profile as auCkSPS, different picture size (1920x800).
	auCkSPS2 = []byte{
		0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00, 0x03,
		0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
		0x00, 0x78, 0xa0, 0x03, 0xc0, 0x80, 0x32, 0x16,
		0x59, 0x59, 0xa4, 0x93, 0x2b, 0xc0, 0x5a, 0x80,
		0x80, 0x80, 0x82, 0x00, 0x00, 0x07, 0xd2, 0x00,
		0x00, 0xbb, 0x80, 0x10,
	}

	auCkPPS = []byte{
		0x44, 0x01, 0xc1, 0x72, 0xb4, 0x62, 0x40,
	}

	// valid PPS with the same IDs as auCkPPS but different num_extra_slice_header_bits.
	auCkPPS2 = []byte{
		0x44, 0x01, 0xc3, 0x72, 0xb4, 0x62, 0x40,
	}

	// IDR_W_RADL slice, referencing PPS 0.
	auCkIDR = []byte{
		0x26, 0x01, 0xaf, 0x08, 0x42, 0x23, 0x48, 0x8a,
		0x43, 0xe2,
	}

	// same slice of auCkIDR, with the NALU type changed to CRA_NUT (21).
	auCkCRA = []byte{
		0x2a, 0x01, 0xaf, 0x08, 0x42, 0x23, 0x48, 0x8a,
		0x43, 0xe2,
	}

	// same slice of auCkIDR, with the NALU type changed to BLA_W_LP (16).
	auCkBLA = []byte{
		0x20, 0x01, 0xaf, 0x08, 0x42, 0x23, 0x48, 0x8a,
		0x43, 0xe2,
	}

	// TRAIL_R slice.
	auCkTRAIL = []byte{
		0x02, 0x01, 0xd0, 0x19, 0x5f, 0x8c, 0xb4, 0x42,
		0x49, 0x20, 0x40, 0x11, 0x16, 0x92, 0x93, 0xea,
		0x54, 0x57, 0x4e, 0x0a,
	}

	auCkAUD = []byte{0x46, 0x01, 0x50}

	auCkSEI = []byte{0x4e, 0x01, 0x05, 0x02, 0x00, 0x00, 0x80}

	auCkEOS = []byte{0x48, 0x01}
)

func TestH265RandomAccessTypeString(t *testing.T) {
	require.Equal(t, "none", RandomAccessTypeNone.String())
	require.Equal(t, "CRA", RandomAccessTypeCRA.String())
	require.Equal(t, "BLA", RandomAccessTypeBLA.String())
	require.Equal(t, "IDR", RandomAccessTypeIDR.String())
	require.Equal(t, "unknown (100)", RandomAccessType(100).String())
}

func TestH265NALUTypeIsVCL(t *testing.T) {
	for i := 0; i <= 31; i++ {
		require.Equal(t, true, NALUType(i).IsVCL(), "type %d", i)
	}

	for _, v := range []NALUType{32, 33, 34, 35, 36, 37, 38, 39, 40, 47, 48, 63} {
		require.Equal(t, false, v.IsVCL(), "type %d", v)
	}
}

func TestH265AccessUnitCheckerEmpty(t *testing.T) {
	var c AccessUnitChecker
	info, err := c.Check(nil)
	require.ErrorIs(t, err, ErrEmptyAccessUnit)
	require.Zero(t, info)

	info, err = c.Check([][]byte{})
	require.ErrorIs(t, err, ErrEmptyAccessUnit)
	require.Zero(t, info)
}

func TestH265AccessUnitCheckerTruncatedNALU(t *testing.T) {
	var c AccessUnitChecker

	for _, nalu := range [][]byte{{}, {0x26}} {
		_, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, nalu})
		require.ErrorIs(t, err, ErrTruncatedNALU)

		// a rejected access unit must not change the state
		require.Nil(t, c.VPS())
		require.Nil(t, c.SPS())
		require.Nil(t, c.PPS())
	}

	// the checker keeps working after a rejection
	info, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, auCkIDR})
	require.NoError(t, err)
	require.True(t, info.HasVCL)
	require.Equal(t, RandomAccessTypeIDR, info.RandomAccessType)
}

func TestH265AccessUnitCheckerIDR(t *testing.T) {
	var c AccessUnitChecker

	info, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, auCkIDR})
	require.NoError(t, err)
	require.Equal(t, AccessUnitInfo{
		HasVCL:           true,
		RandomAccessType: RandomAccessTypeIDR,
		VPS:              auCkVPS,
		SPS:              auCkSPS,
		PPS:              auCkPPS,
		VPSChanged:       true,
		SPSChanged:       true,
		PPSChanged:       true,
	}, info)
	require.True(t, info.IsRandomAccess())
	require.True(t, info.ParameterSetsChanged())

	// the state is stored across frames
	require.Equal(t, auCkVPS, c.VPS())
	require.Equal(t, auCkSPS, c.SPS())
	require.Equal(t, auCkPPS, c.PPS())
}

func TestH265AccessUnitCheckerCRA(t *testing.T) {
	var c AccessUnitChecker

	_, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, auCkIDR})
	require.NoError(t, err)

	info, err := c.Check([][]byte{auCkCRA})
	require.NoError(t, err)
	require.True(t, info.HasVCL)
	require.Equal(t, RandomAccessTypeCRA, info.RandomAccessType)
	require.True(t, info.IsRandomAccess())
	require.Nil(t, info.VPS)
	require.Nil(t, info.SPS)
	require.Nil(t, info.PPS)
	require.False(t, info.ParameterSetsChanged())
}

func TestH265AccessUnitCheckerBLA(t *testing.T) {
	var c AccessUnitChecker

	for _, typ := range []NALUType{
		NALUType_BLA_W_LP,
		NALUType_BLA_W_RADL,
		NALUType_BLA_N_LP,
	} {
		alu := append([]byte(nil), auCkBLA...)
		alu[0] = byte(typ) << 1

		info, err := c.Check([][]byte{alu})
		require.NoError(t, err)
		require.Equal(t, RandomAccessTypeBLA, info.RandomAccessType, typ.String())
		require.True(t, info.IsRandomAccess())
	}

	// reserved IRAP types are treated as the weakest IRAP type.
	for _, typ := range []NALUType{NALUType_RSV_IRAP_VCL22, NALUType_RSV_IRAP_VCL23} {
		alu := append([]byte(nil), auCkCRA...)
		alu[0] = byte(typ) << 1

		info, err := c.Check([][]byte{alu})
		require.NoError(t, err)
		require.Equal(t, RandomAccessTypeCRA, info.RandomAccessType)
	}
}

func TestH265AccessUnitCheckerNonRandomAccessFrame(t *testing.T) {
	var c AccessUnitChecker

	info, err := c.Check([][]byte{auCkTRAIL})
	require.NoError(t, err)
	require.True(t, info.HasVCL)
	require.Equal(t, RandomAccessTypeNone, info.RandomAccessType)
	require.False(t, info.IsRandomAccess())
	require.False(t, info.ParameterSetsChanged())

	// no parameter set has been seen yet, so the frame can't be decoded on its own.
	require.True(t, info.MissingParameterSets)
}

func TestH265AccessUnitCheckerStrongestRandomAccessType(t *testing.T) {
	var c AccessUnitChecker

	info, err := c.Check([][]byte{auCkTRAIL, auCkCRA, auCkIDR})
	require.NoError(t, err)
	require.Equal(t, RandomAccessTypeIDR, info.RandomAccessType)

	info, err = c.Check([][]byte{auCkIDR, auCkCRA})
	require.NoError(t, err)
	require.Equal(t, RandomAccessTypeIDR, info.RandomAccessType)
}

func TestH265AccessUnitCheckerNonVCL(t *testing.T) {
	var c AccessUnitChecker

	// AUD/SEI/EOS are not pictures and must not be reported as frames.
	info, err := c.Check([][]byte{auCkAUD, auCkSEI, auCkEOS})
	require.NoError(t, err)
	require.False(t, info.HasVCL)
	require.Equal(t, RandomAccessTypeNone, info.RandomAccessType)
	require.False(t, info.IsRandomAccess())
	require.False(t, info.ParameterSetsChanged())

	// a parameter-set-only access unit is not a picture either.
	info, err = c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS})
	require.NoError(t, err)
	require.False(t, info.HasVCL)
	require.True(t, info.ParameterSetsChanged())
	require.False(t, info.MissingParameterSets)
}

func TestH265AccessUnitCheckerRepeatedParameterSets(t *testing.T) {
	var c AccessUnitChecker

	first, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, auCkIDR})
	require.NoError(t, err)
	require.True(t, first.ParameterSetsChanged())

	// all parameter sets are repeated with the same content.
	second, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, auCkTRAIL})
	require.NoError(t, err)
	require.True(t, second.HasVCL)
	require.False(t, second.VPSChanged)
	require.False(t, second.SPSChanged)
	require.False(t, second.PPSChanged)
	require.False(t, second.ParameterSetsChanged())

	// a frame without parameter sets doesn't report changes.
	third, err := c.Check([][]byte{auCkTRAIL})
	require.NoError(t, err)
	require.False(t, third.ParameterSetsChanged())

	// only the parameter set that changed is reported.
	fourth, err := c.Check([][]byte{auCkSPS2, auCkPPS2, auCkIDR})
	require.NoError(t, err)
	require.False(t, fourth.VPSChanged)
	require.True(t, fourth.SPSChanged)
	require.True(t, fourth.PPSChanged)
	require.True(t, fourth.ParameterSetsChanged())
	require.Equal(t, auCkSPS2, c.SPS())
	require.Equal(t, auCkPPS2, c.PPS())
	require.Equal(t, auCkVPS, c.VPS())
}

func TestH265AccessUnitCheckerParameterSetsAfterSlice(t *testing.T) {
	var c AccessUnitChecker

	// a slice that precedes the parameter sets is still a picture and the
	// parameter sets are still collected and stored.
	info, err := c.Check([][]byte{auCkIDR, auCkVPS, auCkSPS, auCkPPS})
	require.NoError(t, err)
	require.True(t, info.HasVCL)
	require.Equal(t, RandomAccessTypeIDR, info.RandomAccessType)
	require.Equal(t, auCkVPS, info.VPS)
	require.Equal(t, auCkSPS, info.SPS)
	require.Equal(t, auCkPPS, info.PPS)
	require.True(t, info.ParameterSetsChanged())
	require.False(t, info.MissingParameterSets)
	require.Equal(t, auCkSPS, c.SPS())

	// parameter sets stored by a later access unit are available to the
	// following access units.
	info, err = c.Check([][]byte{auCkTRAIL})
	require.NoError(t, err)
	require.False(t, info.ParameterSetsChanged())
	require.False(t, info.MissingParameterSets)
}

func TestH265AccessUnitCheckerSliceBeforeParameterSets(t *testing.T) {
	var c AccessUnitChecker

	// the slice comes before the parameter sets that configure it: the picture is
	// reported, the parameter sets are collected, but at the time the access unit is
	// processed the configuration is incomplete.
	info, err := c.Check([][]byte{auCkIDR, auCkSPS})
	require.NoError(t, err)
	require.True(t, info.HasVCL)
	require.Equal(t, RandomAccessTypeIDR, info.RandomAccessType)
	require.Equal(t, auCkSPS, info.SPS)
	require.Nil(t, info.VPS)
	require.Nil(t, info.PPS)
	require.True(t, info.SPSChanged)
	require.True(t, info.MissingParameterSets)

	// the rest of the configuration arrives with the next access unit, which becomes
	// the first one with a complete configuration.
	info, err = c.Check([][]byte{auCkVPS, auCkPPS, auCkTRAIL})
	require.NoError(t, err)
	require.True(t, info.VPSChanged)
	require.True(t, info.PPSChanged)
	require.False(t, info.SPSChanged)
	require.False(t, info.MissingParameterSets)

	// missing parameter sets are reported again when the checker is reset.
	c.Initialize()
	info, err = c.Check([][]byte{auCkIDR})
	require.NoError(t, err)
	require.True(t, info.MissingParameterSets)
}

func TestH265AccessUnitCheckerContradictoryParameterSets(t *testing.T) {
	var c AccessUnitChecker

	cases := []struct {
		name string
		au   [][]byte
	}{
		{"VPS", [][]byte{auCkVPS, append(append([]byte(nil), auCkVPS...), 0x00)}},
		{"SPS", [][]byte{auCkSPS, auCkSPS2}},
		{"PPS", [][]byte{auCkPPS, auCkPPS2}},
	}

	for _, ca := range cases {
		t.Run(ca.name, func(t *testing.T) {
			_, err := c.Check(append(ca.au, auCkIDR))
			require.ErrorIs(t, err, ErrContradictoryParameterSets)

			// nothing is committed
			require.Nil(t, c.VPS())
			require.Nil(t, c.SPS())
			require.Nil(t, c.PPS())
		})
	}

	// identical duplicates are accepted and stored once.
	info, err := c.Check([][]byte{auCkVPS, auCkVPS, auCkSPS, auCkSPS, auCkPPS, auCkPPS, auCkIDR})
	require.NoError(t, err)
	require.True(t, info.ParameterSetsChanged())
	require.Equal(t, auCkSPS, c.SPS())
}

func TestH265AccessUnitCheckerInvalidParameterSets(t *testing.T) {
	var c AccessUnitChecker

	// an invalid parameter set rejects the access unit.
	_, err := c.Check([][]byte{auCkVPS, {0x42, 0x01}, auCkPPS, auCkIDR})
	require.ErrorIs(t, err, ErrInvalidParameterSet)

	_, err = c.Check([][]byte{auCkVPS, auCkSPS, {0x44, 0x01}, auCkIDR})
	require.ErrorIs(t, err, ErrInvalidParameterSet)

	_, err = c.Check([][]byte{{0x40, 0x01}, auCkSPS, auCkPPS, auCkIDR})
	require.ErrorIs(t, err, ErrInvalidParameterSet)

	// the state is untouched.
	require.Nil(t, c.VPS())
	require.Nil(t, c.SPS())
	require.Nil(t, c.PPS())

	// a valid access unit is accepted afterwards.
	info, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, auCkIDR})
	require.NoError(t, err)
	require.True(t, info.ParameterSetsChanged())
}

func TestH265AccessUnitCheckerInvalidParameterSetKeepsPrevious(t *testing.T) {
	var c AccessUnitChecker

	_, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, auCkIDR})
	require.NoError(t, err)

	// an invalid SPS doesn't replace the previous one.
	_, err = c.Check([][]byte{{0x42, 0x01}, auCkIDR})
	require.ErrorIs(t, err, ErrInvalidParameterSet)
	require.Equal(t, auCkSPS, c.SPS())

	// the previous parameter sets are still available.
	info, err := c.Check([][]byte{auCkIDR})
	require.NoError(t, err)
	require.False(t, info.SPSChanged)
}

func TestH265AccessUnitCheckerUnchangedParameterSetsAreNotRevalidated(t *testing.T) {
	var c AccessUnitChecker

	_, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS})
	require.NoError(t, err)

	// the SPS is already stored and valid, so it is not parsed again.
	info, err := c.Check([][]byte{auCkSPS})
	require.NoError(t, err)
	require.False(t, info.SPSChanged)
}

func TestH265AccessUnitCheckerStoresAPrivateCopy(t *testing.T) {
	var c AccessUnitChecker

	vps := append([]byte(nil), auCkVPS...)
	sps := append([]byte(nil), auCkSPS...)
	pps := append([]byte(nil), auCkPPS...)

	_, err := c.Check([][]byte{vps, sps, pps})
	require.NoError(t, err)

	vps[0] = 0x00
	sps[0] = 0x00
	pps[0] = 0x00

	require.Equal(t, auCkVPS, c.VPS())
	require.Equal(t, auCkSPS, c.SPS())
	require.Equal(t, auCkPPS, c.PPS())
}

func TestH265AccessUnitCheckerInitialize(t *testing.T) {
	var c AccessUnitChecker

	_, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, auCkIDR})
	require.NoError(t, err)

	c.Initialize()
	require.Nil(t, c.VPS())
	require.Nil(t, c.SPS())
	require.Nil(t, c.PPS())

	info, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, auCkIDR})
	require.NoError(t, err)
	require.True(t, info.ParameterSetsChanged())
}

func TestH265AccessUnitCheckerMultipleSlices(t *testing.T) {
	var c AccessUnitChecker

	// multiple slice segments of the same picture belong to a single access unit
	// and don't change the result.
	info, err := c.Check([][]byte{auCkVPS, auCkSPS, auCkPPS, auCkIDR, auCkIDR})
	require.NoError(t, err)
	require.True(t, info.HasVCL)
	require.Equal(t, RandomAccessTypeIDR, info.RandomAccessType)
}

func FuzzAccessUnitChecker(f *testing.F) {
	f.Add([]byte{0x26, 0x01, 0xaf}, []byte{0x40, 0x01}, []byte{0x42, 0x01}, []byte{0x44, 0x01})
	f.Add([]byte{0x2a, 0x01}, []byte{0x42, 0x01, 0x01}, []byte{}, []byte{0x44})
	f.Add([]byte{}, []byte{}, []byte{}, []byte{})

	f.Fuzz(func(_ *testing.T, idr, vps, sps, pps []byte) {
		var c AccessUnitChecker

		// the checker must not panic on arbitrary input, and a rejected access
		// unit must not leave a partial state behind.
		for range 2 {
			_, _ = c.Check([][]byte{idr, vps, sps, pps}) //nolint:errcheck
		}
	})
}
