package h264_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
)

var (
	seiUserDataUUID1 = [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}

	seiUserDataUUID2 = [16]byte{
		0x10, 0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09,
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01,
	}
)

// SEI NALU that contains a recovery point message followed by
// two user_data_unregistered messages. The payload of the second
// message requires emulation prevention bytes.
var seiUserDataSample = []byte{
	0x06,
	// recovery point
	0x06, 0x01, 0xc4,
	// user_data_unregistered 1
	0x05, 0x18,
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x63, 0x61, 0x6d, 0x65, 0x72, 0x61, 0x2d, 0x41, // "camera-A"
	// user_data_unregistered 2
	0x05, 0x17,
	0x10, 0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09,
	0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01,
	0x00, 0x00, 0x03, 0x01, 0x00, 0x00, 0x03, 0x00, 0x02,
	// rbsp_trailing_bits
	0x80,
}

var seiUserDataSampleMessages = []h264.SEIUserDataUnregistered{
	{
		UUID:    seiUserDataUUID1,
		Payload: []byte("camera-A"),
	},
	{
		UUID:    seiUserDataUUID2,
		Payload: []byte{0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x02},
	},
}

// SEI NALU that contains the two user_data_unregistered messages only.
var seiUserDataMarshaledSample = []byte{
	0x06,
	// user_data_unregistered 1
	0x05, 0x18,
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x63, 0x61, 0x6d, 0x65, 0x72, 0x61, 0x2d, 0x41, // "camera-A"
	// user_data_unregistered 2
	0x05, 0x17,
	0x10, 0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09,
	0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01,
	0x00, 0x00, 0x03, 0x01, 0x00, 0x00, 0x03, 0x00, 0x02,
	// rbsp_trailing_bits
	0x80,
}

func TestSEIUserDataUnregisteredExtract(t *testing.T) {
	t.Run("sample", func(t *testing.T) {
		msgs, err := h264.SEIUserDataUnregisteredExtract(seiUserDataSample)
		require.NoError(t, err)
		require.Equal(t, seiUserDataSampleMessages, msgs)
	})

	t.Run("no user data", func(t *testing.T) {
		msgs, err := h264.SEIUserDataUnregisteredExtract([]byte{0x06, 0x06, 0x01, 0xc4, 0x80})
		require.NoError(t, err)
		require.Empty(t, msgs)
	})

	t.Run("extended payload type", func(t *testing.T) {
		msgs, err := h264.SEIUserDataUnregisteredExtract(
			[]byte{0x06, 0xff, 0x05, 0x02, 0xaa, 0xbb, 0x80})
		require.NoError(t, err)
		require.Empty(t, msgs)
	})

	t.Run("extended payload size", func(t *testing.T) {
		payload := bytes.Repeat([]byte{0x42}, 244)
		nalu := []byte{0x06, 0x05, 0xff, 0x05}
		nalu = append(nalu, seiUserDataUUID1[:]...)
		nalu = append(nalu, payload...)
		nalu = append(nalu, 0x80)

		msgs, err := h264.SEIUserDataUnregisteredExtract(nalu)
		require.NoError(t, err)
		require.Equal(t, []h264.SEIUserDataUnregistered{{
			UUID:    seiUserDataUUID1,
			Payload: payload,
		}}, msgs)
	})
}

func TestSEIUserDataUnregisteredExtractErrors(t *testing.T) {
	for _, ca := range []struct {
		name string
		nalu []byte
		err  string
	}{
		{
			"non-SEI NALU",
			[]byte{0x67, 0x42, 0x00, 0x1f},
			"expected a SEI NALU, got a SPS one",
		},
		{
			"empty NALU",
			[]byte{},
			"NALU too short",
		},
		{
			"only NALU header",
			[]byte{0x06},
			"NALU too short",
		},
		{
			"missing RBSP trailing bits",
			[]byte{0x06, 0x06, 0x01, 0xc4},
			"invalid RBSP trailing bits",
		},
		{
			"invalid RBSP trailing bits",
			[]byte{0x06, 0x06, 0x01, 0xc4, 0x40},
			"invalid RBSP trailing bits",
		},
		{
			"truncated payload type",
			[]byte{0x06, 0x01, 0x00, 0xff, 0x80},
			"invalid SEI payload type",
		},
		{
			"truncated payload size",
			[]byte{0x06, 0x01, 0xff, 0x80},
			"invalid SEI payload size",
		},
		{
			"declared size exceeds data",
			[]byte{0x06, 0x05, 0x14, 0x01, 0x02, 0x80},
			"SEI payload size 20 exceeds remaining data (2 bytes)",
		},
		{
			"payload too short for UUID",
			[]byte{0x06, 0x05, 0x05, 0x01, 0x02, 0x03, 0x04, 0x05, 0x80},
			"user_data_unregistered payload is 5 bytes long, cannot contain a UUID",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			_, err := h264.SEIUserDataUnregisteredExtract(ca.nalu)
			require.EqualError(t, err, ca.err)
		})
	}
}

