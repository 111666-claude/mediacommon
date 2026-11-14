package mp4

import (
	"bytes"
	"testing"

	amp4 "github.com/abema/go-mp4"
	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/formats/mp4/codecs"
)

var (
	testH264SPS = []byte{0x67, 0x42, 0xc0, 0x1f, 0xd9, 0x00, 0xf0, 0x11}
	testH264PPS = []byte{0x68, 0xcb, 0x8c, 0xb2}
	testH265VPS = []byte{0x40, 0x01, 0x0c, 0x01}
	testH265SPS = []byte{0x42, 0x01, 0x01, 0x01}
	testH265PPS = []byte{0x44, 0x01, 0xc0}
)

type paramSet struct {
	length uint16
	nalu   []byte
}

// buildAvcCPayload builds the payload of an AVCDecoderConfiguration box.
func buildAvcCPayload(numSPS uint8, sps []paramSet, numPPS uint8, pps []paramSet) []byte {
	buf := []byte{
		1,    // configurationVersion
		0x42, // AVCProfileIndication
		0x00, // profile_compatibility
		0x1f, // AVCLevelIndication
		0xfc, // lengthSizeMinusOne
		0xe0 | (numSPS & 0b11111),
	}

	for _, e := range sps {
		buf = append(buf, byte(e.length>>8), byte(e.length))
		buf = append(buf, e.nalu...)
	}

	buf = append(buf, numPPS)

	for _, e := range pps {
		buf = append(buf, byte(e.length>>8), byte(e.length))
		buf = append(buf, e.nalu...)
	}

	return buf
}

// buildHvcCPayload builds the payload of an HvcC box.
func buildHvcCPayload(numArrays uint8, arrays ...[]byte) []byte {
	buf := []byte{
		1,                      // configurationVersion
		0x00,                   // profile space / tier / profile idc
		0x00, 0x00, 0x00, 0x00, // general profile compatibility
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // general constraint indicator
		0x00,       // general level idc
		0xf0, 0x00, // reserved + min spatial segmentation idc
		0xfc,       // reserved + parallelism type
		0xfc,       // reserved + chroma format idc
		0xf8,       // reserved + bit depth luma
		0xf8,       // reserved + bit depth chroma
		0x00, 0x00, // avg frame rate
		0x0f,      // const frame rate + num temporal layers + temporal id nested + length size
		numArrays, // num of nalu arrays
	}

	for _, arr := range arrays {
		buf = append(buf, arr...)
	}

	return buf
}

// buildHvcCNaluArray builds an HEVCNaluArray. numNalus is written as is,
// so that it can be inconsistent with the number of NALUs on purpose.
func buildHvcCNaluArray(naluType uint8, numNalus uint16, nalus ...[]byte) []byte {
	buf := []byte{0x80 | (naluType & 0b111111), byte(numNalus >> 8), byte(numNalus)}

	for _, nalu := range nalus {
		buf = append(buf, byte(len(nalu)>>8), byte(len(nalu)))
		buf = append(buf, nalu...)
	}

	return buf
}

// newCodecBoxHandle builds a read handle for a codec-related box, as
// produced by abema/go-mp4 while traversing a stsd box.
func newCodecBoxHandle(boxType amp4.BoxType, readPayload func() (amp4.IBox, uint64, error)) *amp4.ReadHandle {
	path := make(amp4.BoxPath, 7)
	for i := range path {
		path[i] = amp4.BoxTypeFtyp()
	}

	return &amp4.ReadHandle{
		BoxInfo: amp4.BoxInfo{
			Type: boxType,
		},
		Path: path,
		ReadPayload: func() (amp4.IBox, uint64, error) {
			if readPayload == nil {
				return nil, 0, nil
			}
			return readPayload()
		},
		Expand: func(_ ...any) ([]any, error) {
			return nil, nil
		},
	}
}

// newAvcCHandle builds a read handle that decodes an avcC box from its
// raw payload, like abema/go-mp4 does.
func newAvcCHandle(payload []byte) *amp4.ReadHandle {
	return newCodecBoxHandle(amp4.BoxTypeAvcC(), func() (amp4.IBox, uint64, error) {
		box, n, err := amp4.UnmarshalAny(bytes.NewReader(payload), amp4.BoxTypeAvcC(), uint64(len(payload)), amp4.Context{})
		return box, n, err
	})
}

// newHvcCHandle builds a read handle that decodes an hvcC box from its
// raw payload, like abema/go-mp4 does.
func newHvcCHandle(payload []byte) *amp4.ReadHandle {
	return newCodecBoxHandle(amp4.BoxTypeHvcC(), func() (amp4.IBox, uint64, error) {
		box, n, err := amp4.UnmarshalAny(bytes.NewReader(payload), amp4.BoxTypeHvcC(), uint64(len(payload)), amp4.Context{})
		return box, n, err
	})
}

