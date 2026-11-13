package av1

import (
	"testing"

	"github.com/bluenviron/mediacommon/v2/pkg/bits"
	"github.com/stretchr/testify/require"
)

// buildSequenceHeader builds a sequence header OBU (without size field)
// by writing bits with the given function.
// If padTo is greater than the marshaled payload size, the payload
// is padded with zero bytes up to padTo bytes.
func buildSequenceHeader(padTo int, write func(buf []byte, pos *int)) []byte {
	payload := make([]byte, 8192)
	pos := 0
	write(payload, &pos)
	n := (pos + 7) / 8
	if padTo > n {
		n = padTo
	}
	out := make([]byte, 1+n)
	out[0] = 0x08 // sequence header OBU, no size field
	copy(out[1:], payload[:n])
	return out
}

var casesSequenceHeader = []struct {
	name   string
	byts   []byte
	sh     SequenceHeader
	width  int
	height int
}{
	{
		"chrome webrtc",
		[]byte{
			8, 0, 0, 0, 66, 167, 191, 228, 96, 13, 0, 64,
		},
		SequenceHeader{
			OperatingPointIdc:              []uint16{0},
			SeqLevelIdx:                    []uint8{8},
			SeqTier:                        []bool{false},
			DecoderModelPresentForThisOp:   []bool{false},
			InitialDisplayPresentForThisOp: []bool{false},
			InitialDisplayDelayMinus1:      []uint8{0},
			MaxFrameWidthMinus1:            1919,
			MaxFrameHeightMinus1:           803,
			SeqChooseScreenContentTools:    true,
			SeqForceScreenContentTools:     2,
			SeqChooseIntegerMv:             true,
			SeqForceIntegerMv:              2,
			EnableCdef:                     true,
			ColorConfig: SequenceHeader_ColorConfig{
				BitDepth:                8,
				ColorPrimaries:          2,
				TransferCharacteristics: 2,
				MatrixCoefficients:      2,
				SubsamplingX:            true,
				SubsamplingY:            true,
			},
		},
		1920,
		804,
	},
	{
		"av1 sample",
		[]byte{
			10, 11, 0, 0, 0, 66, 167, 191, 230, 46, 223, 200, 66,
		},
		SequenceHeader{
			OperatingPointIdc:              []uint16{0},
			SeqLevelIdx:                    []uint8{8},
			SeqTier:                        []bool{false},
			DecoderModelPresentForThisOp:   []bool{false},
			InitialDisplayPresentForThisOp: []bool{false},
			InitialDisplayDelayMinus1:      []uint8{0},
			MaxFrameWidthMinus1:            1919,
			MaxFrameHeightMinus1:           817,
			Use128x128Superblock:           true,
			EnableFilterIntra:              true,
			EnableIntraEdgeFilter:          true,
			EnableMaskedCompound:           true,
			EnableWarpedMotion:             true,
			EnableOrderHint:                true,
			EnableJntComp:                  true,
			EnableRefFrameMvs:              true,
			SeqChooseScreenContentTools:    true,
			SeqForceScreenContentTools:     2,
			SeqChooseIntegerMv:             true,
			OrderHintBitsMinus1:            6,
			SeqForceIntegerMv:              2,
			EnableCdef:                     true,
			ColorConfig: SequenceHeader_ColorConfig{
				BitDepth:                8,
				ColorPrimaries:          2,
				TransferCharacteristics: 2,
				MatrixCoefficients:      2,
				ColorRange:              true,
				SubsamplingX:            true,
				SubsamplingY:            true,
			},
		},
		1920,
		818,
	},
	{
		"libsvtav1",
		[]byte{
			0x8, 0x0, 0x0, 0x0, 0x42, 0xab, 0xbf, 0xc3, 0x71, 0xab, 0xe6, 0x1,
		},
		SequenceHeader{
			OperatingPointIdc:              []uint16{0},
			SeqLevelIdx:                    []uint8{8},
			SeqTier:                        []bool{false},
			DecoderModelPresentForThisOp:   []bool{false},
			InitialDisplayPresentForThisOp: []bool{false},
			InitialDisplayDelayMinus1:      []uint8{0},
			MaxFrameWidthMinus1:            1919,
			MaxFrameHeightMinus1:           1079,
			EnableIntraEdgeFilter:          true,
			EnableInterintraCompound:       true,
			EnableWarpedMotion:             true,
			EnableOrderHint:                true,
			EnableRefFrameMvs:              true,
			SeqChooseScreenContentTools:    true,
			SeqForceScreenContentTools:     2,
			SeqChooseIntegerMv:             true,
			SeqForceIntegerMv:              2,
			OrderHintBitsMinus1:            6,
			EnableCdef:                     true,
			EnableRestoration:              true,
			ColorConfig: SequenceHeader_ColorConfig{
				BitDepth:                8,
				ColorPrimaries:          2,
				TransferCharacteristics: 2,
				MatrixCoefficients:      2,
				SubsamplingX:            true,
				SubsamplingY:            true,
			},
		},
		1920,
		1080,
	},
	{
		"amd hardware av1",
		[]byte{
			0x08, 0x04, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00,
			0x00, 0xf3, 0x00, 0x00, 0x0e, 0x55, 0x77, 0xf8,
			0x73, 0xd0, 0x02, 0x7d, 0x10, 0x10, 0x10, 0x10,
			0x40,
		},
		SequenceHeader{
			TimingInfo: &SequenceHeader_TimingInfo{
				NumUnitsInDisplayTick: 1,
				TimeScale:             60,
				EqualPictureInterval:  true,
			},
			OperatingPointIdc:              []uint16{0},
			SeqLevelIdx:                    []uint8{0x0e},
			SeqTier:                        []bool{false},
			DecoderModelPresentForThisOp:   []bool{false},
			InitialDisplayPresentForThisOp: []bool{false},
			InitialDisplayDelayMinus1:      []uint8{0},
			MaxFrameWidthMinus1:            1919,
			MaxFrameHeightMinus1:           1081,
			FrameIDNumbersPresentFlag:      true,
			DeltaFrameIDLengthMinus2:       13,
			EnableOrderHint:                true,
			SeqChooseScreenContentTools:    true,
			SeqForceScreenContentTools:     SequenceHeader_SeqForceScreenContentTools(2),
			SeqChooseIntegerMv:             true,
			SeqForceIntegerMv:              2,
			OrderHintBitsMinus1:            7,
			EnableCdef:                     true,
			ColorConfig: SequenceHeader_ColorConfig{
				BitDepth:                    8,
				ColorDescriptionPresentFlag: true,
				ColorPrimaries:              1,
				TransferCharacteristics:     SequenceHeader_TransferCharacteristics(1),
				MatrixCoefficients:          SequenceHeader_MatrixCoefficients(1),
				SubsamplingX:                true,
				SubsamplingY:                true,
			},
		},
		1920,
		1082,
	},
	{
		"libaom initial display delay",
		[]byte{
			0x0a, 0x14, 0x04, 0x00, 0x00, 0x00, 0x04, 0x00,
			0x00, 0x00, 0x7b, 0x40, 0x00, 0x05, 0xbd, 0x4c,
			0xff, 0xb3, 0xcc, 0xaf, 0x90, 0x04,
		},
		SequenceHeader{
			TimingInfo: &SequenceHeader_TimingInfo{
				NumUnitsInDisplayTick: 1,
				TimeScale:             30,
				EqualPictureInterval:  true,
			},
			InitialDisplayDelayPresentFlag: true,
			OperatingPointIdc:              []uint16{0},
			SeqLevelIdx:                    []uint8{5},
			SeqTier:                        []bool{false},
			DecoderModelPresentForThisOp:   []bool{false},
			InitialDisplayPresentForThisOp: []bool{true},
			InitialDisplayDelayMinus1:      []uint8{7},
			MaxFrameWidthMinus1:            1279,
			MaxFrameHeightMinus1:           719,
			EnableFilterIntra:              true,
			EnableIntraEdgeFilter:          true,
			EnableWarpedMotion:             true,
			EnableOrderHint:                true,
			EnableRefFrameMvs:              true,
			SeqChooseScreenContentTools:    true,
			SeqForceScreenContentTools:     2,
			SeqChooseIntegerMv:             true,
			SeqForceIntegerMv:              2,
			OrderHintBitsMinus1:            6,
			EnableCdef:                     true,
			ColorConfig: SequenceHeader_ColorConfig{
				BitDepth:                8,
				ColorPrimaries:          2,
				TransferCharacteristics: 2,
				MatrixCoefficients:      2,
				SubsamplingX:            true,
				SubsamplingY:            true,
			},
		},
		1280,
		720,
	},
	{
		"reduced still picture",
		buildSequenceHeader(0, func(buf []byte, pos *int) {
			bits.WriteBitsUnsafe(buf, pos, 0, 3)     // seq_profile
			bits.WriteFlagUnsafe(buf, pos, true)     // still_picture
			bits.WriteFlagUnsafe(buf, pos, true)     // reduced_still_picture_header
			bits.WriteBitsUnsafe(buf, pos, 8, 5)     // seq_level_idx[0]
			bits.WriteBitsUnsafe(buf, pos, 10, 4)    // frame_width_bits_minus_1
			bits.WriteBitsUnsafe(buf, pos, 9, 4)     // frame_height_bits_minus_1
			bits.WriteBitsUnsafe(buf, pos, 1279, 11) // max_frame_width_minus_1
			bits.WriteBitsUnsafe(buf, pos, 719, 10)  // max_frame_height_minus_1
			bits.WriteFlagUnsafe(buf, pos, false)    // use_128x128_superblock
			bits.WriteFlagUnsafe(buf, pos, false)    // enable_filter_intra
			bits.WriteFlagUnsafe(buf, pos, false)    // enable_intra_edge_filter
			bits.WriteFlagUnsafe(buf, pos, false)    // enable_superres
			bits.WriteFlagUnsafe(buf, pos, true)     // enable_cdef
			bits.WriteFlagUnsafe(buf, pos, false)    // enable_restoration
			bits.WriteFlagUnsafe(buf, pos, false)    // high_bitdepth
			bits.WriteFlagUnsafe(buf, pos, false)    // mono_chrome
			bits.WriteFlagUnsafe(buf, pos, false)    // color_description_present_flag
			bits.WriteFlagUnsafe(buf, pos, false)    // color_range
			bits.WriteBitsUnsafe(buf, pos, 0, 2)     // chroma_sample_position
			bits.WriteFlagUnsafe(buf, pos, false)    // film_grain_params_present
		}),
		SequenceHeader{
			StillPicture:                   true,
			ReducedStillPictureHeader:      true,
			OperatingPointIdc:              []uint16{0},
			SeqLevelIdx:                    []uint8{8},
			SeqTier:                        []bool{false},
			DecoderModelPresentForThisOp:   []bool{false},
			InitialDisplayPresentForThisOp: []bool{false},
			InitialDisplayDelayMinus1:      []uint8{0},
			MaxFrameWidthMinus1:            1279,
			MaxFrameHeightMinus1:           719,
			SeqForceScreenContentTools:     2,
			SeqForceIntegerMv:              2,
			EnableCdef:                     true,
			ColorConfig: SequenceHeader_ColorConfig{
				BitDepth:                8,
				ColorPrimaries:          2,
				TransferCharacteristics: 2,
				MatrixCoefficients:      2,
				SubsamplingX:            true,
				SubsamplingY:            true,
			},
		},
		1280,
		720,
	},
	{
		"multiple operating points with initial display delay",
		buildSequenceHeader(0, func(buf []byte, pos *int) {
			bits.WriteBitsUnsafe(buf, pos, 0, 3)      // seq_profile
			bits.WriteFlagUnsafe(buf, pos, false)     // still_picture
			bits.WriteFlagUnsafe(buf, pos, false)     // reduced_still_picture_header
			bits.WriteFlagUnsafe(buf, pos, false)     // timing_info_present_flag
			bits.WriteFlagUnsafe(buf, pos, true)      // initial_display_delay_present_flag
			bits.WriteBitsUnsafe(buf, pos, 1, 5)      // operating_points_cnt_minus_1
			bits.WriteBitsUnsafe(buf, pos, 0, 12)     // operating_point_idc[0]
			bits.WriteBitsUnsafe(buf, pos, 8, 5)      // seq_level_idx[0]
			bits.WriteFlagUnsafe(buf, pos, false)     // seq_tier[0]
			bits.WriteFlagUnsafe(buf, pos, true)      // initial_display_delay_present_for_this_op[0]
			bits.WriteBitsUnsafe(buf, pos, 7, 4)      // initial_display_delay_minus_1[0]
			bits.WriteBitsUnsafe(buf, pos, 0x123, 12) // operating_point_idc[1]
			bits.WriteBitsUnsafe(buf, pos, 5, 5)      // seq_level_idx[1]
			bits.WriteFlagUnsafe(buf, pos, false)     // initial_display_delay_present_for_this_op[1]
			bits.WriteBitsUnsafe(buf, pos, 10, 4)     // frame_width_bits_minus_1
			bits.WriteBitsUnsafe(buf, pos, 9, 4)      // frame_height_bits_minus_1
			bits.WriteBitsUnsafe(buf, pos, 1279, 11)  // max_frame_width_minus_1
			bits.WriteBitsUnsafe(buf, pos, 719, 10)   // max_frame_height_minus_1
			bits.WriteFlagUnsafe(buf, pos, false)     // frame_id_numbers_present_flag
			bits.WriteFlagUnsafe(buf, pos, false)     // use_128x128_superblock
			bits.WriteFlagUnsafe(buf, pos, false)     // enable_filter_intra
			bits.WriteFlagUnsafe(buf, pos, false)     // enable_intra_edge_filter
			bits.WriteFlagUnsafe(buf, pos, false)     // enable_interintra_compound
			bits.WriteFlagUnsafe(buf, pos, false)     // enable_masked_compound
			bits.WriteFlagUnsafe(buf, pos, false)     // enable_warped_motion
			bits.WriteFlagUnsafe(buf, pos, false)     // enable_dual_filter
			bits.WriteFlagUnsafe(buf, pos, false)     // enable_order_hint
			bits.WriteFlagUnsafe(buf, pos, true)      // seq_choose_screen_content_tools
			bits.WriteFlagUnsafe(buf, pos, true)      // seq_choose_integer_mv
			bits.WriteFlagUnsafe(buf, pos, false)     // enable_superres
			bits.WriteFlagUnsafe(buf, pos, true)      // enable_cdef
			bits.WriteFlagUnsafe(buf, pos, false)     // enable_restoration
			bits.WriteFlagUnsafe(buf, pos, false)     // high_bitdepth
			bits.WriteFlagUnsafe(buf, pos, false)     // mono_chrome
			bits.WriteFlagUnsafe(buf, pos, false)     // color_description_present_flag
			bits.WriteFlagUnsafe(buf, pos, false)     // color_range
			bits.WriteBitsUnsafe(buf, pos, 0, 2)      // chroma_sample_position
			bits.WriteFlagUnsafe(buf, pos, false)     // film_grain_params_present
		}),
		SequenceHeader{
			InitialDisplayDelayPresentFlag: true,
			OperatingPointsCntMinus1:       1,
			OperatingPointIdc:              []uint16{0, 0x123},
			SeqLevelIdx:                    []uint8{8, 5},
			SeqTier:                        []bool{false, false},
			DecoderModelPresentForThisOp:   []bool{false, false},
			InitialDisplayPresentForThisOp: []bool{true, false},
			InitialDisplayDelayMinus1:      []uint8{7, 0},
			MaxFrameWidthMinus1:            1279,
			MaxFrameHeightMinus1:           719,
			SeqChooseScreenContentTools:    true,
			SeqForceScreenContentTools:     2,
			SeqChooseIntegerMv:             true,
			SeqForceIntegerMv:              2,
			EnableCdef:                     true,
			ColorConfig: SequenceHeader_ColorConfig{
				BitDepth:                8,
				ColorPrimaries:          2,
				TransferCharacteristics: 2,
				MatrixCoefficients:      2,
				SubsamplingX:            true,
				SubsamplingY:            true,
			},
		},
		1280,
		720,
	},
}

