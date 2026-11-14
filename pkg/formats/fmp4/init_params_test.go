package fmp4

import (
	"bytes"
	"testing"

	amp4 "github.com/abema/go-mp4"
	"github.com/stretchr/testify/require"

	imp4 "github.com/bluenviron/mediacommon/v2/internal/mp4"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/fmp4/seekablebuffer"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mp4/codecs"
)

// buildInit builds a fMP4 initialization block that contains a single
// video track whose codec box is provided by the caller.
func buildInit(t *testing.T, entry amp4.BoxType, codecBox amp4.IImmutableBox) []byte {
	t.Helper()

	var buf seekablebuffer.Buffer
	mw := &imp4.Writer{W: &buf}
	mw.Initialize()

	_, err := mw.WriteBox(&amp4.Ftyp{
		MajorBrand:       [4]byte{'m', 'p', '4', '2'},
		MinorVersion:     1,
		CompatibleBrands: []amp4.CompatibleBrandElem{{CompatibleBrand: [4]byte{'i', 's', 'o', 'm'}}},
	})
	require.NoError(t, err)

	_, err = mw.WriteBoxStart(&amp4.Moov{})
	require.NoError(t, err)

	_, err = mw.WriteBox(&amp4.Mvhd{
		Timescale:   1000,
		Rate:        65536,
		Volume:      256,
		Matrix:      [9]int32{0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000},
		NextTrackID: 2,
	})
	require.NoError(t, err)

	_, err = mw.WriteBoxStart(&amp4.Trak{})
	require.NoError(t, err)

	_, err = mw.WriteBox(&amp4.Tkhd{
		FullBox: amp4.FullBox{Flags: [3]byte{0, 0, 3}},
		TrackID: 1,
		Width:   1280 * 65536,
		Height:  720 * 65536,
		Matrix:  [9]int32{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
	})
	require.NoError(t, err)

	_, err = mw.WriteBoxStart(&amp4.Mdia{})
	require.NoError(t, err)

	_, err = mw.WriteBox(&amp4.Mdhd{
		Timescale: 90000,
		Language:  [3]byte{'u', 'n', 'd'},
	})
	require.NoError(t, err)

	_, err = mw.WriteBoxStart(&amp4.Minf{})
	require.NoError(t, err)

	_, err = mw.WriteBoxStart(&amp4.Stbl{})
	require.NoError(t, err)

	_, err = mw.WriteBoxStart(&amp4.Stsd{EntryCount: 1})
	require.NoError(t, err)

	_, err = mw.WriteBoxStart(&amp4.VisualSampleEntry{
		SampleEntry: amp4.SampleEntry{
			AnyTypeBox:         amp4.AnyTypeBox{Type: entry},
			DataReferenceIndex: 1,
		},
		Width:           1280,
		Height:          720,
		Horizresolution: 4718592,
		Vertresolution:  4718592,
		FrameCount:      1,
		Depth:           24,
		PreDefined3:     -1,
	})
	require.NoError(t, err)

	_, err = mw.WriteBox(codecBox)
	require.NoError(t, err)

	require.NoError(t, mw.WriteBoxEnd()) // </*>
	require.NoError(t, mw.WriteBoxEnd()) // </stsd>
	require.NoError(t, mw.WriteBoxEnd()) // </stbl>
	require.NoError(t, mw.WriteBoxEnd()) // </minf>
	require.NoError(t, mw.WriteBoxEnd()) // </mdia>
	require.NoError(t, mw.WriteBoxEnd()) // </trak>

	_, err = mw.WriteBoxStart(&amp4.Mvex{})
	require.NoError(t, err)

	_, err = mw.WriteBox(&amp4.Trex{
		TrackID:                       1,
		DefaultSampleDescriptionIndex: 1,
	})
	require.NoError(t, err)

	require.NoError(t, mw.WriteBoxEnd()) // </mvex>
	require.NoError(t, mw.WriteBoxEnd()) // </moov>

	return buf.Bytes()
}

func newAvcC(sps []amp4.AVCParameterSet, pps []amp4.AVCParameterSet) *amp4.AVCDecoderConfiguration {
	return &amp4.AVCDecoderConfiguration{
		AnyTypeBox:                 amp4.AnyTypeBox{Type: amp4.BoxTypeAvcC()},
		ConfigurationVersion:       1,
		Profile:                    0x42,
		ProfileCompatibility:       0,
		Level:                      0x1f,
		Reserved:                   0b111111,
		LengthSizeMinusOne:         3,
		Reserved2:                  0b111,
		NumOfSequenceParameterSets: uint8(len(sps)),
		SequenceParameterSets:      sps,
		NumOfPictureParameterSets:  uint8(len(pps)),
		PictureParameterSets:       pps,
	}
}

func newHvcC(numArrays uint8, arrays []amp4.HEVCNaluArray) *amp4.HvcC {
	return &amp4.HvcC{
		ConfigurationVersion: 1,
		Reserved1:            0b1111,
		Reserved2:            0b111111,
		Reserved3:            0b111111,
		Reserved4:            0b11111,
		Reserved5:            0b11111,
		NumTemporalLayers:    1,
		LengthSizeMinusOne:   3,
		NumOfNaluArrays:      numArrays,
		NaluArrays:           arrays,
	}
}

func newHvcNaluArray(naluType byte, nalus ...amp4.HEVCNalu) amp4.HEVCNaluArray {
	return amp4.HEVCNaluArray{
		Completeness: true,
		NaluType:     naluType,
		NumNalus:     uint16(len(nalus)),
		Nalus:        nalus,
	}
}

func newHvcNalu(data []byte) amp4.HEVCNalu {
	return amp4.HEVCNalu{
		Length:  uint16(len(data)),
		NALUnit: data,
	}
}

func TestInitUnmarshalH264Params(t *testing.T) {
	for _, ca := range []struct {
		name   string
		codec  amp4.IImmutableBox
		expect func(t *testing.T, init *Init)
		err    string
	}{
		{
			"valid",
			newAvcC(
				[]amp4.AVCParameterSet{{Length: uint16(len(testSPS)), NALUnit: testSPS}},
				[]amp4.AVCParameterSet{{Length: 1, NALUnit: []byte{0x08}}},
			),
			func(t *testing.T, init *Init) {
				require.Equal(t, &codecs.H264{SPS: testSPS, PPS: []byte{0x08}}, init.Tracks[0].Codec)
			},
			"",
		},
		{
			"empty SPS",
			newAvcC(
				[]amp4.AVCParameterSet{{Length: 0, NALUnit: []byte{}}},
				[]amp4.AVCParameterSet{{Length: 1, NALUnit: []byte{0x08}}},
			),
			nil,
			"H264 parameters not provided",
		},
		{
			"empty PPS",
			newAvcC(
				[]amp4.AVCParameterSet{{Length: uint16(len(testSPS)), NALUnit: testSPS}},
				[]amp4.AVCParameterSet{{Length: 0, NALUnit: []byte{}}},
			),
			nil,
			"H264 parameters not provided",
		},
		{
			"no SPS",
			newAvcC(nil,
				[]amp4.AVCParameterSet{{Length: 1, NALUnit: []byte{0x08}}}),
			nil,
			"H264 parameters not provided",
		},
		{
			"no PPS",
			newAvcC(
				[]amp4.AVCParameterSet{{Length: uint16(len(testSPS)), NALUnit: testSPS}},
				nil),
			nil,
			"H264 parameters not provided",
		},
		{
			"multiple SPS",
			newAvcC(
				[]amp4.AVCParameterSet{
					{Length: uint16(len(testSPS)), NALUnit: testSPS},
					{Length: uint16(len(testSPS)), NALUnit: testSPS},
				},
				[]amp4.AVCParameterSet{{Length: 1, NALUnit: []byte{0x08}}},
			),
			nil,
			"multiple H264 parameters are not supported",
		},
		{
			"multiple PPS",
			newAvcC(
				[]amp4.AVCParameterSet{{Length: uint16(len(testSPS)), NALUnit: testSPS}},
				[]amp4.AVCParameterSet{
					{Length: 1, NALUnit: []byte{0x08}},
					{Length: 1, NALUnit: []byte{0x08}},
				},
			),
			nil,
			"multiple H264 parameters are not supported",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			byts := buildInit(t, amp4.BoxTypeAvc1(), ca.codec)

			var init Init
			err := init.Unmarshal(bytes.NewReader(byts))
			if ca.err != "" {
				require.EqualError(t, err, ca.err)
				return
			}

			require.NoError(t, err)
			ca.expect(t, &init)
		})
	}
}