func TestH264FindParams(t *testing.T) {
	for _, ca := range []struct {
		name    string
		payload []byte
		sps     []byte
		pps     []byte
		err     string
	}{
		{
			"valid",
			buildAvcCPayload(1, []paramSet{{uint16(len(testH264SPS)), testH264SPS}}, 1,
				[]paramSet{{uint16(len(testH264PPS)), testH264PPS}}),
			testH264SPS,
			testH264PPS,
			"",
		},
		{
			"no SPS",
			buildAvcCPayload(0, nil, 1, []paramSet{{uint16(len(testH264PPS)), testH264PPS}}),
			nil, nil,
			"H264 parameters not provided",
		},
		{
			"no PPS",
			buildAvcCPayload(1, []paramSet{{uint16(len(testH264SPS)), testH264SPS}}, 0, nil),
			nil, nil,
			"H264 parameters not provided",
		},
		{
			"empty SPS",
			buildAvcCPayload(1, []paramSet{{0, []byte{}}}, 1,
				[]paramSet{{uint16(len(testH264PPS)), testH264PPS}}),
			nil, nil,
			"H264 parameters not provided",
		},
		{
			"empty PPS",
			buildAvcCPayload(1, []paramSet{{uint16(len(testH264SPS)), testH264SPS}}, 1,
				[]paramSet{{0, []byte{}}}),
			nil, nil,
			"H264 parameters not provided",
		},
		{
			"multiple SPS",
			buildAvcCPayload(2, []paramSet{
				{uint16(len(testH264SPS)), testH264SPS},
				{uint16(len(testH264SPS)), testH264SPS},
			}, 1, []paramSet{{uint16(len(testH264PPS)), testH264PPS}}),
			nil, nil,
			"multiple H264 parameters are not supported",
		},
		{
			"multiple PPS",
			buildAvcCPayload(1, []paramSet{{uint16(len(testH264SPS)), testH264SPS}}, 2, []paramSet{
				{uint16(len(testH264PPS)), testH264PPS},
				{uint16(len(testH264PPS)), testH264PPS},
			}),
			nil, nil,
			"multiple H264 parameters are not supported",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			box, _, err := amp4.UnmarshalAny(bytes.NewReader(ca.payload), amp4.BoxTypeAvcC(),
				uint64(len(ca.payload)), amp4.Context{})
			require.NoError(t, err)

			sps, pps, err := h264FindParams(box.(*amp4.AVCDecoderConfiguration))
			if ca.err != "" {
				require.EqualError(t, err, ca.err)
				require.Nil(t, sps)
				require.Nil(t, pps)
				return
			}

			require.NoError(t, err)
			require.Equal(t, ca.sps, sps)
			require.Equal(t, ca.pps, pps)
		})
	}
}