func TestSequenceHeaderUnmarshal(t *testing.T) {
	for _, ca := range casesSequenceHeader {
		t.Run(ca.name, func(t *testing.T) {
			var sh SequenceHeader
			err := sh.Unmarshal(ca.byts)
			require.NoError(t, err)
			require.Equal(t, ca.sh, sh)
			require.Equal(t, ca.width, sh.Width())
			require.Equal(t, ca.height, sh.Height())
		})
	}
}

func TestSequenceHeaderUnmarshalReuse(t *testing.T) {
	var sh SequenceHeader

	// parse a header with timing info and frame ID numbers
	err := sh.Unmarshal(casesSequenceHeader[3].byts)
	require.NoError(t, err)
	require.Equal(t, casesSequenceHeader[3].sh, sh)

	// parse a minimal header on the same object:
	// no state from the previous parse must leak into the result
	err = sh.Unmarshal(casesSequenceHeader[0].byts)
	require.NoError(t, err)
	require.Equal(t, casesSequenceHeader[0].sh, sh)

	// a failed parse must not poison subsequent parses
	err = sh.Unmarshal(casesSequenceHeader[4].byts[:5])
	require.Error(t, err)

	err = sh.Unmarshal(casesSequenceHeader[4].byts)
	require.NoError(t, err)
	require.Equal(t, casesSequenceHeader[4].sh, sh)
}

