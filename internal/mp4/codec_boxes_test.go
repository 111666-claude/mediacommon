package mp4

import (
	"bytes"
	"encoding/binary"
	"testing"

	amp4 "github.com/abema/go-mp4"
	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/formats/mp4/codecs"
)

var (
	testH264SPS = []byte{0x67, 0x42, 0xc0, 0x28}
	testH264PPS = []byte{0x68, 0xce, 0x3c, 0x80}
	testH265VPS = []byte{0x40, 0x01, 0x0c, 0x01}
	testH265SPS = []byte{0x42, 0x01, 0x01, 0x01}
	testH265PPS = []byte{0x44, 0x01, 0xc0, 0x25}
)

func TestH264FindParams(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		sps, pps, err := h264FindParams(&amp4.AVCDecoderConfiguration{
			SequenceParameterSets: []amp4.AVCParameterSet{
				{Length: uint16(len(testH264SPS)), NALUnit: testH264SPS},
			},
			PictureParameterSets: []amp4.AVCParameterSet{
				{Length: uint16(len(testH264PPS)), NALUnit: testH264PPS},
			},
		})
		require.NoError(t, err)
		require.Equal(t, testH264SPS, sps)
		require.Equal(t, testH264PPS, pps)
	})

	t.Run("missing sps", func(t *testing.T) {
		_, _, err := h264FindParams(&amp4.AVCDecoderConfiguration{
			PictureParameterSets: []amp4.AVCParameterSet{
				{Length: uint16(len(testH264PPS)), NALUnit: testH264PPS},
			},
		})
		require.EqualError(t, err, "H264 SPS not provided")
	})

	t.Run("empty sps", func(t *testing.T) {
		_, _, err := h264FindParams(&amp4.AVCDecoderConfiguration{
			SequenceParameterSets: []amp4.AVCParameterSet{
				{Length: 0, NALUnit: nil},
			},
			PictureParameterSets: []amp4.AVCParameterSet{
				{Length: uint16(len(testH264PPS)), NALUnit: testH264PPS},
			},
		})
		require.EqualError(t, err, "H264 SPS is empty")
	})

	t.Run("multiple sps", func(t *testing.T) {
		_, _, err := h264FindParams(&amp4.AVCDecoderConfiguration{
			SequenceParameterSets: []amp4.AVCParameterSet{
				{Length: uint16(len(testH264SPS)), NALUnit: testH264SPS},
				{Length: uint16(len(testH264SPS)), NALUnit: testH264SPS},
			},
			PictureParameterSets: []amp4.AVCParameterSet{
				{Length: uint16(len(testH264PPS)), NALUnit: testH264PPS},
			},
		})
		require.EqualError(t, err, "multiple H264 SPS are not supported")
	})

	t.Run("missing pps", func(t *testing.T) {
		_, _, err := h264FindParams(&amp4.AVCDecoderConfiguration{
			SequenceParameterSets: []amp4.AVCParameterSet{
				{Length: uint16(len(testH264SPS)), NALUnit: testH264SPS},
			},
		})
		require.EqualError(t, err, "H264 PPS not provided")
	})

	t.Run("empty pps", func(t *testing.T) {
		_, _, err := h264FindParams(&amp4.AVCDecoderConfiguration{
			SequenceParameterSets: []amp4.AVCParameterSet{
				{Length: uint16(len(testH264SPS)), NALUnit: testH264SPS},
			},
			PictureParameterSets: []amp4.AVCParameterSet{
				{Length: 0, NALUnit: nil},
			},
		})
		require.EqualError(t, err, "H264 PPS is empty")
	})

	t.Run("multiple pps", func(t *testing.T) {
		_, _, err := h264FindParams(&amp4.AVCDecoderConfiguration{
			SequenceParameterSets: []amp4.AVCParameterSet{
				{Length: uint16(len(testH264SPS)), NALUnit: testH264SPS},
			},
			PictureParameterSets: []amp4.AVCParameterSet{
				{Length: uint16(len(testH264PPS)), NALUnit: testH264PPS},
				{Length: uint16(len(testH264PPS)), NALUnit: testH264PPS},
			},
		})
		require.EqualError(t, err, "multiple H264 PPS are not supported")
	})
}