func TestH265FindParams(t *testing.T) {
	validArrays := [][]byte{
		buildHvcCNaluArray(32, 1, testH265VPS),
		buildHvcCNaluArray(33, 1, testH265SPS),
		buildHvcCNaluArray(34, 1, testH265PPS),
	}

	for _, ca := range []struct {
		name   string
		arrays [][]byte
		vps    []byte
		sps    []byte
		pps    []byte
		err    string
	}{
		{
			"valid",
			validArrays,
			testH265VPS, testH265SPS, testH265PPS,
			"",
		},
		{
			"valid with unknown arrays",
			[][]byte{
				validArrays[0],
				validArrays[1],
				validArrays[2],
				buildHvcCNaluArray(39, 2, []byte{0x01}, []byte{0x02}),
				buildHvcCNaluArray(40, 1, []byte{0x03}),
			},
			testH265VPS, testH265SPS, testH265PPS,
			"",
		},
		{
			"no arrays",
			nil,
			nil, nil, nil,
			"H265 parameters not provided",
		},
		{
			"no VPS",
			[][]byte{validArrays[1], validArrays[2]},
			nil, nil, nil,
			"H265 parameters not provided",
		},
		{
			"no SPS",
			[][]byte{validArrays[0], validArrays[2]},
			nil, nil, nil,
			"H265 parameters not provided",
		},
		{
			"no PPS",
			validArrays[:2],
			nil, nil, nil,
			"H265 parameters not provided",
		},
		{
			"empty VPS",
			[][]byte{
				buildHvcCNaluArray(32, 1, []byte{}),
				validArrays[1],
				validArrays[2],
			},
			nil, nil, nil,
			"H265 parameter not provided",
		},
		{
			"empty SPS",
			[][]byte{
				validArrays[0],
				buildHvcCNaluArray(33, 1, []byte{}),
				validArrays[2],
			},
			nil, nil, nil,
			"H265 parameter not provided",
		},
		{
			"empty PPS",
			[][]byte{
				validArrays[0],
				validArrays[1],
				buildHvcCNaluArray(34, 1, []byte{}),
			},
			nil, nil, nil,
			"H265 parameter not provided",
		},
		{
			"multiple VPS",
			[][]byte{
				validArrays[0],
				validArrays[0],
				validArrays[1],
				validArrays[2],
			},
			nil, nil, nil,
			"multiple H265 VPS are not supported",
		},
		{
			"multiple SPS",
			[][]byte{
				validArrays[0],
				validArrays[1],
				validArrays[1],
				validArrays[2],
			},
			nil, nil, nil,
			"multiple H265 SPS are not supported",
		},
		{
			"multiple PPS",
			[][]byte{
				validArrays[0],
				validArrays[1],
				validArrays[2],
				validArrays[2],
			},
			nil, nil, nil,
			"multiple H265 PPS are not supported",
		},
		{
			"two NALUs in VPS array",
			[][]byte{
				buildHvcCNaluArray(32, 2, testH265VPS, testH265VPS),
				validArrays[1],
				validArrays[2],
			},
			nil, nil, nil,
			"multiple H265 parameters are not supported",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			payload := buildHvcCPayload(uint8(len(ca.arrays)), ca.arrays...)

			box, _, err := amp4.UnmarshalAny(bytes.NewReader(payload), amp4.BoxTypeHvcC(), uint64(len(payload)), amp4.Context{})
			require.NoError(t, err)

			vps, sps, pps, err := h265FindParams(box.(*amp4.HvcC))
			if ca.err != "" {
				require.EqualError(t, err, ca.err)
				require.Nil(t, vps)
				require.Nil(t, sps)
				require.Nil(t, pps)
				return
			}

			require.NoError(t, err)
			require.Equal(t, ca.vps, vps)
			require.Equal(t, ca.sps, sps)
			require.Equal(t, ca.pps, pps)
		})
	}
}

func TestH265FindParamsEmptyNalusSlice(t *testing.T) {
	// an array that declares a NALU but doesn't contain it must be rejected
	// without indexing into an empty slice.
	_, _, _, err := h265FindParams(&amp4.HvcC{
		NaluArrays: []amp4.HEVCNaluArray{
			{NaluType: 32, NumNalus: 1},
			{NaluType: 33, NumNalus: 1, Nalus: []amp4.HEVCNalu{{NALUnit: testH265SPS}}},
			{NaluType: 34, NumNalus: 1, Nalus: []amp4.HEVCNalu{{NALUnit: testH265PPS}}},
		},
	})
	require.EqualError(t, err, "multiple H265 parameters are not supported")
}

func TestCodecBoxesReaderParams(t *testing.T) {
	for _, ca := range []struct {
		name     string
		entry    string
		payload  []byte
		expected codecs.Codec
		err      string
	}{
		{
			"h264",
			"avc1",
			buildAvcCPayload(1, []paramSet{{uint16(len(testH264SPS)), testH264SPS}}, 1,
				[]paramSet{{uint16(len(testH264PPS)), testH264PPS}}),
			&codecs.H264{SPS: testH264SPS, PPS: testH264PPS},
			"",
		},
		{
			"h264 empty parameters",
			"avc1",
			buildAvcCPayload(1, []paramSet{{0, []byte{}}}, 1, []paramSet{{0, []byte{}}}),
			nil,
			"H264 parameters not provided",
		},
		{
			"h265",
			"hvc1",
			buildHvcCPayload(3,
				buildHvcCNaluArray(32, 1, testH265VPS),
				buildHvcCNaluArray(33, 1, testH265SPS),
				buildHvcCNaluArray(34, 1, testH265PPS)),
			&codecs.H265{VPS: testH265VPS, SPS: testH265SPS, PPS: testH265PPS},
			"",
		},
		{
			"h265 empty parameters",
			"hev1",
			buildHvcCPayload(3,
				buildHvcCNaluArray(32, 1, []byte{}),
				buildHvcCNaluArray(33, 1, []byte{}),
				buildHvcCNaluArray(34, 1, []byte{})),
			nil,
			"H265 parameter not provided",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			r := &CodecBoxesReader{}
			_, err := r.Read(newCodecBoxHandle(amp4.StrToBoxType(ca.entry), nil))
			require.NoError(t, err)

			h := newAvcCHandle(ca.payload)
			if ca.entry == "hvc1" || ca.entry == "hev1" {
				h = newHvcCHandle(ca.payload)
			}

			_, err = r.Read(h)
			if ca.err != "" {
				require.EqualError(t, err, ca.err)
				require.Nil(t, r.Codec)
				return
			}

			require.NoError(t, err)
			require.Equal(t, ca.expected, r.Codec)
		})
	}
}