func TestSequenceHeaderUnmarshalTruncated(t *testing.T) {
	for _, ca := range casesSequenceHeader {
		t.Run(ca.name, func(t *testing.T) {
			for i := 0; i < len(ca.byts); i++ {
				var sh SequenceHeader
				err := sh.Unmarshal(ca.byts[:i])
				if err == nil {
					// a prefix can only parse successfully when the removed
					// bytes are unused padding; the result must be identical
					// and must not contain default values in place of
					// missing fields.
					require.Equal(t, ca.sh, sh, "prefix of %d bytes", i)
				}
			}
		})
	}
}

func TestSequenceHeaderUnmarshalDecoderModelInfo(t *testing.T) {
	byts := buildSequenceHeader(0, func(buf []byte, pos *int) {
		bits.WriteBitsUnsafe(buf, pos, 0, 3)   // seq_profile
		bits.WriteFlagUnsafe(buf, pos, false)  // still_picture
		bits.WriteFlagUnsafe(buf, pos, false)  // reduced_still_picture_header
		bits.WriteFlagUnsafe(buf, pos, true)   // timing_info_present_flag
		bits.WriteBitsUnsafe(buf, pos, 1, 32)  // num_units_in_display_tick
		bits.WriteBitsUnsafe(buf, pos, 30, 32) // time_scale
		bits.WriteFlagUnsafe(buf, pos, false)  // equal_picture_interval
		bits.WriteFlagUnsafe(buf, pos, true)   // decoder_model_info_present_flag
	})

	var sh SequenceHeader
	err := sh.Unmarshal(byts)
	require.EqualError(t, err, "decoder_model_info_present_flag is not supported yet")
}