func TestSEIUserDataUnregisteredMarshal(t *testing.T) {
	t.Run("sample", func(t *testing.T) {
		nalu, err := h264.SEIUserDataUnregisteredMarshal(seiUserDataSampleMessages)
		require.NoError(t, err)
		require.Equal(t, seiUserDataMarshaledSample, nalu)
	})

	t.Run("round trip", func(t *testing.T) {
		nalu, err := h264.SEIUserDataUnregisteredMarshal(seiUserDataSampleMessages)
		require.NoError(t, err)

		msgs, err := h264.SEIUserDataUnregisteredExtract(nalu)
		require.NoError(t, err)
		require.Equal(t, seiUserDataSampleMessages, msgs)
	})

	t.Run("extended payload size", func(t *testing.T) {
		payload := bytes.Repeat([]byte{0x42}, 300)
		nalu, err := h264.SEIUserDataUnregisteredMarshal([]h264.SEIUserDataUnregistered{{
			UUID:    seiUserDataUUID1,
			Payload: payload,
		}})
		require.NoError(t, err)
		require.Equal(t, []byte{0x06, 0x05, 0xff, 0x3d}, nalu[:4])

		msgs, err := h264.SEIUserDataUnregisteredExtract(nalu)
		require.NoError(t, err)
		require.Equal(t, []h264.SEIUserDataUnregistered{{
			UUID:    seiUserDataUUID1,
			Payload: payload,
		}}, msgs)
	})

	t.Run("oversized payload", func(t *testing.T) {
		_, err := h264.SEIUserDataUnregisteredMarshal([]h264.SEIUserDataUnregistered{{
			UUID:    seiUserDataUUID1,
			Payload: make([]byte, h264.SEIUserDataUnregisteredMaxPayloadSize+1),
		}})
		require.EqualError(t, err, "payload size 1048577 exceeds maximum 1048576")
	})
}

func TestSEIUserDataUnregisteredReplace(t *testing.T) {
	t.Run("sample is preserved", func(t *testing.T) {
		msgs, err := h264.SEIUserDataUnregisteredExtract(seiUserDataSample)
		require.NoError(t, err)

		nalu, err := h264.SEIUserDataUnregisteredReplace(seiUserDataSample, msgs)
		require.NoError(t, err)
		require.Equal(t, seiUserDataSample, nalu)
	})

	t.Run("remove all", func(t *testing.T) {
		nalu, err := h264.SEIUserDataUnregisteredReplace(seiUserDataSample, nil)
		require.NoError(t, err)
		require.Equal(t, []byte{0x06, 0x06, 0x01, 0xc4, 0x80}, nalu)
	})

	t.Run("replace payload", func(t *testing.T) {
		newMsgs := []h264.SEIUserDataUnregistered{{
			UUID:    seiUserDataUUID2,
			Payload: []byte("recorder-B"),
		}}

		nalu, err := h264.SEIUserDataUnregisteredReplace(seiUserDataSample, newMsgs)
		require.NoError(t, err)

		// the recovery point message is preserved
		require.Equal(t, []byte{0x06, 0x06, 0x01, 0xc4}, nalu[:4])
		require.Equal(t, byte(0x80), nalu[len(nalu)-1])

		msgs, err := h264.SEIUserDataUnregisteredExtract(nalu)
		require.NoError(t, err)
		require.Equal(t, newMsgs, msgs)
	})

	t.Run("NALU header is preserved", func(t *testing.T) {
		header := append([]byte{0x66}, seiUserDataSample[1:]...)

		nalu, err := h264.SEIUserDataUnregisteredReplace(header, nil)
		require.NoError(t, err)
		require.Equal(t, []byte{0x66, 0x06, 0x01, 0xc4, 0x80}, nalu)
	})

	t.Run("non-SEI NALU", func(t *testing.T) {
		_, err := h264.SEIUserDataUnregisteredReplace([]byte{0x65, 0x88, 0x84}, nil)
		require.EqualError(t, err, "expected a SEI NALU, got a IDR one")
	})

	t.Run("oversized payload", func(t *testing.T) {
		_, err := h264.SEIUserDataUnregisteredReplace(seiUserDataSample,
			[]h264.SEIUserDataUnregistered{{
				UUID:    seiUserDataUUID1,
				Payload: make([]byte, h264.SEIUserDataUnregisteredMaxPayloadSize+1),
			}})
		require.EqualError(t, err, "payload size 1048577 exceeds maximum 1048576")
	})
}

func FuzzSEIUserDataUnregisteredExtract(f *testing.F) {
	f.Add(seiUserDataSample)
	f.Add([]byte{0x06, 0x06, 0x01, 0xc4, 0x80})
	f.Add([]byte{0x06, 0xff, 0xff, 0xff, 0x80})

	f.Fuzz(func(_ *testing.T, nalu []byte) {
		msgs, err := h264.SEIUserDataUnregisteredExtract(nalu)
		if err != nil {
			return
		}

		// a valid NALU must round-trip
		out, err2 := h264.SEIUserDataUnregisteredReplace(nalu, msgs)
		if err2 != nil {
			panic(err2)
		}

		msgs2, err2 := h264.SEIUserDataUnregisteredExtract(out)
		if err2 != nil {
			panic(err2)
		}

		if len(msgs) == 0 && len(msgs2) == 0 {
			return
		}

		ok := true
		if len(msgs) != len(msgs2) {
			ok = false
		} else {
			for i := range msgs {
				if msgs[i].UUID != msgs2[i].UUID ||
					!bytes.Equal(msgs[i].Payload, msgs2[i].Payload) {
					ok = false
					break
				}
			}
		}

		if !ok {
			panic("round trip mismatch")
		}
	})
}
