package h264_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
)

var casesSPS = []struct {
	name   string
	byts   []byte
	sps    h264.SPS
	width  int
	height int
	fps    float64
}{
	{
		"352x288",
		[]byte{
			0x67, 0x64, 0x00, 0x0c, 0xac, 0x3b, 0x50, 0xb0,
			0x4b, 0x42, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00,
			0x00, 0x03, 0x00, 0x3d, 0x08,
		},
		h264.SPS{
			ProfileIdc:                     100,
			LevelIdc:                       12,
			ChromaFormatIdc:                1,
			Log2MaxFrameNumMinus4:          6,
			PicOrderCntType:                2,
			MaxNumRefFrames:                1,
			GapsInFrameNumValueAllowedFlag: true,
			PicWidthInMbsMinus1:            21,
			PicHeightInMapUnitsMinus1:      17,
			FrameMbsOnlyFlag:               true,
			Direct8x8InferenceFlag:         true,
			VUI: &h264.SPS_VUI{
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick:     1,
					TimeScale:          30,
					FixedFrameRateFlag: true,
				},
			},
		},
		352,
		288,
		15,
	},
	{
		"1280x720",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0xac, 0xd9, 0x40, 0x50,
			0x05, 0xbb, 0x01, 0x6c, 0x80, 0x00, 0x00, 0x03,
			0x00, 0x80, 0x00, 0x00, 0x1e, 0x07, 0x8c, 0x18,
			0xcb,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    31,
			ChromaFormatIdc:             1,
			Log2MaxPicOrderCntLsbMinus4: 2,
			MaxNumRefFrames:             4,
			PicWidthInMbsMinus1:         79,
			PicHeightInMapUnitsMinus1:   44,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
			VUI: &h264.SPS_VUI{
				AspectRatioInfoPresentFlag: true,
				AspectRatioIdc:             1,
				VideoSignalTypePresentFlag: true,
				VideoFormat:                5,
				VideoFullRangeFlag:         true,
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick: 1,
					TimeScale:      60,
				},
				BitstreamRestriction: &h264.SPS_BitstreamRestriction{
					MotionVectorsOverPicBoundariesFlag: true,
					Log2MaxMvLengthHorizontal:          11,
					Log2MaxMvLengthVertical:            11,
					MaxNumReorderFrames:                2,
					MaxDecFrameBuffering:               4,
				},
			},
		},
		1280,
		720,
		30,
	},
	{
		"1920x1080 baseline",
		[]byte{
			0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02,
			0x27, 0xe5, 0x84, 0x00, 0x00, 0x03, 0x00, 0x04,
			0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9, 0x20,
		},
		h264.SPS{
			ProfileIdc:                66,
			ChromaFormatIdc:           1,
			ConstraintSet0Flag:        true,
			ConstraintSet1Flag:        true,
			LevelIdc:                  40,
			PicOrderCntType:           2,
			MaxNumRefFrames:           3,
			PicWidthInMbsMinus1:       119,
			PicHeightInMapUnitsMinus1: 67,
			FrameMbsOnlyFlag:          true,
			Direct8x8InferenceFlag:    true,
			FrameCropping: &h264.SPS_FrameCropping{
				BottomOffset: 4,
			},
			VUI: &h264.SPS_VUI{
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick: 1,
					TimeScale:      60,
				},
				BitstreamRestriction: &h264.SPS_BitstreamRestriction{
					MotionVectorsOverPicBoundariesFlag: true,
					Log2MaxMvLengthHorizontal:          11,
					Log2MaxMvLengthVertical:            11,
					MaxDecFrameBuffering:               3,
				},
			},
		},
		1920,
		1080,
		30,
	},
	{
		"1920x1080 nvidia",
		[]byte{
			0x67, 0x64, 0x00, 0x28, 0xac, 0xd9, 0x40, 0x78,
			0x02, 0x27, 0xe5, 0x84, 0x00, 0x00, 0x03, 0x00,
			0x04, 0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60,
			0xc6, 0x58,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    40,
			ChromaFormatIdc:             1,
			Log2MaxPicOrderCntLsbMinus4: 2,
			MaxNumRefFrames:             4,
			PicWidthInMbsMinus1:         119,
			PicHeightInMapUnitsMinus1:   67,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				BottomOffset: 4,
			},
			VUI: &h264.SPS_VUI{
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick: 1,
					TimeScale:      60,
				},
				BitstreamRestriction: &h264.SPS_BitstreamRestriction{
					MotionVectorsOverPicBoundariesFlag: true,
					Log2MaxMvLengthHorizontal:          11,
					Log2MaxMvLengthVertical:            11,
					MaxNumReorderFrames:                2,
					MaxDecFrameBuffering:               4,
				},
			},
		},
		1920,
		1080,
		30,
	},
	{
		"1920x1080",
		[]byte{
			0x67, 0x64, 0x00, 0x29, 0xac, 0x13, 0x31, 0x40,
			0x78, 0x04, 0x47, 0xde, 0x03, 0xea, 0x02, 0x02,
			0x03, 0xe0, 0x00, 0x00, 0x03, 0x00, 0x20, 0x00,
			0x00, 0x06, 0x52, // 0x80,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    41,
			ChromaFormatIdc:             1,
			Log2MaxFrameNumMinus4:       8,
			Log2MaxPicOrderCntLsbMinus4: 5,
			MaxNumRefFrames:             4,
			PicWidthInMbsMinus1:         119,
			PicHeightInMapUnitsMinus1:   33,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				BottomOffset: 2,
			},
			VUI: &h264.SPS_VUI{
				AspectRatioInfoPresentFlag:   true,
				AspectRatioIdc:               1,
				OverscanInfoPresentFlag:      true,
				OverscanAppropriateFlag:      true,
				VideoSignalTypePresentFlag:   true,
				VideoFormat:                  5,
				ColourDescriptionPresentFlag: true,
				ColourPrimaries:              1,
				TransferCharacteristics:      1,
				MatrixCoefficients:           1,
				ChromaLocInfoPresentFlag:     true,
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick:     1,
					TimeScale:          50,
					FixedFrameRateFlag: true,
				},
				PicStructPresentFlag: true,
			},
		},
		1920,
		1080,
		25,
	},
	{
		"hikvision",
		[]byte{103, 100, 0, 32, 172, 23, 42, 1, 64, 30, 104, 64, 0, 1, 194, 0, 0, 87, 228, 33},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    32,
			ChromaFormatIdc:             1,
			Log2MaxPicOrderCntLsbMinus4: 4,
			MaxNumRefFrames:             1,
			PicWidthInMbsMinus1:         79,
			PicHeightInMapUnitsMinus1:   59,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
			Log2MaxFrameNumMinus4:       10,
			VUI: &h264.SPS_VUI{
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick:     1800,
					TimeScale:          90000,
					FixedFrameRateFlag: true,
				},
			},
		},
		1280,
		960,
		25,
	},
	{
		"scaling matrix",
		[]byte{
			103, 100, 0, 50, 173, 132, 1, 12, 32, 8, 97, 0, 67, 8, 2,
			24, 64, 16, 194, 0, 132, 59, 80, 20, 0, 90, 211,
			112, 16, 16, 20, 0, 0, 3, 0, 4, 0, 0, 3, 0, 162, 16,
		},
		h264.SPS{
			ProfileIdc:      100,
			LevelIdc:        50,
			ChromaFormatIdc: 1,
			ScalingList4x4: [][]int32{
				{
					16, 16, 16, 16, 16, 16, 16, 16,
					16, 16, 16, 16, 16, 16, 16, 16,
				},
				{
					16, 16, 16, 16, 16, 16, 16, 16,
					16, 16, 16, 16, 16, 16, 16, 16,
				},
				{
					16, 16, 16, 16, 16, 16, 16, 16,
					16, 16, 16, 16, 16, 16, 16, 16,
				},
				{
					16, 16, 16, 16, 16, 16, 16, 16,
					16, 16, 16, 16, 16, 16, 16, 16,
				},
				{
					16, 16, 16, 16, 16, 16, 16, 16,
					16, 16, 16, 16, 16, 16, 16, 16,
				},
				{
					16, 16, 16, 16, 16, 16, 16, 16,
					16, 16, 16, 16, 16, 16, 16, 16,
				},
			},
			UseDefaultScalingMatrix4x4Flag: []bool{
				false, false, false, false, false, false,
			},
			Log2MaxFrameNumMinus4:          6,
			PicOrderCntType:                2,
			MaxNumRefFrames:                1,
			GapsInFrameNumValueAllowedFlag: true,
			PicWidthInMbsMinus1:            159,
			PicHeightInMapUnitsMinus1:      89,
			FrameMbsOnlyFlag:               true,
			Direct8x8InferenceFlag:         true,
			VUI: &h264.SPS_VUI{
				VideoSignalTypePresentFlag:   true,
				VideoFormat:                  5,
				VideoFullRangeFlag:           true,
				ColourDescriptionPresentFlag: true,
				ColourPrimaries:              1,
				TransferCharacteristics:      1,
				MatrixCoefficients:           1,
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick:     1,
					TimeScale:          40,
					FixedFrameRateFlag: true,
				},
			},
		},
		2560,
		1440,
		20,
	},
	{
		"1920x1080 nvenc hrd",
		[]byte{
			103, 100, 0, 42, 172, 44, 172, 7,
			128, 34, 126, 92, 5, 168, 8, 8,
			10, 0, 0, 7, 208, 0, 3, 169,
			129, 192, 0, 0, 76, 75, 0, 0,
			38, 37, 173, 222, 92, 20,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    42,
			ChromaFormatIdc:             1,
			Log2MaxFrameNumMinus4:       4,
			Log2MaxPicOrderCntLsbMinus4: 4,
			MaxNumRefFrames:             2,
			PicWidthInMbsMinus1:         119,
			PicHeightInMapUnitsMinus1:   67,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				BottomOffset: 4,
			},
			VUI: &h264.SPS_VUI{
				AspectRatioInfoPresentFlag:   true,
				AspectRatioIdc:               1,
				VideoSignalTypePresentFlag:   true,
				VideoFormat:                  5,
				ColourDescriptionPresentFlag: true,
				ColourPrimaries:              1,
				TransferCharacteristics:      1,
				MatrixCoefficients:           1,
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick:     1000,
					TimeScale:          120000,
					FixedFrameRateFlag: true,
				},
				NalHRD: &h264.SPS_HRD{
					BitRateValueMinus1:                 []uint32{39061},
					CpbSizeValueMinus1:                 []uint32{156249},
					CbrFlag:                            []bool{true},
					InitialCpbRemovalDelayLengthMinus1: 23,
					CpbRemovalDelayLengthMinus1:        15,
					DpbOutputDelayLengthMinus1:         5,
					TimeOffsetLength:                   24,
				},
				PicStructPresentFlag: true,
			},
		},
		1920,
		1080,
		60,
	},
	{
		"1920x1080 hikvision nal hrd + vcl hrd",
		[]byte{
			103, 77, 0, 41, 154, 100, 3, 192,
			17, 63, 46, 2, 220, 4, 4, 5,
			0, 0, 3, 3, 232, 0, 0, 195,
			80, 232, 96, 0, 186, 180, 0, 2,
			234, 196, 187, 203, 141, 12, 0, 23,
			86, 128, 0, 93, 88, 151, 121, 112,
			160,
		},
		h264.SPS{
			ProfileIdc:                  77,
			ChromaFormatIdc:             1,
			LevelIdc:                    41,
			Log2MaxFrameNumMinus4:       5,
			Log2MaxPicOrderCntLsbMinus4: 5,
			MaxNumRefFrames:             1,
			PicWidthInMbsMinus1:         119,
			PicHeightInMapUnitsMinus1:   67,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				BottomOffset: 4,
			},
			VUI: &h264.SPS_VUI{
				AspectRatioInfoPresentFlag:   true,
				AspectRatioIdc:               1,
				VideoSignalTypePresentFlag:   true,
				VideoFormat:                  5,
				VideoFullRangeFlag:           true,
				ColourDescriptionPresentFlag: true,
				ColourPrimaries:              1,
				TransferCharacteristics:      1,
				MatrixCoefficients:           1,
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick:     1000,
					TimeScale:          50000,
					FixedFrameRateFlag: true,
				},
				NalHRD: &h264.SPS_HRD{
					BitRateScale:                       4,
					CpbSizeScale:                       3,
					BitRateValueMinus1:                 []uint32{11948},
					CpbSizeValueMinus1:                 []uint32{95585},
					CbrFlag:                            []bool{false},
					InitialCpbRemovalDelayLengthMinus1: 23,
					CpbRemovalDelayLengthMinus1:        15,
					DpbOutputDelayLengthMinus1:         5,
					TimeOffsetLength:                   24,
				},
				VclHRD: &h264.SPS_HRD{
					BitRateScale:                       4,
					CpbSizeScale:                       3,
					BitRateValueMinus1:                 []uint32{11948},
					CpbSizeValueMinus1:                 []uint32{95585},
					CbrFlag:                            []bool{false},
					InitialCpbRemovalDelayLengthMinus1: 23,
					CpbRemovalDelayLengthMinus1:        15,
					DpbOutputDelayLengthMinus1:         5,
					TimeOffsetLength:                   24,
				},
				PicStructPresentFlag: true,
			},
		},
		1920,
		1080,
		25,
	},
	{
		"1920x1080 mbs_only_flag = 0",
		[]byte{
			0x67, 0x4d, 0x40, 0x28, 0xab, 0x60, 0x3c, 0x02,
			0x23, 0xef, 0x01, 0x10, 0x00, 0x00, 0x03, 0x00,
			0x10, 0x00, 0x00, 0x03, 0x03, 0x2e, 0x94, 0x00,
			0x35, 0x64, 0x06, 0xb2, 0x85, 0x08, 0x0e, 0xe2,
			0xc5, 0x22, 0xc0,
		},
		h264.SPS{
			ProfileIdc:                  77,
			ChromaFormatIdc:             1,
			ConstraintSet1Flag:          true,
			LevelIdc:                    40,
			Log2MaxFrameNumMinus4:       1,
			Log2MaxPicOrderCntLsbMinus4: 2,
			MaxNumRefFrames:             2,
			PicWidthInMbsMinus1:         119,
			PicHeightInMapUnitsMinus1:   33,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				BottomOffset: 2,
			},
			VUI: &h264.SPS_VUI{
				AspectRatioInfoPresentFlag: true,
				AspectRatioIdc:             1,
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick:     1,
					TimeScale:          50,
					FixedFrameRateFlag: true,
				},
				NalHRD: &h264.SPS_HRD{
					BitRateScale:                       4,
					CpbSizeScale:                       10,
					BitRateValueMinus1:                 []uint32{3416},
					CpbSizeValueMinus1:                 []uint32{213},
					CbrFlag:                            []bool{false},
					InitialCpbRemovalDelayLengthMinus1: 20,
					CpbRemovalDelayLengthMinus1:        5,
					DpbOutputDelayLengthMinus1:         1,
				},
				PicStructPresentFlag: true,
				BitstreamRestriction: &h264.SPS_BitstreamRestriction{
					MotionVectorsOverPicBoundariesFlag: true,
					MaxBytesPerPicDenom:                2,
					Log2MaxMvLengthHorizontal:          10,
					Log2MaxMvLengthVertical:            9,
					MaxNumReorderFrames:                1,
					MaxDecFrameBuffering:               4,
				},
			},
		},
		1920,
		1080,
		25,
	},
	{
		// monochrome (chroma_format_idc = 0) with horizontal and vertical
		// cropping: crop offsets are in luma samples, so the right edge must
		// actually be removed instead of being left as a black bar.
		"1366x766 monochrome",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0xf2, 0xd0, 0x0a, 0xc0,
			0xc3, 0xc5, 0xdc, 0x20, 0x00, 0x00, 0xe1, 0x00,
			0x00, 0x2b, 0xf2, 0x10, 0x80,
		},
		h264.SPS{
			ProfileIdc:                100,
			LevelIdc:                  31,
			ChromaFormatIdc:           0,
			Log2MaxFrameNumMinus4:     0,
			PicOrderCntType:           2,
			MaxNumRefFrames:           1,
			PicWidthInMbsMinus1:       85,
			PicHeightInMapUnitsMinus1: 47,
			FrameMbsOnlyFlag:          true,
			Direct8x8InferenceFlag:    true,
			FrameCropping: &h264.SPS_FrameCropping{
				RightOffset:  10,
				BottomOffset: 2,
			},
			VUI: &h264.SPS_VUI{
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick:     1800,
					TimeScale:          90000,
					FixedFrameRateFlag: true,
				},
			},
		},
		1366,
		766,
		25,
	},
	{
		// monochrome interlaced stream: frame_mbs_only_flag = 0 makes the
		// vertical crop unit 2 luma samples.
		"172x284 monochrome interlaced",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0xf2, 0xd0, 0x58, 0x93,
			0x95, 0x24,
		},
		h264.SPS{
			ProfileIdc:                100,
			LevelIdc:                  31,
			ChromaFormatIdc:           0,
			Log2MaxFrameNumMinus4:     0,
			PicOrderCntType:           2,
			MaxNumRefFrames:           1,
			PicWidthInMbsMinus1:       10,
			PicHeightInMapUnitsMinus1: 8,
			FrameMbsOnlyFlag:          false,
			Direct8x8InferenceFlag:    true,
			FrameCropping: &h264.SPS_FrameCropping{
				RightOffset:  4,
				TopOffset:    1,
				BottomOffset: 1,
			},
		},
		172,
		284,
		0,
	},
	{
		// 4:2:2 (chroma_format_idc = 2): horizontal crop unit is 2,
		// vertical crop unit is 1.
		"172x143 4:2:2",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0xbc, 0xb4, 0x16, 0x27,
			0xdd, 0x20,
		},
		h264.SPS{
			ProfileIdc:                100,
			LevelIdc:                  31,
			ChromaFormatIdc:           2,
			Log2MaxFrameNumMinus4:     0,
			PicOrderCntType:           2,
			MaxNumRefFrames:           1,
			PicWidthInMbsMinus1:       10,
			PicHeightInMapUnitsMinus1: 8,
			FrameMbsOnlyFlag:          true,
			Direct8x8InferenceFlag:    true,
			FrameCropping: &h264.SPS_FrameCropping{
				RightOffset:  2,
				BottomOffset: 1,
			},
		},
		172,
		143,
		0,
	},
	{
		// 4:4:4 (chroma_format_idc = 3): crop units are 1 in both directions.
		"174x143 4:4:4",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0x91, 0x96, 0x82, 0xc4,
			0xfb, 0xa4,
		},
		h264.SPS{
			ProfileIdc:                100,
			LevelIdc:                  31,
			ChromaFormatIdc:           3,
			Log2MaxFrameNumMinus4:     0,
			PicOrderCntType:           2,
			MaxNumRefFrames:           1,
			PicWidthInMbsMinus1:       10,
			PicHeightInMapUnitsMinus1: 8,
			FrameMbsOnlyFlag:          true,
			Direct8x8InferenceFlag:    true,
			FrameCropping: &h264.SPS_FrameCropping{
				RightOffset:  2,
				BottomOffset: 1,
			},
		},
		174,
		143,
		0,
	},
	{
		// 4:4:4 with separate_colour_plane_flag: the array type becomes
		// monochrome and crop offsets are again in luma samples.
		"174x143 4:4:4 separate colour planes",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0x93, 0x96, 0x82, 0xc4,
			0xfb, 0xa4,
		},
		h264.SPS{
			ProfileIdc:                100,
			LevelIdc:                  31,
			ChromaFormatIdc:           3,
			SeparateColourPlaneFlag:   true,
			Log2MaxFrameNumMinus4:     0,
			PicOrderCntType:           2,
			MaxNumRefFrames:           1,
			PicWidthInMbsMinus1:       10,
			PicHeightInMapUnitsMinus1: 8,
			FrameMbsOnlyFlag:          true,
			Direct8x8InferenceFlag:    true,
			FrameCropping: &h264.SPS_FrameCropping{
				RightOffset:  2,
				BottomOffset: 1,
			},
		},
		174,
		143,
		0,
	},
	{
		// monochrome without cropping and without VUI: raw macroblock size.
		"352x288 monochrome no VUI",
		[]byte{
			0x67, 0x64, 0x00, 0x1e, 0xf2, 0xd0, 0x2c, 0x12,
			0xc8,
		},
		h264.SPS{
			ProfileIdc:                100,
			LevelIdc:                  30,
			ChromaFormatIdc:           0,
			Log2MaxFrameNumMinus4:     0,
			PicOrderCntType:           2,
			MaxNumRefFrames:           1,
			PicWidthInMbsMinus1:       21,
			PicHeightInMapUnitsMinus1: 17,
			FrameMbsOnlyFlag:          true,
			Direct8x8InferenceFlag:    true,
		},
		352,
		288,
		0,
	},
}