func TestH265FindParams(t *testing.T) {
	naluArray := func(typ byte, nalus ...[]byte) amp4.HEVCNaluArray {
		arr := amp4.HEVCNaluArray{
			NaluType: typ,
			NumNalus: uint16(len(nalus)),
		}
		for _, nalu := range nalus {
			arr.Nalus = append(arr.Nalus, amp4.HEVCNalu{
				Length:  uint16(len(nalu)),
				NALUnit: nalu,
			})
		}
		return arr
	}

	t.Run("valid", func(t *testing.T) {
		vps, sps, pps, err := h265FindParams([]amp4.HEVCNaluArray{
			naluArray(32, testH265VPS),
			naluArray(33, testH265SPS),
			naluArray(34, testH265PPS),
		})
		require.NoError(t, err)
		require.Equal(t, testH265VPS, vps)
		require.Equal(t, testH265SPS, sps)
		require.Equal(t, testH265PPS, pps)
	})

	t.Run("valid with unknown array", func(t *testing.T) {
		vps, sps, pps, err := h265FindParams([]amp4.HEVCNaluArray{
			naluArray(35, []byte{0x01}), // AUD
			naluArray(32, testH265VPS),
			naluArray(33, testH265SPS),
			naluArray(34, testH265PPS),
		})
		require.NoError(t, err)
		require.Equal(t, testH265VPS, vps)
		require.Equal(t, testH265SPS, sps)
		require.Equal(t, testH265PPS, pps)
	})

	t.Run("missing vps", func(t *testing.T) {
		_, _, _, err := h265FindParams([]amp4.HEVCNaluArray{
			naluArray(33, testH265SPS),
			naluArray(34, testH265PPS),
		})
		require.EqualError(t, err, "H265 VPS not provided")
	})

	t.Run("missing sps", func(t *testing.T) {
		_, _, _, err := h265FindParams([]amp4.HEVCNaluArray{
			naluArray(32, testH265VPS),
			naluArray(34, testH265PPS),
		})
		require.EqualError(t, err, "H265 SPS not provided")
	})

	t.Run("missing pps", func(t *testing.T) {
		_, _, _, err := h265FindParams([]amp4.HEVCNaluArray{
			naluArray(32, testH265VPS),
			naluArray(33, testH265SPS),
		})
		require.EqualError(t, err, "H265 PPS not provided")
	})

	t.Run("empty vps", func(t *testing.T) {
		_, _, _, err := h265FindParams([]amp4.HEVCNaluArray{
			naluArray(32, nil),
			naluArray(33, testH265SPS),
			naluArray(34, testH265PPS),
		})
		require.EqualError(t, err, "H265 VPS is empty")
	})

	t.Run("empty sps", func(t *testing.T) {
		_, _, _, err := h265FindParams([]amp4.HEVCNaluArray{
			naluArray(32, testH265VPS),
			naluArray(33, nil),
			naluArray(34, testH265PPS),
		})
		require.EqualError(t, err, "H265 SPS is empty")
	})

	t.Run("empty pps", func(t *testing.T) {
		_, _, _, err := h265FindParams([]amp4.HEVCNaluArray{
			naluArray(32, testH265VPS),
			naluArray(33, testH265SPS),
			naluArray(34, nil),
		})
		require.EqualError(t, err, "H265 PPS is empty")
	})

	t.Run("duplicate vps", func(t *testing.T) {
		_, _, _, err := h265FindParams([]amp4.HEVCNaluArray{
			naluArray(32, testH265VPS),
			naluArray(32, testH265VPS),
			naluArray(33, testH265SPS),
			naluArray(34, testH265PPS),
		})
		require.EqualError(t, err, "multiple H265 VPS are not supported")
	})

	t.Run("multiple nalus in array", func(t *testing.T) {
		_, _, _, err := h265FindParams([]amp4.HEVCNaluArray{
			naluArray(32, testH265VPS, testH265VPS),
			naluArray(33, testH265SPS),
			naluArray(34, testH265PPS),
		})
		require.EqualError(t, err, "multiple H265 VPS are not supported")
	})

	t.Run("declared count mismatches actual nalus", func(t *testing.T) {
		// NumNalus declares 1 NALU but the array contains none:
		// must return an error, not panic with an out-of-bounds access.
		_, _, _, err := h265FindParams([]amp4.HEVCNaluArray{
			{NaluType: 32, NumNalus: 1},
			naluArray(33, testH265SPS),
			naluArray(34, testH265PPS),
		})
		require.EqualError(t, err, "H265 VPS not provided")
	})
}

