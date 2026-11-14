package mpegts

import "fmt"

// Maximum amount of data that can be stored in a single metadata_AU_cell,
// since the length field of the cell is 16 bits wide.
const metadataAUCellMaxDataSize = 0xFFFF

// metadata_AU_cell fragment indications.
// Specification: ISO 13818-1, table 2-97
const (
	metadataAUFragmentMiddle uint8 = 0b00
	metadataAUFragmentLast   uint8 = 0b01
	metadataAUFragmentFirst  uint8 = 0b10
	metadataAUFragmentNone   uint8 = 0b11
)

// readMetadataAUWrapper decodes the metadata_AU_cells contained in a PES payload
// into the complete metadata AU they carry.
func readMetadataAUWrapper(in []byte) ([]byte, error) {
	out := make([]byte, 0, len(in))
	expectedSeqNum := uint8(0)

	for i := 0; ; i++ {
		if len(in) == 0 {
			if i == 0 {
				return nil, fmt.Errorf("buffer is too small")
			}
			return nil, fmt.Errorf("cell fragment is truncated")
		}

		var cell metadataAUCell
		n, err := cell.unmarshalFrom(in)
		if err != nil {
			return nil, err
		}
		in = in[n:]

		if cell.SequenceNumber != expectedSeqNum {
			return nil, fmt.Errorf("unexpected sequence number: %d, expected %d",
				cell.SequenceNumber, expectedSeqNum)
		}
		expectedSeqNum++

		out = append(out, cell.AUCellData...)

		if i == 0 {
			switch cell.CellFragmentIndication {
			case metadataAUFragmentNone:
				if len(in) != 0 {
					return nil, fmt.Errorf("unread bytes detected")
				}
				return out, nil

			case metadataAUFragmentFirst:
				// continue with the following fragments

			case metadataAUFragmentMiddle:
				// a single cell with indication 0 has always been used to carry
				// a complete message; keep accepting it for compatibility.

				if len(in) != 0 {
					return nil, fmt.Errorf("unread bytes detected")
				}
				return out, nil

			default:
				return nil, fmt.Errorf("unexpected cell_fragment_indication %v",
					cell.CellFragmentIndication)
			}
			continue
		}

		switch cell.CellFragmentIndication {
		case metadataAUFragmentMiddle:
			// continue with the following fragments

		case metadataAUFragmentLast:
			if len(in) != 0 {
				return nil, fmt.Errorf("unread bytes detected")
			}
			return out, nil

		default:
			return nil, fmt.Errorf("unexpected cell_fragment_indication %v",
				cell.CellFragmentIndication)
		}
	}
}

// writeMetadataAUWrapper encodes a metadata AU into one or more metadata_AU_cells,
// splitting it when it exceeds the capacity of a single cell.
func writeMetadataAUWrapper(in []byte) ([]byte, error) {
	cellCount := len(in) / metadataAUCellMaxDataSize
	if len(in)%metadataAUCellMaxDataSize != 0 {
		cellCount++
	}
	if cellCount == 0 {
		cellCount = 1
	}

	out := make([]byte, 0, len(in)+5*cellCount)

	n := 0
	for i := 0; i < cellCount; i++ {
		end := n + metadataAUCellMaxDataSize
		if end > len(in) {
			end = len(in)
		}

		var fragmentIndication uint8
		switch {
		case cellCount == 1:
			// keep using the historical indication for complete messages,
			// so that the output of small messages is unchanged.
			fragmentIndication = metadataAUFragmentMiddle

		case i == 0:
			fragmentIndication = metadataAUFragmentFirst

		case i == cellCount-1:
			fragmentIndication = metadataAUFragmentLast

		default:
			fragmentIndication = metadataAUFragmentMiddle
		}

		buf, err := metadataAUCell{
			MetadataServiceID:      0,
			SequenceNumber:         uint8(i),
			CellFragmentIndication: fragmentIndication,
			DecoderConfigFlag:      false,
			RandomAccessIndicator:  true,
			AUCellData:             in[n:end],
		}.marshal()
		if err != nil {
			return nil, err
		}
		out = append(out, buf...)

		n = end
	}

	return out, nil
}
