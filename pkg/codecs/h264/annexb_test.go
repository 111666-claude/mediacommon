package h264

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var casesAnnexB = []struct {
	name   string
	encin  []byte
	encout []byte
	dec    [][]byte
}{
	{
		"2 zeros",
		[]byte{
			0x00, 0x00, 0x01, 0xaa, 0xbb, 0x00, 0x00, 0x01,
			0xcc, 0xdd, 0x00, 0x00, 0x01, 0xee, 0xff,
		},
		[]byte{
			0x00, 0x00, 0x00, 0x01, 0xaa, 0xbb,
			0x00, 0x00, 0x00, 0x01, 0xcc, 0xdd,
			0x00, 0x00, 0x00, 0x01, 0xee, 0xff,
		},
		[][]byte{
			{0xaa, 0xbb},
			{0xcc, 0xdd},
			{0xee, 0xff},
		},
	},
	{
		"3 zeros",
		[]byte{
			0x00, 0x00, 0x00, 0x01, 0xaa, 0xbb,
			0x00, 0x00, 0x00, 0x01, 0xcc, 0xdd,
			0x00, 0x00, 0x00, 0x01, 0xee, 0xff,
		},
		[]byte{
			0x00, 0x00, 0x00, 0x01, 0xaa, 0xbb,
			0x00, 0x00, 0x00, 0x01, 0xcc, 0xdd,
			0x00, 0x00, 0x00, 0x01, 0xee, 0xff,
		},
		[][]byte{
			{0xaa, 0xbb},
			{0xcc, 0xdd},
			{0xee, 0xff},
		},
	},
	{
		// used by Apple inside HLS test streams
		"2 or 3 zeros",
		[]byte{
			0, 0, 0, 1, 9, 240,
			0, 0, 0, 1, 39, 66, 224, 21, 169, 24, 60, 23, 252, 184, 3, 80, 96, 16, 107, 108, 43, 94, 247, 192, 64,
			0, 0, 0, 1, 40, 222, 9, 200,
			0, 0, 1, 6, 0, 7, 131, 236, 119,
			0, 0, 0, 0, 1, 3, 0, 64, 128,
			0, 0, 1, 6, 5, 17, 3, 135, 244, 78, 205, 10, 75, 220, 161, 148, 58, 195, 212, 155, 23, 31, 0, 128,
		},
		[]byte{
			0, 0, 0, 1, 9, 240,
			0, 0, 0, 1, 39, 66, 224, 21, 169, 24, 60, 23, 252, 184, 3, 80, 96, 16, 107, 108, 43, 94, 247, 192, 64,
			0, 0, 0, 1, 40, 222, 9, 200,
			0, 0, 0, 1, 6, 0, 7, 131, 236, 119, 0,
			0, 0, 0, 1, 3, 0, 64, 128,
			0, 0, 0, 1, 6, 5, 17, 3, 135, 244, 78, 205, 10, 75, 220, 161, 148, 58, 195, 212, 155, 23, 31, 0, 128,
		},
		[][]byte{
			{9, 240},
			{39, 66, 224, 21, 169, 24, 60, 23, 252, 184, 3, 80, 96, 16, 107, 108, 43, 94, 247, 192, 64},
			{40, 222, 9, 200},
			{6, 0, 7, 131, 236, 119, 0},
			{3, 0, 64, 128},
			{6, 5, 17, 3, 135, 244, 78, 205, 10, 75, 220, 161, 148, 58, 195, 212, 155, 23, 31, 0, 128},
		},
	},
	{
		"AUs end with zeros",
		[]byte{
			0x00, 0x00, 0x00, 0x01, 0xaa, 0xbb, 0x00,
			0x00, 0x00, 0x00, 0x01, 0xcc, 0xdd, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x01, 0xee, 0xff, 0x00, 0x00,
			0x00, 0x00, 0x01, 0x1a, 0x1b, 0x1c,
		},
		[]byte{
			0x00, 0x00, 0x00, 0x01, 0xaa, 0xbb, 0x00,
			0x00, 0x00, 0x00, 0x01, 0xcc, 0xdd, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x01, 0xee, 0xff, 0x00,
			0x00, 0x00, 0x00, 0x01, 0x1a, 0x1b, 0x1c,
		},
		[][]byte{
			{0xaa, 0xbb, 0},
			{0xcc, 0xdd, 0, 0, 0},
			{0xee, 0xff, 0},
			{0x1a, 0x1b, 0x1c},
		},
	},
}

func TestAnnexBUnmarshal(t *testing.T) {
	for _, ca := range casesAnnexB {
		t.Run(ca.name, func(t *testing.T) {
			dec, err := AnnexBUnmarshal(ca.encin)
			require.NoError(t, err)
			require.Equal(t, ca.dec, dec)
		})
	}
}

