package av1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
			10, 19, 4, 0, 0, 0, 4, 0, 0, 0, 120, 128, 0, 16, 181, 76, 255, 179, 192, 26, 1,
		},
		SequenceHeader{
			TimingInfo: &SequenceHeader_TimingInfo{
				NumUnitsInDisplayTick: 1,
				TimeScale:             30,
			},
			InitialDisplayDelayPresentFlag: true,
			OperatingPointIdc:              []uint16{0},
			SeqLevelIdx:                    []uint8{8},
			SeqTier:                        []bool{false},
			DecoderModelPresentForThisOp:   []bool{false},
			InitialDisplayPresentForThisOp: []bool{true},
			InitialDisplayDelayMinus1:      []uint8{6}, // delay = 6 + 1 = 7
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
	{
		"multiple operating points with initial display delay",
		[]byte{
			10, 14, 2, 16, 0, 66, 192, 0, 218, 166, 127, 217, 224, 13, 0, 128,
		},
		SequenceHeader{
			InitialDisplayDelayPresentFlag: true,
			OperatingPointsCntMinus1:       1,
			OperatingPointIdc:              []uint16{0, 0},
			SeqLevelIdx:                    []uint8{8, 13},
			SeqTier:                        []bool{false, true},
			DecoderModelPresentForThisOp:   []bool{false, false},
			InitialDisplayPresentForThisOp: []bool{true, false},
			InitialDisplayDelayMinus1:      []uint8{6, 0},
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

func TestSequenceHeaderUnmarshalTruncated(t *testing.T) {
	for _, ca := range casesSequenceHeader {
		t.Run(ca.name, func(t *testing.T) {
			// find the shortest prefix that still contains all fields
			minLen := -1
			for i := 0; i <= len(ca.byts); i++ {
				var sh SequenceHeader
				if sh.Unmarshal(ca.byts[:i]) == nil {
					minLen = i
					break
				}
			}
			require.Greater(t, minLen, 2)

			// every shorter prefix misses a field and must return an error
			for i := 0; i < minLen; i++ {
				var sh SequenceHeader
				err := sh.Unmarshal(ca.byts[:i])
				require.Error(t, err, "truncated to %d bytes", i)
			}
		})
	}
}

func TestSequenceHeaderUnmarshalReuse(t *testing.T) {
	var sh SequenceHeader

	// parse a header with initial display delay
	err := sh.Unmarshal(casesSequenceHeader[4].byts)
	require.NoError(t, err)
	require.Equal(t, []uint8{6}, sh.InitialDisplayDelayMinus1)

	// a failed parse must not leave stale state that leaks into the next one
	err = sh.Unmarshal(casesSequenceHeader[4].byts[:10])
	require.Error(t, err)

	// parsing a header without the flag must reset all delay-related fields
	err = sh.Unmarshal(casesSequenceHeader[0].byts)
	require.NoError(t, err)
	require.Equal(t, casesSequenceHeader[0].sh, sh)
	require.False(t, sh.InitialDisplayDelayPresentFlag)
	require.Equal(t, []uint8{0}, sh.InitialDisplayDelayMinus1)

	// parsing again must give the exact same result (no state accumulation)
	var sh2 SequenceHeader
	err = sh2.Unmarshal(casesSequenceHeader[0].byts)
	require.NoError(t, err)
	require.Equal(t, sh2, sh)
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
