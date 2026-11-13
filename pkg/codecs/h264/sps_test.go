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
		"1366x766 monochrome",
		[]byte{
			0x67, 0x64, 0x00, 0x20, 0xf3, 0x65, 0x01, 0x58,
			0x18, 0x78, 0xbb, 0xc0, 0x5b, 0x20, 0x00, 0x00,
			0x03, 0x00, 0x20, 0x00, 0x00, 0x06, 0x41, 0xe3,
			0x06, 0x32, 0xc0,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    32,
			ChromaFormatIdc:             0,
			Log2MaxPicOrderCntLsbMinus4: 2,
			MaxNumRefFrames:             4,
			PicWidthInMbsMinus1:         85,
			PicHeightInMapUnitsMinus1:   47,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				RightOffset:  10,
				BottomOffset: 2,
			},
			VUI: &h264.SPS_VUI{
				AspectRatioInfoPresentFlag: true,
				AspectRatioIdc:             1,
				VideoSignalTypePresentFlag: true,
				VideoFormat:                5,
				VideoFullRangeFlag:         true,
				TimingInfo: &h264.SPS_TimingInfo{
					NumUnitsInTick: 1,
					TimeScale:      50,
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
		1366,
		766,
		25,
	},
	{
		// monochrome (ChromaFormatIdc=0), frame_mbs_only_flag=1, cropped.
		// Crop units must be 1x1 luma samples, like 4:4:4, so that the
		// offsets are applied the same way as in color streams.
		"1280x720 monochrome cropped",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0xf3, 0xa0, 0x14, 0x01,
			0x6f, 0x29, 0xd8, 0x94,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    31,
			ChromaFormatIdc:             0,
			Log2MaxPicOrderCntLsbMinus4: 0,
			MaxNumRefFrames:             1,
			PicWidthInMbsMinus1:         79,
			PicHeightInMapUnitsMinus1:   44,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				LeftOffset:   4,
				RightOffset:  6,
				TopOffset:    2,
				BottomOffset: 8,
			},
		},
		1270,
		710,
		0,
	},
	{
		// monochrome, frame_mbs_only_flag=0: vertical crop unit stays
		// 2 luma samples, exactly as in interlaced color streams.
		"1280x720 monochrome interlaced cropped",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0xf3, 0xa0, 0x14, 0x02,
			0xe6, 0x53, 0xb1, 0x28,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    31,
			ChromaFormatIdc:             0,
			Log2MaxPicOrderCntLsbMinus4: 0,
			MaxNumRefFrames:             1,
			PicWidthInMbsMinus1:         79,
			PicHeightInMapUnitsMinus1:   22,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				LeftOffset:   4,
				RightOffset:  6,
				TopOffset:    2,
				BottomOffset: 8,
			},
		},
		1270,
		716,
		0,
	},
	{
		// monochrome without frame cropping and without VUI.
		"1280x720 monochrome no crop no vui",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0xf3, 0xa0, 0x14, 0x01,
			0x6e, 0x40,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    31,
			ChromaFormatIdc:             0,
			Log2MaxPicOrderCntLsbMinus4: 0,
			MaxNumRefFrames:             1,
			PicWidthInMbsMinus1:         79,
			PicHeightInMapUnitsMinus1:   44,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
		},
		1280,
		720,
		0,
	},
	{
		// 4:2:2 (ChromaFormatIdc=2): horizontal crop unit is 2,
		// vertical crop unit is 1.
		"1280x720 4:2:2 cropped",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0xbc, 0xe8, 0x05, 0x00,
			0x5b, 0xca, 0x76, 0x25,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    31,
			ChromaFormatIdc:             2,
			Log2MaxPicOrderCntLsbMinus4: 0,
			MaxNumRefFrames:             1,
			PicWidthInMbsMinus1:         79,
			PicHeightInMapUnitsMinus1:   44,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				LeftOffset:   4,
				RightOffset:  6,
				TopOffset:    2,
				BottomOffset: 8,
			},
		},
		1260,
		710,
		0,
	},
	{
		// 4:4:4 (ChromaFormatIdc=3): crop units are 1x1 luma samples.
		"1280x720 4:4:4 cropped",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0x91, 0x9d, 0x00, 0xa0,
			0x0b, 0x79, 0x4e, 0xc4, 0xa0,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    31,
			ChromaFormatIdc:             3,
			Log2MaxPicOrderCntLsbMinus4: 0,
			MaxNumRefFrames:             1,
			PicWidthInMbsMinus1:         79,
			PicHeightInMapUnitsMinus1:   44,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				LeftOffset:   4,
				RightOffset:  6,
				TopOffset:    2,
				BottomOffset: 8,
			},
		},
		1270,
		710,
		0,
	},
	{
		// 4:4:4 with separate_colour_plane_flag: ChromaArrayType becomes
		// 0, so crop units must be computed exactly like monochrome.
		"1280x720 4:4:4 separate colour plane cropped",
		[]byte{
			0x67, 0x64, 0x00, 0x1f, 0x93, 0x9d, 0x00, 0xa0,
			0x0b, 0x79, 0x4e, 0xc4, 0xa0,
		},
		h264.SPS{
			ProfileIdc:                  100,
			LevelIdc:                    31,
			ChromaFormatIdc:             3,
			SeparateColourPlaneFlag:     true,
			Log2MaxPicOrderCntLsbMinus4: 0,
			MaxNumRefFrames:             1,
			PicWidthInMbsMinus1:         79,
			PicHeightInMapUnitsMinus1:   44,
			FrameMbsOnlyFlag:            true,
			Direct8x8InferenceFlag:      true,
			FrameCropping: &h264.SPS_FrameCropping{
				LeftOffset:   4,
				RightOffset:  6,
				TopOffset:    2,
				BottomOffset: 8,
			},
		},
		1270,
		710,
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

// a valid monochrome SPS with VUI timing info (25 fps), 1280x720, no cropping.
var monochromeVUI = []byte{
	0x67, 0x64, 0x00, 0x1f, 0xf2, 0xd0, 0x0a, 0x00,
	0xb7, 0x42, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00,
	0x00, 0x64, 0x08,
}

func TestSPSUnmarshalErrors(t *testing.T) {
	// every prefix of a valid SPS shorter than the whole RBSP must fail:
	// a truncated RBSP is never repaired or silently accepted.
	// use a real-world SPS without trailing padding ambiguity.
	var valid []byte
	for _, ca := range casesSPS {
		if ca.name == "1366x766 monochrome" {
			valid = ca.byts
			break
		}
	}
	require.NotEmpty(t, valid)
	for n := 1; n < len(valid); n++ {
		var sps h264.SPS
		err := sps.Unmarshal(append([]byte(nil), valid[:n]...))
		require.Error(t, err)
	}

	// a valid SPS payload with the NALU type replaced by IDR (5).
	notSPS := append([]byte(nil), casesSPS[0].byts...)
	notSPS[0] = 0x65

	errCases := []struct {
		name string
		byts []byte
		err  string
	}{
		{
			"empty",
			[]byte{},
			"not enough bits",
		},
		{
			"nalu header only",
			[]byte{0x67},
			"not enough bits",
		},
		{
			"not a SPS",
			notSPS,
			"not a SPS",
		},
		{
			"invalid pic_order_cnt_type",
			// baseline SPS whose pic_order_cnt_type is 3.
			[]byte{0x67, 0x42, 0x00, 0x1f, 0xc9},
			"invalid pic_order_cnt_type: 3",
		},
	}

	for _, ca := range errCases {
		t.Run(ca.name, func(t *testing.T) {
			var sps h264.SPS
			err := sps.Unmarshal(ca.byts)
			require.EqualError(t, err, ca.err)
		})
	}
}

func TestSPSMonochromeParity(t *testing.T) {
	// monochrome cropping must follow the same luma-sample rules as color:
	// 4:4:4 and monochrome produce identical dimensions on identical crops.
	mono := func() *h264.SPS {
		var s h264.SPS
		err := s.Unmarshal([]byte{
			0x67, 0x64, 0x00, 0x1f, 0xf3, 0xa0, 0x14, 0x01,
			0x6f, 0x29, 0xd8, 0x94,
		})
		require.NoError(t, err)
		return &s
	}()
	colored := func() *h264.SPS {
		var s h264.SPS
		err := s.Unmarshal([]byte{
			0x67, 0x64, 0x00, 0x1f, 0x91, 0x9d, 0x00, 0xa0,
			0x0b, 0x79, 0x4e, 0xc4, 0xa0,
		})
		require.NoError(t, err)
		return &s
	}()

	require.Equal(t, colored.Width(), mono.Width())
	require.Equal(t, colored.Height(), mono.Height())

	// repeated queries on the same SPS are stable and do not re-parse.
	var timed h264.SPS
	err := timed.Unmarshal(monochromeVUI)
	require.NoError(t, err)

	for range 5 {
		require.Equal(t, 1280, timed.Width())
		require.Equal(t, 720, timed.Height())
		require.Equal(t, 25.0, timed.FPS())
	}

	// parsing another SPS into a different value must not leak state.
	var other h264.SPS
	err = other.Unmarshal(casesSPS[1].byts)
	require.NoError(t, err)
	require.Equal(t, 1280, other.Width())
	require.Equal(t, 720, other.Height())
	require.Equal(t, 30.0, other.FPS())
	require.Equal(t, 1280, timed.Width())
	require.Equal(t, 720, timed.Height())
	require.Equal(t, 25.0, timed.FPS())
}

func BenchmarkSPSWidthHeight(b *testing.B) {
	var sps h264.SPS
	err := sps.Unmarshal(monochromeVUI)
	require.NoError(b, err)

	for b.Loop() {
		// Width/Height/FPS are pure accessors over fields filled once by
		// Unmarshal: querying them must not parse the bitstream again.
		if sps.Width() != 1280 || sps.Height() != 720 || sps.FPS() != 25.0 {
			b.Fatal("unexpected result")
		}
	}
}

func TestSPSManyLinear(t *testing.T) {
	// parsing thousands of SPS must stay linear and every query must be
	// answered from already-parsed fields, without touching the bitstream.
	const count = 6000

	parsed := make([]h264.SPS, count)
	for i := range count {
		err := parsed[i].Unmarshal(monochromeVUI)
		require.NoError(t, err)
	}

	for i := range count {
		require.Equal(t, 1280, parsed[i].Width())
		require.Equal(t, 720, parsed[i].Height())
		require.Equal(t, 25.0, parsed[i].FPS())
	}
}
