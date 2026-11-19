package vp9_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/vp9"
)

var keyFrame = []byte{
	0x82, 0x49, 0x83, 0x42, 0x00, 0x77, 0xf0, 0x32,
	0x34, 0x30, 0x38, 0x24, 0x1c, 0x19, 0x40, 0x18,
	0x03, 0x40, 0x5f, 0xb4,
}

var nonKeyFrame = []byte{0x86}

var casesSuperFrame = []struct {
	name string
	enc  []byte
	dec  vp9.SuperFrame
}{
	{
		"single frame, no index",
		append([]byte{}, keyFrame...),
		vp9.SuperFrame{keyFrame},
	},
	{
		"three frames",
		[]byte{
			0x82, 0x49, 0x83, // frame 0 (3 bytes)
			0xAA, 0xBB, 0xCC, 0xDD, // frame 1 (4 bytes)
			0x01, 0x02, // frame 2 (2 bytes)
			0xC2,             // marker: width=1, 3 frames
			0x03, 0x04, 0x02, // frame sizes, big endian
			0xC2, // mirrored marker
		},
		vp9.SuperFrame{
			{0x82, 0x49, 0x83},
			{0xAA, 0xBB, 0xCC, 0xDD},
			{0x01, 0x02},
		},
	},
}

func TestSuperFrameUnmarshal(t *testing.T) {
	for _, ca := range casesSuperFrame {
		t.Run(ca.name, func(t *testing.T) {
			var dec vp9.SuperFrame
			err := dec.Unmarshal(ca.enc)
			require.NoError(t, err)
			require.Equal(t, ca.dec, dec)
		})
	}
}

func TestSuperFrameUnmarshalBoundaries(t *testing.T) {
	// frames returned by Unmarshal must be sub-slices at the exact boundaries
	buf := []byte{
		0x82, 0x49, 0x83,
		0xAA, 0xBB, 0xCC, 0xDD,
		0x01, 0x02,
		0xC2, 0x03, 0x04, 0x02, 0xC2,
	}

	var dec vp9.SuperFrame
	err := dec.Unmarshal(buf)
	require.NoError(t, err)
	require.Len(t, dec, 3)
	require.Equal(t, buf[0:3], dec[0])
	require.Equal(t, buf[3:7], dec[1])
	require.Equal(t, buf[7:9], dec[2])
}

func TestSuperFrameMarshal(t *testing.T) {
	for _, ca := range casesSuperFrame {
		t.Run(ca.name, func(t *testing.T) {
			enc, err := ca.dec.Marshal()
			require.NoError(t, err)
			require.Equal(t, ca.enc, enc)
		})
	}
}

func TestSuperFrameRoundTrip(t *testing.T) {
	t.Run("single frame bytes are returned untouched", func(t *testing.T) {
		sf := vp9.SuperFrame{keyFrame}
		enc, err := sf.Marshal()
		require.NoError(t, err)
		require.Equal(t, keyFrame, enc)

		var dec vp9.SuperFrame
		err = dec.Unmarshal(enc)
		require.NoError(t, err)
		require.Equal(t, sf, dec)
	})

	t.Run("three frames", func(t *testing.T) {
		sf := vp9.SuperFrame{
			{0x82, 0x49, 0x83},
			{0xAA, 0xBB, 0xCC, 0xDD},
			{0x01, 0x02},
		}

		enc, err := sf.Marshal()
		require.NoError(t, err)

		var dec vp9.SuperFrame
		err = dec.Unmarshal(enc)
		require.NoError(t, err)
		require.Equal(t, sf, dec)

		// marshalling again must produce identical bytes
		enc2, err := dec.Marshal()
		require.NoError(t, err)
		require.Equal(t, enc, enc2)
	})

	t.Run("two byte frame sizes", func(t *testing.T) {
		bigFrame := make([]byte, 256)
		bigFrame[0] = 0x82

		sf := vp9.SuperFrame{bigFrame, {0x01}}

		enc, err := sf.Marshal()
		require.NoError(t, err)

		// marker must encode width = 2 bytes and 2 frames: 0xC0 | (1<<4) | 1
		marker := byte(0xD1)
		require.Equal(t, marker, enc[257])
		require.Equal(t, marker, enc[len(enc)-1])
		require.Equal(t, []byte{0x01, 0x00, 0x00, 0x01}, enc[258:262])

		var dec vp9.SuperFrame
		err = dec.Unmarshal(enc)
		require.NoError(t, err)
		require.Equal(t, sf, dec)
	})
}

func TestSuperFrameIsRandomAccess(t *testing.T) {
	t.Run("single key frame", func(t *testing.T) {
		var sf vp9.SuperFrame
		err := sf.Unmarshal(keyFrame)
		require.NoError(t, err)
		require.True(t, sf.IsRandomAccess())
	})

	t.Run("all key frames", func(t *testing.T) {
		sf := vp9.SuperFrame{keyFrame, keyFrame}
		enc, err := sf.Marshal()
		require.NoError(t, err)

		var dec vp9.SuperFrame
		err = dec.Unmarshal(enc)
		require.NoError(t, err)
		require.True(t, dec.IsRandomAccess())
	})

	t.Run("one non key frame", func(t *testing.T) {
		sf := vp9.SuperFrame{keyFrame, nonKeyFrame}
		enc, err := sf.Marshal()
		require.NoError(t, err)

		var dec vp9.SuperFrame
		err = dec.Unmarshal(enc)
		require.NoError(t, err)
		require.False(t, dec.IsRandomAccess())
	})
}

