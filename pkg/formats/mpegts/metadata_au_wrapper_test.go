package mpegts

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataAUWrapper(t *testing.T) {
	for _, ca := range []struct {
		name       string
		size       int
		cellCount  int
		indication []uint8
	}{
		{"empty", 0, 1, []uint8{metadataAUFragmentMiddle}},
		{"small", 10, 1, []uint8{metadataAUFragmentMiddle}},
		{"max single cell", 65535, 1, []uint8{metadataAUFragmentMiddle}},
		{"two fragments, remainder full", 65536, 2, []uint8{metadataAUFragmentFirst, metadataAUFragmentLast}},
		{"two fragments, remainder partial", 65600, 2, []uint8{metadataAUFragmentFirst, metadataAUFragmentLast}},
		{"two full fragments", 131070, 2, []uint8{metadataAUFragmentFirst, metadataAUFragmentLast}},
		{"two fragments and a remainder", 131071, 3, []uint8{
			metadataAUFragmentFirst, metadataAUFragmentMiddle, metadataAUFragmentLast,
		}},
		{"many fragments", 200000, 4, []uint8{
			metadataAUFragmentFirst, metadataAUFragmentMiddle, metadataAUFragmentMiddle, metadataAUFragmentLast,
		}},
	} {
		t.Run(ca.name, func(t *testing.T) {
			in := bytes.Repeat([]byte{0xAB}, ca.size)

			enc, err := writeMetadataAUWrapper(in)
			require.NoError(t, err)

			// inspect cells
			var indications []uint8
			rest := enc
			for len(rest) > 0 {
				var cell metadataAUCell
				n, err2 := cell.unmarshalFrom(rest)
				require.NoError(t, err2)
				require.Equal(t, uint8(len(indications)), cell.SequenceNumber)
				require.LessOrEqual(t, len(cell.AUCellData), metadataAUCellMaxDataSize)
				indications = append(indications, cell.CellFragmentIndication)
				rest = rest[n:]
			}
			require.Equal(t, ca.indication, indications)
			require.Len(t, indications, ca.cellCount)

			dec, err := readMetadataAUWrapper(enc)
			require.NoError(t, err)
			require.Equal(t, in, dec)
		})
	}
}

func TestMetadataAUWrapperCompatibility(t *testing.T) {
	// a single cell with indication 0 or 3 carries a complete message
	for _, indication := range []uint8{metadataAUFragmentMiddle, metadataAUFragmentNone} {
		buf, err := metadataAUCell{
			MetadataServiceID:      0,
			SequenceNumber:         0,
			CellFragmentIndication: indication,
			RandomAccessIndicator:  true,
			AUCellData:             []byte{1, 2, 3},
		}.marshal()
		require.NoError(t, err)

		dec, err := readMetadataAUWrapper(buf)
		require.NoError(t, err)
		require.Equal(t, []byte{1, 2, 3}, dec)
	}
}

func FuzzMetadataAUWrapper(f *testing.F) {
	f.Add([]byte{0, 0, 0x3f, 0, 3, 1, 2, 3})
	f.Add([]byte{0, 0, 0xbf, 0xff, 0xff})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, buf []byte) {
		dec, err := readMetadataAUWrapper(buf)
		if err != nil {
			return
		}

		enc, err := writeMetadataAUWrapper(dec)
		require.NoError(t, err)

		dec2, err := readMetadataAUWrapper(enc)
		require.NoError(t, err)
		require.Equal(t, dec, dec2)
	})
}
