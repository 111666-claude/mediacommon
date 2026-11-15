package h264

import (
	"errors"
	"fmt"
)

// ErrAnnexBNoNALUs is returned by AnnexBUnmarshal when no NALUs have been decoded.
var ErrAnnexBNoNALUs = errors.New("Annex-B unit doesn't contain any NALU")

// ErrAnnexBNoInitialDelimiter is returned by AnnexBUnmarshal when the initial delimiter is not found.
var ErrAnnexBNoInitialDelimiter = errors.New("initial delimiter not found")

// AnnexBUnmarshal decodes an access unit from the Annex-B stream format.
// Specification: ITU-T Rec. H.264, Annex B
//
// The pseudo logic from ITU-T Rec. H.264, Annex B:
//
//	byte_stream_nal_unit(NumBytesInNALunit) {
//	    while (next_bits(24) != 0x000001 &&
//	           next_bits(32) != 0x00000001)
//	        leading_zero_8bits /* equal to 0x00 */
//
//	    if (next_bits(24) != 0x000001)
//	        zero_byte /* equal to 0x00 */
//
//	    if (more_data_in_byte_stream()) {
//	        start_code_prefix_one_3bytes /* equal to 0x000001 */
//	        nal_unit(NumBytesInNALunit)
//	    }
//
//	    while (more_data_in_byte_stream() &&
//	           next_bits(24) != 0x000001 &&
//	           next_bits(32) != 0x00000001)
//	        trailing_zero_8bits /* equal to 0x00 */
//	}
func AnnexBUnmarshal(buf []byte) ([][]byte, error) {
	count, err := annexBCountNALUs(buf)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, ErrAnnexBNoNALUs
	}
	if !annexBHasInitialDelimiter(buf) {
		return nil, ErrAnnexBNoInitialDelimiter
	}

	au := make([][]byte, 0, count)
	i := 0
	start := 0

	for i < len(buf) {
		lim := 4
		if lim > len(buf)-i {
			lim = len(buf) - i
		}
		data := buf[i : i+lim]

		switch {
		case len(data) >= 3 && data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01:
			// NALU with a 3-byte start code prefix.
			if i > start {
				au = append(au, buf[start:i])
			}
			i += 3
			start = i

		case len(data) >= 4 && data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x00 && data[3] == 0x01:
			// NALU with a 4-byte start code prefix.
			// The last byte of a NALU is kept even when it is 0x00: the first zero
			// of the run is part of the NALU, while the remaining ones belong to the
			// start code prefix.
			if i > start {
				au = append(au, buf[start:i])
			}
			i += 4
			start = i

		default:
			i++
		}
	}

	if i > start {
		au = append(au, buf[start:i])
	}

	return au, nil
}

// annexBCountNALUs counts the NALUs contained in an Annex-B stream and checks
// that their total size does not exceed MaxAccessUnitSize.
func annexBCountNALUs(buf []byte) (int, error) {
	n := 0
	i := 0
	start := 0
	auSize := 0

	for i < len(buf) {
		lim := 4
		if lim > len(buf)-i {
			lim = len(buf) - i
		}
		data := buf[i : i+lim]

		switch {
		case len(data) >= 3 && data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01:
			if i > start {
				auSize += i - start
				if auSize > MaxAccessUnitSize {
					return 0, fmt.Errorf("access unit size (%d) is too big, maximum is %d",
						auSize, MaxAccessUnitSize)
				}
				n++
			}
			i += 3
			start = i

		case len(data) >= 4 && data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x00 && data[3] == 0x01:
			if i > start {
				auSize += i - start
				if auSize > MaxAccessUnitSize {
					return 0, fmt.Errorf("access unit size (%d) is too big, maximum is %d",
						auSize, MaxAccessUnitSize)
				}
				n++
			}
			i += 4
			start = i

		default:
			i++
		}
	}

	if i > start {
		if (auSize + i - start) > MaxAccessUnitSize {
			return 0, fmt.Errorf("access unit size (%d) is too big, maximum is %d",
				auSize+i-start, MaxAccessUnitSize)
		}
		n++
	}

	return n, nil
}

func annexBHasInitialDelimiter(buf []byte) bool {
	switch {
	case len(buf) >= 4 && buf[0] == 0x00 && buf[1] == 0x00 && buf[2] == 0x00 && buf[3] == 0x01:
		return true

	case len(buf) >= 3 && buf[0] == 0x00 && buf[1] == 0x00 && buf[2] == 0x01:
		return true
	}

	return false
}

func annexBMarshalSize(au [][]byte) int {
	n := 0
	for _, nalu := range au {
		n += 4 + len(nalu)
	}
	return n
}

// AnnexBMarshal encodes an access unit into the Annex-B stream format.
// Specification: ITU-T Rec. H.264, Annex B
func AnnexBMarshal(au [][]byte) ([]byte, error) {
	buf := make([]byte, annexBMarshalSize(au))
	pos := 0

	for _, nalu := range au {
		pos += copy(buf[pos:], []byte{0x00, 0x00, 0x00, 0x01})
		pos += copy(buf[pos:], nalu)
	}

	return buf, nil
}