func TestInitUnmarshalH265Params(t *testing.T) {
	// a syntactically valid SPS is required only when marshaling back,
	// therefore any non-empty bytes are enough here.
	vps := []byte{0x40, 0x01}
	sps := []byte{0x42, 0x01}
	pps := []byte{0x44, 0x01}

	for _, ca := range []struct {
		name  string
		codec amp4.IImmutableBox
		err   string
	}{
		{
			"valid",
			newHvcC(3, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu(vps)),
				newHvcNaluArray(33, newHvcNalu(sps)),
				newHvcNaluArray(34, newHvcNalu(pps)),
			}),
			"",
		},
		{
			"empty VPS",
			newHvcC(3, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu(nil)),
				newHvcNaluArray(33, newHvcNalu(sps)),
				newHvcNaluArray(34, newHvcNalu(pps)),
			}),
			"H265 parameter not provided",
		},
		{
			"empty SPS",
			newHvcC(3, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu(vps)),
				newHvcNaluArray(33, newHvcNalu(nil)),
				newHvcNaluArray(34, newHvcNalu(pps)),
			}),
			"H265 parameter not provided",
		},
		{
			"empty PPS",
			newHvcC(3, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu(vps)),
				newHvcNaluArray(33, newHvcNalu(sps)),
				newHvcNaluArray(34, newHvcNalu(nil)),
			}),
			"H265 parameter not provided",
		},
		{
			"missing VPS",
			newHvcC(2, []amp4.HEVCNaluArray{
				newHvcNaluArray(33, newHvcNalu(sps)),
				newHvcNaluArray(34, newHvcNalu(pps)),
			}),
			"H265 parameters not provided",
		},
		{
			"missing SPS",
			newHvcC(2, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu(vps)),
				newHvcNaluArray(34, newHvcNalu(pps)),
			}),
			"H265 parameters not provided",
		},
		{
			"missing PPS",
			newHvcC(2, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu(vps)),
				newHvcNaluArray(33, newHvcNalu(sps)),
			}),
			"H265 parameters not provided",
		},
		{
			"multiple VPS",
			newHvcC(4, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu(vps)),
				newHvcNaluArray(32, newHvcNalu(vps)),
				newHvcNaluArray(33, newHvcNalu(sps)),
				newHvcNaluArray(34, newHvcNalu(pps)),
			}),
			"multiple H265 VPS are not supported",
		},
		{
			"multiple SPS",
			newHvcC(4, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu(vps)),
				newHvcNaluArray(33, newHvcNalu(sps)),
				newHvcNaluArray(33, newHvcNalu(sps)),
				newHvcNaluArray(34, newHvcNalu(pps)),
			}),
			"multiple H265 SPS are not supported",
		},
		{
			"multiple PPS",
			newHvcC(4, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu(vps)),
				newHvcNaluArray(33, newHvcNalu(sps)),
				newHvcNaluArray(34, newHvcNalu(pps)),
				newHvcNaluArray(34, newHvcNalu(pps)),
			}),
			"multiple H265 PPS are not supported",
		},
		{
			// unknown NALU arrays (e.g. SEI) are not parameters and are ignored.
			"unknown arrays",
			newHvcC(5, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu(vps)),
				newHvcNaluArray(33, newHvcNalu(sps)),
				newHvcNaluArray(34, newHvcNalu(pps)),
				newHvcNaluArray(39, newHvcNalu([]byte{0x01}), newHvcNalu([]byte{0x02})),
				newHvcNaluArray(40, newHvcNalu([]byte{0x03})),
			}),
			"",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			byts := buildInit(t, amp4.BoxTypeHvc1(), ca.codec)

			var init Init
			err := init.Unmarshal(bytes.NewReader(byts))
			if ca.err != "" {
				require.EqualError(t, err, ca.err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, &codecs.H265{VPS: vps, SPS: sps, PPS: pps}, init.Tracks[0].Codec)
		})
	}
}