func TestSPSUnmarshal(t *testing.T) {
	for _, ca := range casesSPS {
		t.Run(ca.name, func(t *testing.T) {
			var sps h264.SPS
			err := sps.Unmarshal(ca.byts)
			require.NoError(t, err)
			require.Equal(t, ca.sps, sps)
			require.Equal(t, ca.width, sps.Width())
			require.Equal(t, ca.height, sps.Height())
			require.Equal(t, ca.fps, sps.FPS())
		})
	}
}

func TestSPSUnmarshalErrors(t *testing.T) {
	for _, ca := range []struct {
		name string
		byts []byte
		err  string
	}{
		{
			"empty buffer",
			nil,
			"not enough bits",
		},
		{
			"not a SPS",
			[]byte{0x65, 0x64, 0x00, 0x1f, 0xf2, 0xd0, 0x0a},
			"not a SPS",
		},
		{
			"truncated RBSP",
			[]byte{
				0x67, 0x64, 0x00, 0x1f, 0xf2, 0xd0, 0x0a, 0xc0,
				0xc3, 0xc5,
			},
			"not enough bits",
		},
		{
			"golomb value with too many leading zero bits",
			[]byte{0x67, 0x64, 0x00, 0x1f, 0x00, 0x00, 0x00, 0x00, 0x01},
			"invalid value",
		},
		{
			"invalid pic_order_cnt_type",
			[]byte{0x67, 0x63, 0x00, 0x1e, 0xc9},
			"invalid pic_order_cnt_type: 3",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var sps h264.SPS
			err := sps.Unmarshal(ca.byts)
			require.EqualError(t, err, ca.err)
		})
	}
}