// TestCodecBoxesReaderRecovery checks that a rejected parameter box does
// not corrupt the reader: the very same reader must be able to decode a
// subsequent valid codec box.
func TestCodecBoxesReaderRecovery(t *testing.T) {
	for _, ca := range []struct {
		name     string
		entry    string
		invalid  []byte
		valid    []byte
		expected codecs.Codec
	}{
		{
			"h264",
			"avc1",
			buildAvcCPayload(1, []paramSet{{0, []byte{}}}, 1, []paramSet{{0, []byte{}}}),
			buildAvcCPayload(1, []paramSet{{uint16(len(testH264SPS)), testH264SPS}}, 1,
				[]paramSet{{uint16(len(testH264PPS)), testH264PPS}}),
			&codecs.H264{SPS: testH264SPS, PPS: testH264PPS},
		},
		{
			"h265",
			"hvc1",
			buildHvcCPayload(3,
				buildHvcCNaluArray(32, 1, []byte{}),
				buildHvcCNaluArray(33, 1, []byte{}),
				buildHvcCNaluArray(34, 1, []byte{})),
			buildHvcCPayload(3,
				buildHvcCNaluArray(32, 1, testH265VPS),
				buildHvcCNaluArray(33, 1, testH265SPS),
				buildHvcCNaluArray(34, 1, testH265PPS)),
			&codecs.H265{VPS: testH265VPS, SPS: testH265SPS, PPS: testH265PPS},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			r := &CodecBoxesReader{}
			_, err := r.Read(newCodecBoxHandle(amp4.StrToBoxType(ca.entry), nil))
			require.NoError(t, err)

			h := newAvcCHandle(ca.invalid)
			hValid := newAvcCHandle(ca.valid)
			if ca.entry == "hvc1" {
				h = newHvcCHandle(ca.invalid)
				hValid = newHvcCHandle(ca.valid)
			}

			_, err = r.Read(h)
			require.Error(t, err)
			require.Nil(t, r.Codec)

			// the same reader must not be poisoned by the error: the next
			// valid codec box of the same kind is decoded normally.
			_, err = r.Read(hValid)
			require.NoError(t, err)
			require.Equal(t, ca.expected, r.Codec)

			// a second, independent reader must decode it as well.
			r2 := &CodecBoxesReader{}
			_, err = r2.Read(newCodecBoxHandle(amp4.StrToBoxType(ca.entry), nil))
			require.NoError(t, err)

			_, err = r2.Read(hValid)
			require.NoError(t, err)
			require.Equal(t, ca.expected, r2.Codec)
		})
	}
}

func TestCodecBoxesReaderCountMismatch(t *testing.T) {
	for _, ca := range []struct {
		name    string
		entry   string
		payload []byte
	}{
		{
			"h264 SPS count greater than actual",
			"avc1",
			buildAvcCPayload(2, []paramSet{{uint16(len(testH264SPS)), testH264SPS}}, 1,
				[]paramSet{{uint16(len(testH264PPS)), testH264PPS}}),
		},
		{
			"h264 SPS payload missing",
			"avc1",
			buildAvcCPayload(1, []paramSet{{0x10, []byte{0x67}}}, 1,
				[]paramSet{{uint16(len(testH264PPS)), testH264PPS}}),
		},
		{
			"h265 arrays count greater than actual",
			"hvc1",
			buildHvcCPayload(4,
				buildHvcCNaluArray(32, 1, testH265VPS),
				buildHvcCNaluArray(33, 1, testH265SPS),
				buildHvcCNaluArray(34, 1, testH265PPS)),
		},
		{
			"h265 NALUs count greater than actual",
			"hev1",
			buildHvcCPayload(3,
				buildHvcCNaluArray(32, 1, testH265VPS),
				buildHvcCNaluArray(33, 2, testH265SPS),
				buildHvcCNaluArray(34, 1, testH265PPS)),
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			r := &CodecBoxesReader{}
			_, err := r.Read(newCodecBoxHandle(amp4.StrToBoxType(ca.entry), nil))
			require.NoError(t, err)

			// the box declares more parameters than it contains:
			// the error must be raised while reading the codec box,
			// without indexing into a slice that is too short.
			h := newAvcCHandle(ca.payload)
			if ca.entry == "hvc1" || ca.entry == "hev1" {
				h = newHvcCHandle(ca.payload)
			}

			_, err = r.Read(h)
			require.Error(t, err)
			require.NotEqual(t, ErrReadEnded, err)
			require.Nil(t, r.Codec)
		})
	}
}
