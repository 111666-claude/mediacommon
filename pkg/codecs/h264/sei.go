package h264

import (
	"fmt"

	"github.com/bluenviron/mediacommon/v2/pkg/bits"
)

// SEIPayloadTypeRecoveryPoint is the payload type of the recovery point SEI
// message, as defined in ITU-T Rec. H.264, table D-1.
const SEIPayloadTypeRecoveryPoint = 6

// hasRecoveryPointSEI reports whether the NAL unit (whose type must be
// NALUTypeSEI) contains a recovery point SEI message.
//
// A recovery point SEI message marks an access unit from which the decoding
// process can be started without visible corruption, even though the first
// decoded picture is not an IDR picture.
//
// An error is returned only when the NAL unit claims to be an SEI but its
// payload cannot be parsed, so that a corrupted NAL unit is not mistaken for
// either a valid recovery point or an ordinary SEI message.
// Specification: ITU-T Rec. H.264, section 7.3.2.3.
func hasRecoveryPointSEI(nalu []byte) (bool, error) {
	buf := EmulationPreventionRemove(nalu[1:])
	pos := 0

	for pos < len(buf) {
		payloadType := 0
		for {
			if pos >= len(buf) {
				return false, fmt.Errorf("invalid SEI: truncated payload type")
			}
			b := buf[pos]
			pos++
			payloadType += int(b)
			if b != 0xFF {
				break
			}
		}

		payloadSize := 0
		for {
			if pos >= len(buf) {
				return false, fmt.Errorf("invalid SEI: truncated payload size")
			}
			b := buf[pos]
			pos++
			payloadSize += int(b)
			if b != 0xFF {
				break
			}
		}

		if payloadSize > len(buf)-pos {
			return false, fmt.Errorf("invalid SEI: payload exceeds NAL unit size")
		}

		if payloadType != SEIPayloadTypeRecoveryPoint {
			pos += payloadSize
			continue
		}

		if payloadSize < 1 {
			return false, fmt.Errorf("invalid SEI: empty recovery point payload")
		}

		// recovery_frame_cnt
		if _, err := bits.ReadGolombUnsigned(buf, &pos); err != nil {
			return false, fmt.Errorf("invalid SEI: %w", err)
		}

		// exact_match_flag, broken_link_flag, changing_slice_group_idc
		if err := bits.HasSpace(buf, pos, 6); err != nil {
			return false, fmt.Errorf("invalid SEI: %w", err)
		}

		return true, nil
	}

	return false, nil
}