func TestSPSQueryStabilityAndIsolation(t *testing.T) {
	// queries must be deterministic, work on copies and never re-parse.
	var mono, color *struct {
		name   string
		byts   []byte
		sps    h264.SPS
		width  int
		height int
		fps    float64
	}
	for i := range casesSPS {
		switch casesSPS[i].name {
		case "1366x766 monochrome":
			mono = &casesSPS[i]
		case "1280x720":
			color = &casesSPS[i]
		}
	}
	require.NotNil(t, mono)
	require.NotNil(t, color)

	var sps h264.SPS
	require.NoError(t, sps.Unmarshal(mono.byts))
	for i := 0; i < 5; i++ {
		require.Equal(t, mono.width, sps.Width())
		require.Equal(t, mono.height, sps.Height())
		require.Equal(t, mono.fps, sps.FPS())
	}

	// a value copy must return the same results.
	copySPS := sps
	require.Equal(t, mono.width, copySPS.Width())
	require.Equal(t, mono.height, copySPS.Height())
	require.Equal(t, mono.fps, copySPS.FPS())

	// parsing another SPS into a different struct must not leak state.
	var other h264.SPS
	require.NoError(t, other.Unmarshal(color.byts))
	require.Equal(t, color.width, other.Width())
	require.Equal(t, color.height, other.Height())
	require.Equal(t, color.fps, other.FPS())
	require.Equal(t, mono.width, sps.Width())
	require.Equal(t, mono.height, sps.Height())
	require.Equal(t, mono.fps, sps.FPS())
}