func TestSequenceHeaderUnmarshalLarge(t *testing.T) {
	byts := buildSequenceHeader(5999, func(buf []byte, pos *int) {
		bits.WriteBitsUnsafe(buf, pos, 0, 3)  // seq_profile
		bits.WriteFlagUnsafe(buf, pos, false) // still_picture
		bits.WriteFlagUnsafe(buf, pos, false) // reduced_still_picture_header
		bits.WriteFlagUnsafe(buf, pos, false) // timing_info_present_flag
		bits.WriteFlagUnsafe(buf, pos, true)  // initial_display_delay_present_flag
		bits.WriteBitsUnsafe(buf, pos, 31, 5) // operating_points_cnt_minus_1
		for i := 0; i < 32; i++ {
			bits.WriteBitsUnsafe(buf, pos, uint64(i), 12)   // operating_point_idc[i]
			bits.WriteBitsUnsafe(buf, pos, 5, 5)            // seq_level_idx[i]
			bits.WriteFlagUnsafe(buf, pos, true)            // initial_display_delay_present_for_this_op[i]
			bits.WriteBitsUnsafe(buf, pos, uint64(i%16), 4) // initial_display_delay_minus_1[i]
		}
		bits.WriteBitsUnsafe(buf, pos, 10, 4)    // frame_width_bits_minus_1
		bits.WriteBitsUnsafe(buf, pos, 9, 4)     // frame_height_bits_minus_1
		bits.WriteBitsUnsafe(buf, pos, 1279, 11) // max_frame_width_minus_1
		bits.WriteBitsUnsafe(buf, pos, 719, 10)  // max_frame_height_minus_1
		bits.WriteFlagUnsafe(buf, pos, false)    // frame_id_numbers_present_flag
		bits.WriteFlagUnsafe(buf, pos, false)    // use_128x128_superblock
		bits.WriteFlagUnsafe(buf, pos, false)    // enable_filter_intra
		bits.WriteFlagUnsafe(buf, pos, false)    // enable_intra_edge_filter
		bits.WriteFlagUnsafe(buf, pos, false)    // enable_interintra_compound
		bits.WriteFlagUnsafe(buf, pos, false)    // enable_masked_compound
		bits.WriteFlagUnsafe(buf, pos, false)    // enable_warped_motion
		bits.WriteFlagUnsafe(buf, pos, false)    // enable_dual_filter
		bits.WriteFlagUnsafe(buf, pos, false)    // enable_order_hint
		bits.WriteFlagUnsafe(buf, pos, true)     // seq_choose_screen_content_tools
		bits.WriteFlagUnsafe(buf, pos, true)     // seq_choose_integer_mv
		bits.WriteFlagUnsafe(buf, pos, false)    // enable_superres
		bits.WriteFlagUnsafe(buf, pos, true)     // enable_cdef
		bits.WriteFlagUnsafe(buf, pos, false)    // enable_restoration
		bits.WriteFlagUnsafe(buf, pos, false)    // high_bitdepth
		bits.WriteFlagUnsafe(buf, pos, false)    // mono_chrome
		bits.WriteFlagUnsafe(buf, pos, false)    // color_description_present_flag
		bits.WriteFlagUnsafe(buf, pos, false)    // color_range
		bits.WriteBitsUnsafe(buf, pos, 0, 2)     // chroma_sample_position
		bits.WriteFlagUnsafe(buf, pos, false)    // film_grain_params_present
	})
	require.Equal(t, 6000, len(byts))

	var sh SequenceHeader
	err := sh.Unmarshal(byts)
	require.NoError(t, err)

	require.Equal(t, uint8(31), sh.OperatingPointsCntMinus1)
	require.Len(t, sh.OperatingPointIdc, 32)
	require.Len(t, sh.SeqLevelIdx, 32)
	require.Len(t, sh.SeqTier, 32)
	require.Len(t, sh.DecoderModelPresentForThisOp, 32)
	require.Len(t, sh.InitialDisplayPresentForThisOp, 32)
	require.Len(t, sh.InitialDisplayDelayMinus1, 32)

	for i := 0; i < 32; i++ {
		require.Equal(t, uint16(i), sh.OperatingPointIdc[i])
		require.Equal(t, uint8(i%16), sh.InitialDisplayDelayMinus1[i])
	}

	require.Equal(t, 1280, sh.Width())
	require.Equal(t, 720, sh.Height())
}

func FuzzSequenceHeaderUnmarshal(f *testing.F) {
	for _, ca := range casesSequenceHeader {
		f.Add(ca.byts)
	}

	f.Fuzz(func(_ *testing.T, b []byte) {
		var sh SequenceHeader
		err := sh.Unmarshal(b)
		if err != nil {
			return
		}

		sh.Width()
		sh.Height()
	})
}