// testBox builds a MP4 box with a 32-bit size header.
func testBox(typ string, payload []byte) []byte {
	buf := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(buf, uint32(8+len(payload)))
	copy(buf[4:], typ)
	copy(buf[8:], payload)
	return buf
}

// testVisualSampleEntry builds the 78-byte header of a visual sample entry,
// followed by the given child boxes.
func testVisualSampleEntry(children ...[]byte) []byte {
	payload := make([]byte, 78)
	payload[7] = 0x01 // data reference index
	for _, child := range children {
		payload = append(payload, child...)
	}
	return payload
}

// testStsd builds the payload of a stsd box containing the given entries.
func testStsd(entries ...[]byte) []byte {
	payload := make([]byte, 8)
	binary.BigEndian.PutUint32(payload[4:], uint32(len(entries)))
	for _, entry := range entries {
		payload = append(payload, entry...)
	}
	return payload
}

// testAvcC builds the payload of an avcC box with the given SPS and PPS.
// A nil SPS or PPS produces an empty NALU.
func testAvcC(sps, pps []byte) []byte {
	payload := []byte{
		0x01,             // configuration version
		0x42, 0x00, 0x1E, // profile, profile compatibility, level
		0xFF, // reserved + length size minus one
		0xE1, // reserved + num of SPS
	}
	payload = append(payload, byte(len(sps)>>8), byte(len(sps)))
	payload = append(payload, sps...)
	payload = append(payload, 0x01) // num of PPS
	payload = append(payload, byte(len(pps)>>8), byte(len(pps)))
	payload = append(payload, pps...)
	return payload
}

// testHvcCNaluArray builds a NALU array of an hvcC box.
// A nil NALU produces an empty NALU.
func testHvcCNaluArray(naluType byte, nalus ...[]byte) []byte {
	payload := []byte{naluType, byte(len(nalus) >> 8), byte(len(nalus))}
	for _, nalu := range nalus {
		payload = append(payload, byte(len(nalu)>>8), byte(len(nalu)))
		payload = append(payload, nalu...)
	}
	return payload
}

// testHvcC builds the payload of an hvcC box with the given NALU arrays.
func testHvcC(arrays ...[]byte) []byte {
	payload := []byte{
		0x01,                   // configuration version
		0x00,                   // profile space, tier flag, profile idc
		0x00, 0x00, 0x00, 0x00, // profile compatibility
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // constraint indicator
		0x00,       // level idc
		0xF0, 0x00, // reserved + min spatial segmentation idc
		0xFC,       // reserved + parallelism type
		0xFC,       // reserved + chroma format idc
		0xF8,       // reserved + bit depth luma minus 8
		0xF8,       // reserved + bit depth chroma minus 8
		0x00, 0x00, // avg frame rate
		0x00,              // constant frame rate, num temporal layers, temporal id nested, length size minus one
		byte(len(arrays)), // num of NALU arrays
	}
	for _, arr := range arrays {
		payload = append(payload, arr...)
	}
	return payload
}

// testCodecBoxes wraps a codec box payload into a
// moov/trak/mdia/minf/stbl/stsd/sample-entry/codec-box structure.
func testCodecBoxes(sampleEntryType, codecBoxType string, codecBoxPayload []byte) []byte {
	return testBox("moov", testBox("trak", testBox("mdia", testBox("minf", testBox("stbl",
		testBox("stsd", testStsd(testBox(sampleEntryType,
			testVisualSampleEntry(testBox(codecBoxType, codecBoxPayload))))))))))
}

func readCodecBoxes(t *testing.T, r *CodecBoxesReader, byts []byte) error {
	t.Helper()

	_, err := amp4.ReadBoxStructure(bytes.NewReader(byts), func(h *amp4.ReadHandle) (any, error) {
		switch h.BoxInfo.Type.String() {
		case "moov", "trak", "mdia", "minf", "stbl", "stsd":
			return h.Expand()

		default:
			return r.Read(h)
		}
	})
	return err
}