func TestSPSManyLinear(t *testing.T) {
	// Width/Height/FPS only read parsed fields: 6000 SPS must stay linear and
	// repeated queries must return stable results without re-parsing.
	const count = 6000
	for i := 0; i < count; i++ {
		var s h264.SPS
		require.NoError(t, s.Unmarshal(casesSPS[i%len(casesSPS)].byts))
		for j := 0; j < 3; j++ {
			_ = s.Width()
			_ = s.Height()
			_ = s.FPS()
		}
		require.Equal(t, casesSPS[i%len(casesSPS)].width, s.Width())
		require.Equal(t, casesSPS[i%len(casesSPS)].height, s.Height())
		require.Equal(t, casesSPS[i%len(casesSPS)].fps, s.FPS())
	}
}

func TestSPSOversizedCroppingNotSilentlyFixed(t *testing.T) {
	// crop offsets exceeding the picture size are parsed as-is; Width/Height
	// must not be clamped to zero, the arithmetic result is preserved.
	byts := []byte{
		0x67, 0x64, 0x00, 0x1e, 0xf2, 0xd3, 0xe0, 0x0f,
		0xa4, 0x01, 0xf4, 0xe8,
	}
	var sps h264.SPS
	require.NoError(t, sps.Unmarshal(byts))
	require.Equal(t, &h264.SPS_FrameCropping{
		LeftOffset:  1000,
		RightOffset: 1000,
	}, sps.FrameCropping)
	// 16 - 2*1000 wraps in uint32 (no clamping) before the cast to int;
	// mirror the production expression so the expectation is arch-independent.
	lumaWidth := uint32(16)
	cropOffset := uint32(1000)
	require.Equal(t, int(lumaWidth-cropOffset-cropOffset), sps.Width())
	require.Equal(t, 16, sps.Height())
}

func BenchmarkSPSUnmarshal(b *testing.B) {
	for b.Loop() {
		var sps h264.SPS
		sps.Unmarshal([]byte{ //nolint:errcheck
			103, 77, 0, 41, 154, 100, 3, 192,
			17, 63, 46, 2, 220, 4, 4, 5,
			0, 0, 3, 3, 232, 0, 0, 195,
			80, 232, 96, 0, 186, 180, 0, 2,
			234, 196, 187, 203, 141, 12, 0, 23,
			86, 128, 0, 93, 88, 151, 121, 112,
			160,
		})
	}
}

func FuzzSPSUnmarshal(f *testing.F) {
	for _, ca := range casesSPS {
		f.Add(ca.byts)
	}

	f.Fuzz(func(_ *testing.T, b []byte) {
		var sps h264.SPS
		err := sps.Unmarshal(b)
		if err != nil {
			return
		}

		sps.Width()
		sps.Height()
		sps.FPS()
	})
}