func TestInitUnmarshalParamsCountMismatch(t *testing.T) {
	// the declared count is greater than the number of parameters that
	// are really contained in the box. Reading must fail inside the codec
	// box, without indexing into a slice that is too short.
	for _, ca := range []struct {
		name     string
		entry    amp4.BoxType
		codec    amp4.IImmutableBox
		boxType  string
		offset   int
		declared byte
	}{
		{
			"H264 SPS count",
			amp4.BoxTypeAvc1(),
			newAvcC(
				[]amp4.AVCParameterSet{
					{Length: uint16(len(testSPS)), NALUnit: testSPS},
					{Length: uint16(len(testSPS)), NALUnit: testSPS},
				},
				[]amp4.AVCParameterSet{{Length: 1, NALUnit: []byte{0x08}}},
			),
			"avcC", 9, 3,
		},
		{
			"H265 arrays count",
			amp4.BoxTypeHvc1(),
			newHvcC(3, []amp4.HEVCNaluArray{
				newHvcNaluArray(32, newHvcNalu([]byte{0x40, 0x01})),
				newHvcNaluArray(33, newHvcNalu([]byte{0x42, 0x01})),
				newHvcNaluArray(34, newHvcNalu([]byte{0x44, 0x01})),
			}),
			"hvcC", 26, 4,
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			byts := buildInit(t, ca.entry, ca.codec)

			// numOfSequenceParameterSets is 5 bytes after the box type,
			// numOfNaluArrays is 22 bytes after the box type.
			// Increase the declared amount of the codec box, so that it is
			// greater than the amount of parameters actually contained.
			pos := bytes.Index(byts, []byte(ca.boxType))
			require.NotEqual(t, -1, pos)
			byts[pos+ca.offset] = ca.declared

			var init Init
			err := init.Unmarshal(bytes.NewReader(byts))
			require.Error(t, err)
			require.NotErrorIs(t, err, imp4.ErrReadEnded)
		})
	}
}
