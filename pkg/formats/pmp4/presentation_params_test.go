package pmp4

import (
	"bytes"
	"testing"

	amp4 "github.com/abema/go-mp4"
	"github.com/stretchr/testify/require"

	imp4 "github.com/bluenviron/mediacommon/v2/internal/mp4"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/fmp4/seekablebuffer"
)

var testParamsSPS = []byte{
	0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02,
	0x27, 0xe5, 0x84, 0x00, 0x00, 0x03, 0x00, 0x04,
	0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9,
	0x20,
}

// buildPresentation builds a MP4 presentation that contains a single
// video track whose codec box is provided by the caller.
func buildPresentation(t *testing.T, entry amp4.BoxType, codecBox amp4.IImmutableBox) []byte {
	t.Helper()

	var buf seekablebuffer.Buffer
	mw := &imp4.Writer{W: &buf}
	mw.Initialize()

	_, err := mw.WriteBox(&amp4.Ftyp{
		MajorBrand:       [4]byte{'i', 's', 'o', 'm'},
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
		FullBox:    amp4.FullBox{Flags: [3]byte{0, 0, 3}},
		TrackID:    1,
		DurationV0: 90000,
		Width:      1280 * 65536,
		Height:     720 * 65536,
		Matrix:     [9]int32{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
	})
	require.NoError(t, err)

	_, err = mw.WriteBoxStart(&amp4.Edts{})
	require.NoError(t, err)

	_, err = mw.WriteBox(&amp4.Elst{
		EntryCount: 1,
		Entries: []amp4.ElstEntry{{
			SegmentDurationV0: 1000,
			MediaRateInteger:  1,
		}},
	})
	require.NoError(t, err)

	require.NoError(t, mw.WriteBoxEnd()) // </edts>

	_, err = mw.WriteBoxStart(&amp4.Mdia{})
	require.NoError(t, err)

	_, err = mw.WriteBox(&amp4.Mdhd{
		Timescale:  90000,
		DurationV0: 90000,
		Language:   [3]byte{'u', 'n', 'd'},
	})
	require.NoError(t, err)

	_, err = mw.WriteBoxStart(&amp4.Minf{})
	require.NoError(t, err)

	_, err = mw.WriteBox(&amp4.Vmhd{FullBox: amp4.FullBox{Flags: [3]byte{0, 0, 1}}})
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

	_, err = mw.WriteBox(&amp4.Stts{
		EntryCount: 1,
		Entries:    []amp4.SttsEntry{{SampleCount: 1, SampleDelta: 90000}},
	})
	require.NoError(t, err)

	_, err = mw.WriteBox(&amp4.Stsc{
		EntryCount: 1,
		Entries:    []amp4.StscEntry{{FirstChunk: 1, SamplesPerChunk: 1, SampleDescriptionIndex: 1}},
	})
	require.NoError(t, err)

	_, err = mw.WriteBox(&amp4.Stsz{
		SampleCount: 1,
		EntrySize:   []uint32{2},
	})
	require.NoError(t, err)

	_, err = mw.WriteBox(&amp4.Stco{
		EntryCount:  1,
		ChunkOffset: []uint32{0},
	})
	require.NoError(t, err)

	require.NoError(t, mw.WriteBoxEnd()) // </stbl>
	require.NoError(t, mw.WriteBoxEnd()) // </minf>
	require.NoError(t, mw.WriteBoxEnd()) // </mdia>
	require.NoError(t, mw.WriteBoxEnd()) // </trak>
	require.NoError(t, mw.WriteBoxEnd()) // </moov>

	return buf.Bytes()
}

func buildH264ParamsBox(sps []amp4.AVCParameterSet, pps []amp4.AVCParameterSet) *amp4.AVCDecoderConfiguration {
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

func buildH265ParamsBox(numArrays uint8, arrays []amp4.HEVCNaluArray) *amp4.HvcC {
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

func buildH265NaluArray(naluType byte, nalus ...amp4.HEVCNalu) amp4.HEVCNaluArray {
	return amp4.HEVCNaluArray{
		Completeness: true,
		NaluType:     naluType,
		NumNalus:     uint16(len(nalus)),
		Nalus:        nalus,
	}
}

func buildH265Nalu(data []byte) amp4.HEVCNalu {
	return amp4.HEVCNalu{
		Length:  uint16(len(data)),
		NALUnit: data,
	}
}

func TestPresentationUnmarshalH264Params(t *testing.T) {
	for _, ca := range []struct {
		name  string
		codec amp4.IImmutableBox
		err   string
	}{
		{
			"valid",
			buildH264ParamsBox(
				[]amp4.AVCParameterSet{{Length: uint16(len(testParamsSPS)), NALUnit: testParamsSPS}},
				[]amp4.AVCParameterSet{{Length: 1, NALUnit: []byte{0x08}}},
			),
			"",
		},
		{
			"empty SPS",
			buildH264ParamsBox(
				[]amp4.AVCParameterSet{{Length: 0, NALUnit: []byte{}}},
				[]amp4.AVCParameterSet{{Length: 1, NALUnit: []byte{0x08}}},
			),
			"H264 parameters not provided",
		},
		{
			"empty PPS",
			buildH264ParamsBox(
				[]amp4.AVCParameterSet{{Length: uint16(len(testParamsSPS)), NALUnit: testParamsSPS}},
				[]amp4.AVCParameterSet{{Length: 0, NALUnit: []byte{}}},
			),
			"H264 parameters not provided",
		},
		{
			"multiple SPS",
			buildH264ParamsBox(
				[]amp4.AVCParameterSet{
					{Length: uint16(len(testParamsSPS)), NALUnit: testParamsSPS},
					{Length: uint16(len(testParamsSPS)), NALUnit: testParamsSPS},
				},
				[]amp4.AVCParameterSet{{Length: 1, NALUnit: []byte{0x08}}},
			),
			"multiple H264 parameters are not supported",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			byts := buildPresentation(t, amp4.BoxTypeAvc1(), ca.codec)

			var p Presentation
			err := p.Unmarshal(bytes.NewReader(byts))
			if ca.err != "" {
				require.EqualError(t, err, ca.err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, 1, len(p.Tracks))
		})
	}
}

func TestPresentationUnmarshalH265Params(t *testing.T) {
	vps := []byte{0x40, 0x01}
	sps := []byte{0x42, 0x01}
	pps := []byte{0x44, 0x01}

	for _, ca := range []struct {
		name      string
		numArrays uint8
		arrays    []amp4.HEVCNaluArray
		err       string
	}{
		{
			"valid",
			3,
			[]amp4.HEVCNaluArray{
				buildH265NaluArray(32, buildH265Nalu(vps)),
				buildH265NaluArray(33, buildH265Nalu(sps)),
				buildH265NaluArray(34, buildH265Nalu(pps)),
			},
			"",
		},
		{
			"empty VPS",
			3,
			[]amp4.HEVCNaluArray{
				buildH265NaluArray(32, buildH265Nalu(nil)),
				buildH265NaluArray(33, buildH265Nalu(sps)),
				buildH265NaluArray(34, buildH265Nalu(pps)),
			},
			"H265 parameter not provided",
		},
		{
			"empty SPS",
			3,
			[]amp4.HEVCNaluArray{
				buildH265NaluArray(32, buildH265Nalu(vps)),
				buildH265NaluArray(33, buildH265Nalu(nil)),
				buildH265NaluArray(34, buildH265Nalu(pps)),
			},
			"H265 parameter not provided",
		},
		{
			"empty PPS",
			3,
			[]amp4.HEVCNaluArray{
				buildH265NaluArray(32, buildH265Nalu(vps)),
				buildH265NaluArray(33, buildH265Nalu(sps)),
				buildH265NaluArray(34, buildH265Nalu(nil)),
			},
			"H265 parameter not provided",
		},
		{
			"missing VPS",
			2,
			[]amp4.HEVCNaluArray{
				buildH265NaluArray(33, buildH265Nalu(sps)),
				buildH265NaluArray(34, buildH265Nalu(pps)),
			},
			"H265 parameters not provided",
		},
		{
			"multiple PPS",
			4,
			[]amp4.HEVCNaluArray{
				buildH265NaluArray(32, buildH265Nalu(vps)),
				buildH265NaluArray(33, buildH265Nalu(sps)),
				buildH265NaluArray(34, buildH265Nalu(pps)),
				buildH265NaluArray(34, buildH265Nalu(pps)),
			},
			"multiple H265 PPS are not supported",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			byts := buildPresentation(t, amp4.BoxTypeHvc1(), buildH265ParamsBox(ca.numArrays, ca.arrays))

			var p Presentation
			err := p.Unmarshal(bytes.NewReader(byts))
			if ca.err != "" {
				require.EqualError(t, err, ca.err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, 1, len(p.Tracks))
		})
	}
}
