package h264

import (
	"fmt"

	"github.com/bluenviron/mediacommon/v2/pkg/bits"
)

// SEI payload types.
// Specification: ITU-T Rec. H.264, Table D-1
const (
	seiPayloadTypeRecoveryPoint = 6
)

// SEI is a H264 supplement enhancement information NALU.
// Specification: ITU-T Rec. H.264, 7.3.2.3
type SEI struct {
	// RecoveryPoint is set when the NALU contains a
	// recovery point SEI message (payload type 6).
	RecoveryPoint bool
}

// Unmarshal decodes an SEI from bytes.
// Only the presence of a recovery point message is retained; other
// payload types are parsed and skipped.
func (s *SEI) Unmarshal(nalu []byte) error {
	if len(nalu) == 0 {
		return fmt.Errorf("not enough bits")
	}

	if NALUType(nalu[0]&0x1F) != NALUTypeSEI {
		return fmt.Errorf("not an SEI")
	}

	buf := EmulationPreventionRemove(nalu[1:])
	pos := 0
	first := true

	for pos < len(buf)*8 {
		// alignment_zero_bits and rbsp_trailing_bits can only appear
		// after at least one payload; a leading zero byte is a valid
		// payloadType (buffering period, type 0).
		if !first && (len(buf)*8-pos)%8 == 0 {
			b := buf[pos/8]
			if b == 0 {
				pos += 8
				continue
			}
			if b&0x80 != 0 {
				// rbsp_trailing_bits
				return nil
			}
		}

		next, recovery, err := s.parsePayload(buf, pos)
		if err != nil {
			return err
		}
		if recovery {
			s.RecoveryPoint = true
		}
		first = false

		if next <= pos {
			// trailing bytes after a complete payload (alignment,
			// rbsp_trailing_bits or non-conformant padding) are ignored
			return nil
		}
		pos = next
	}

	return nil
}

// parsePayload parses a single SEI payload and returns the position of
// the first byte after it. A non-positive position signals that no
// complete payload could be parsed and the caller must stop consuming
// the NALU.
func (s *SEI) parsePayload(buf []byte, pos int) (int, bool, error) {
	if err := bits.HasSpace(buf, pos, 16); err != nil {
		// not enough room for a complete payloadType/payloadSize
		// header pair: this is trailing data, not a payload.
		return 0, false, nil
	}

	// payloadType
	payloadType := 0
	for {
		if err := bits.HasSpace(buf, pos, 8); err != nil {
			return 0, false, nil
		}
		b := byte(bits.ReadBitsUnsafe(buf, &pos, 8))
		payloadType += int(b)
		if b != 0xFF {
			break
		}
	}

	// payloadSize
	payloadSize := 0
	for {
		if err := bits.HasSpace(buf, pos, 8); err != nil {
			return 0, false, nil
		}
		b := byte(bits.ReadBitsUnsafe(buf, &pos, 8))
		payloadSize += int(b)
		if b != 0xFF {
			break
		}
	}

	payloadStart := pos
	payloadEnd := payloadStart + payloadSize*8
	if err := bits.HasSpace(buf, pos, payloadSize*8); err != nil {
		if payloadType == seiPayloadTypeRecoveryPoint {
			// a truncated recovery point would leave the random
			// access state undefined and cannot be tolerated.
			return 0, false, fmt.Errorf("invalid SEI payload size")
		}
		return 0, false, nil
	}

	if payloadType == seiPayloadTypeRecoveryPoint {
		// recovery_frame_cnt
		_, err := bits.ReadGolombUnsigned(buf, &pos)
		if err != nil {
			return 0, false, err
		}

		// exact_match_flag
		_, err = bits.ReadBits(buf, &pos, 1)
		if err != nil {
			return 0, false, err
		}

		// broken_link_flag
		_, err = bits.ReadBits(buf, &pos, 1)
		if err != nil {
			return 0, false, err
		}

		// changing_slice_group_idc
		_, err = bits.ReadBits(buf, &pos, 4)
		if err != nil {
			return 0, false, err
		}

		return payloadEnd, true, nil
	}

	return payloadEnd, false, nil
}