func TestSuperFrameUnmarshalErrors(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		var sf vp9.SuperFrame
		err := sf.Unmarshal([]byte{})
		require.Equal(t, vp9.ErrEmptySuperFrame, err)
	})

	t.Run("marker mismatch", func(t *testing.T) {
		buf := []byte{
			0x82, 0x49, 0x83,
			0xC3, 0x03, 0x04, 0x02, 0xC2, // first marker differs from last
		}
		var sf vp9.SuperFrame
		err := sf.Unmarshal(buf)
		require.EqualError(t, err, "superframe index marker mismatch: first byte 0xc3, last byte 0xc2")
	})

	t.Run("truncated index", func(t *testing.T) {
		buf := []byte{
			0x82, 0x49, 0xC2, // index needs 5 bytes but buffer only has 3
		}
		var sf vp9.SuperFrame
		err := sf.Unmarshal(buf)
		require.ErrorContains(t, err, "truncated superframe index")
	})

	t.Run("lengths exceed available data", func(t *testing.T) {
		buf := []byte{
			0x82, 0x49, 0x83, // only 3 bytes of data
			0xC2, 0x05, 0x04, 0x02, // sizes sum to 11
			0xC2,
		}
		var sf vp9.SuperFrame
		err := sf.Unmarshal(buf)
		require.ErrorContains(t, err, "exceeds available data")
	})

	t.Run("lengths don't cover all data", func(t *testing.T) {
		buf := []byte{
			0x82, 0x49, 0x83, 0xAA, // 4 bytes of data
			0xC2, 0x01, 0x01, 0x01, // sizes sum to 3
			0xC2,
		}
		var sf vp9.SuperFrame
		err := sf.Unmarshal(buf)
		require.ErrorContains(t, err, "doesn't match available data")
	})

	t.Run("empty frame inside index", func(t *testing.T) {
		buf := []byte{
			0x82, 0x49,
			0xC1, 0x02, 0x00, // marker: width=1, 2 frames; second size is 0
			0xC1,
		}
		var sf vp9.SuperFrame
		err := sf.Unmarshal(buf)
		require.EqualError(t, err, "superframe contains an empty frame")
	})

	t.Run("single frame exceeding max size", func(t *testing.T) {
		buf := make([]byte, vp9.MaxFrameSize+1)
		buf[len(buf)-1] = 0x00 // ensure last byte doesn't look like an index marker
		var sf vp9.SuperFrame
		err := sf.Unmarshal(buf)
		require.ErrorContains(t, err, "is too big")
	})

	t.Run("indexed frame exceeding max size", func(t *testing.T) {
		// marker with width = 4 bytes, one frame whose size exceeds the limit
		buf := []byte{
			0xF0,
			0x00, 0x20, 0x00, 0x01, // 2 MiB + 1
			0xF0,
		}
		var sf vp9.SuperFrame
		err := sf.Unmarshal(buf)
		require.ErrorContains(t, err, "is too big")
	})
}

func TestSuperFrameMarshalErrors(t *testing.T) {
	t.Run("empty superframe", func(t *testing.T) {
		var sf vp9.SuperFrame
		_, err := sf.Marshal()
		require.Equal(t, vp9.ErrEmptySuperFrame, err)
	})

	t.Run("too many frames", func(t *testing.T) {
		sf := make(vp9.SuperFrame, 9)
		for i := range sf {
			sf[i] = []byte{byte(i + 1)}
		}
		_, err := sf.Marshal()
		require.EqualError(t, err, "frame count (9) exceeds maximum allowed (8)")
	})

	t.Run("empty frame", func(t *testing.T) {
		sf := vp9.SuperFrame{{0x01}, {}}
		_, err := sf.Marshal()
		require.EqualError(t, err, "superframe contains an empty frame")
	})

	t.Run("frame exceeding max size", func(t *testing.T) {
		sf := vp9.SuperFrame{make([]byte, vp9.MaxFrameSize+1), {0x01}}
		_, err := sf.Marshal()
		require.ErrorContains(t, err, "is too big")
	})
}

func FuzzSuperFrameUnmarshal(f *testing.F) {
	for _, ca := range casesSuperFrame {
		f.Add(ca.enc)
	}
	f.Add(keyFrame)
	f.Add(nonKeyFrame)

	f.Fuzz(func(t *testing.T, b []byte) {
		var sf vp9.SuperFrame
		err := sf.Unmarshal(b)
		if err != nil {
			return
		}

		require.NotEmpty(t, sf)
		for _, frame := range sf {
			require.NotEmpty(t, frame)
		}

		enc, err := sf.Marshal()
		require.NoError(t, err)

		var sf2 vp9.SuperFrame
		err = sf2.Unmarshal(enc)
		require.NoError(t, err)
		require.Equal(t, sf, sf2)
	})
}