func TestAnnexBUnmarshalEmpty(t *testing.T) {
	buf := []byte{0, 0, 0, 1, 0, 0, 0, 1}
	_, err := AnnexBUnmarshal(buf)
	require.Equal(t, err, ErrAnnexBNoNALUs)

	buf = []byte{0, 0, 0, 1, 0, 0, 0, 1, 1}
	dec, err := AnnexBUnmarshal(buf)
	require.NoError(t, err)
	require.Equal(t, [][]byte{{1}}, dec)
}

func TestAnnexBMarshal(t *testing.T) {
	for _, ca := range casesAnnexB {
		t.Run(ca.name, func(t *testing.T) {
			enc, err := AnnexBMarshal(ca.dec)
			require.NoError(t, err)
			require.Equal(t, ca.encout, enc)
		})
	}
}

// A NALU that ends with a 0 byte must not be merged with the following one,
// even when the access unit contains many of them.
func TestAnnexBTrailingZeroRoundTrip(t *testing.T) {
	au := make([][]byte, 1000)
	for i := range au {
		au[i] = []byte{byte(i + 1), byte(i), 0x00}
	}

	enc, err := AnnexBMarshal(au)
	require.NoError(t, err)
	require.Equal(t, 7000, len(enc))

	dec, err := AnnexBUnmarshal(enc)
	require.NoError(t, err)
	require.Equal(t, 1000, len(dec))

	for i := range au {
		require.Equal(t, au[i], dec[i])
	}

	// a second round trip must be identical too
	enc2, err := AnnexBMarshal(dec)
	require.NoError(t, err)
	require.Equal(t, enc, enc2)
}

func TestAnnexBUnmarshalMixedDelimiters(t *testing.T) {
	// buf contains 3-byte and 4-byte delimiters, consecutive units and
	// NALUs ending with runs of zero bytes.
	buf := []byte{
		0x00, 0x00, 0x01, 0xaa, 0x00, // 3-byte delimiter, NALU ending with 0 (ambiguous)
		0x00, 0x00, 0x00, 0x01, 0xbb, 0x00, 0x00, // 4-byte delimiter, NALU ending with 2 zeros
		0x00, 0x00, 0x00, 0x00, 0x01, 0xcc, // 5-byte delimiter
		0x00, 0x00, 0x00, 0x01, // empty boundary
		0x00, 0x00, 0x00, 0x01, 0xdd, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x01, 0xee, // 3-byte delimiter, no zero before it
		0x00, 0x00, 0x00, 0x01, 0xff, 0x00, // last NALU ends with 0
	}

	dec, err := AnnexBUnmarshal(buf)
	require.NoError(t, err)
	require.Equal(t, [][]byte{
		{0xaa, 0x00},
		{0xbb, 0x00, 0x00, 0x00},
		{0xcc},
		{0xdd, 0x00, 0x00, 0x00, 0x00},
		{0xee},
		{0xff, 0x00},
	}, dec)

	// re-encoding and decoding must not merge or split units
	enc, err := AnnexBMarshal(dec)
	require.NoError(t, err)
	dec2, err := AnnexBUnmarshal(enc)
	require.NoError(t, err)
	require.Equal(t, dec, dec2)
}

func TestAnnexBUnmarshalErrors(t *testing.T) {
	_, err := AnnexBUnmarshal([]byte{0xaa, 0xbb})
	require.Equal(t, ErrAnnexBNoInitialDelimiter, err)

	_, err = AnnexBUnmarshal([]byte{})
	require.Equal(t, ErrAnnexBNoNALUs, err)

	_, err = AnnexBUnmarshal([]byte{0x00, 0x00, 0x00, 0x01})
	require.Equal(t, ErrAnnexBNoNALUs, err)

	_, err = AnnexBUnmarshal([]byte{0x00, 0x00, 0x00, 0x01, 0xaa, 0xbb, 0x01, 0xcc})
	require.NoError(t, err)

	buf := make([]byte, 4+MaxAccessUnitSize+1)
	buf[3] = 0x01
	buf[4] = 0xaa
	_, err = AnnexBUnmarshal(buf)
	require.ErrorContains(t, err, "access unit size")
}

func BenchmarkAnnexBUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		AnnexBUnmarshal([]byte{ //nolint:errcheck
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
		})
	}
}

func FuzzAnnexBUnmarshal(f *testing.F) {
	for _, ca := range casesAnnexB {
		f.Add(ca.encin)
	}

	f.Fuzz(func(_ *testing.T, b []byte) {
		au, err := AnnexBUnmarshal(b)
		if err == nil {
			AnnexBMarshal(au) //nolint:errcheck
		}
	})
}