func TestCodecBoxesReaderH264(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		var r CodecBoxesReader
		err := readCodecBoxes(t, &r, testCodecBoxes("avc1", "avcC", testAvcC(testH264SPS, testH264PPS)))
		require.NoError(t, err)
		require.Equal(t, &codecs.H264{
			SPS: testH264SPS,
			PPS: testH264PPS,
		}, r.Codec)
	})

	t.Run("empty sps", func(t *testing.T) {
		var r CodecBoxesReader
		err := readCodecBoxes(t, &r, testCodecBoxes("avc1", "avcC", testAvcC(nil, testH264PPS)))
		require.EqualError(t, err, "H264 SPS is empty")
		require.Nil(t, r.Codec)
	})

	t.Run("empty pps", func(t *testing.T) {
		var r CodecBoxesReader
		err := readCodecBoxes(t, &r, testCodecBoxes("avc1", "avcC", testAvcC(testH264SPS, nil)))
		require.EqualError(t, err, "H264 PPS is empty")
		require.Nil(t, r.Codec)
	})

	t.Run("state is not corrupted by invalid parameters", func(t *testing.T) {
		var r CodecBoxesReader

		err := readCodecBoxes(t, &r, testCodecBoxes("avc1", "avcC", testAvcC(nil, testH264PPS)))
		require.EqualError(t, err, "H264 SPS is empty")
		require.Nil(t, r.Codec)

		// the failed read must not have advanced the reader state:
		// a valid avcC box that follows is read correctly.
		err = readCodecBoxes(t, &r, testBox("moov", testBox("trak", testBox("mdia", testBox("minf",
			testBox("stbl", testBox("stsd", testStsd(testBox("avcC", testAvcC(testH264SPS, testH264PPS))))))))))
		require.NoError(t, err)
		require.Equal(t, &codecs.H264{
			SPS: testH264SPS,
			PPS: testH264PPS,
		}, r.Codec)
	})
}

func TestCodecBoxesReaderH265(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		var r CodecBoxesReader
		err := readCodecBoxes(t, &r, testCodecBoxes("hvc1", "hvcC", testHvcC(
			testHvcCNaluArray(32, testH265VPS),
			testHvcCNaluArray(33, testH265SPS),
			testHvcCNaluArray(34, testH265PPS),
		)))
		require.NoError(t, err)
		require.Equal(t, &codecs.H265{
			VPS: testH265VPS,
			SPS: testH265SPS,
			PPS: testH265PPS,
		}, r.Codec)
	})

	t.Run("empty vps", func(t *testing.T) {
		var r CodecBoxesReader
		err := readCodecBoxes(t, &r, testCodecBoxes("hvc1", "hvcC", testHvcC(
			testHvcCNaluArray(32, nil),
			testHvcCNaluArray(33, testH265SPS),
			testHvcCNaluArray(34, testH265PPS),
		)))
		require.EqualError(t, err, "H265 VPS is empty")
		require.Nil(t, r.Codec)
	})

	t.Run("duplicate vps", func(t *testing.T) {
		var r CodecBoxesReader
		err := readCodecBoxes(t, &r, testCodecBoxes("hvc1", "hvcC", testHvcC(
			testHvcCNaluArray(32, testH265VPS),
			testHvcCNaluArray(32, testH265VPS),
			testHvcCNaluArray(33, testH265SPS),
			testHvcCNaluArray(34, testH265PPS),
		)))
		require.EqualError(t, err, "multiple H265 VPS are not supported")
		require.Nil(t, r.Codec)
	})

	t.Run("missing pps", func(t *testing.T) {
		var r CodecBoxesReader
		err := readCodecBoxes(t, &r, testCodecBoxes("hvc1", "hvcC", testHvcC(
			testHvcCNaluArray(32, testH265VPS),
			testHvcCNaluArray(33, testH265SPS),
		)))
		require.EqualError(t, err, "H265 PPS not provided")
		require.Nil(t, r.Codec)
	})
}
