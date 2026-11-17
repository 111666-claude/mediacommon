package h264

import (
	"fmt"
)

const (
	seiPayloadTypeUserDataUnregistered = 5

	// SEIUserDataUnregisteredMaxPayloadSize is the maximum size of the
	// payload of a SEI user_data_unregistered message.
	SEIUserDataUnregisteredMaxPayloadSize = 1 << 20
)

// SEIUserDataUnregistered is a SEI user_data_unregistered message.
// Specification: ITU-T Rec. H.264, section D.1.6
type SEIUserDataUnregistered struct {
	UUID    [16]byte
	Payload []byte
}

type seiRawMessage struct {
	payloadType int
	payload     []byte
	raw         []byte
}

// seiParseMessages decodes the SEI messages contained in a SEI NALU.
// It returns the NALU header, the messages and the RBSP trailing bits byte.
func seiParseMessages(nalu []byte) (byte, []seiRawMessage, byte, error) {
	if len(nalu) < 2 {
		return 0, nil, 0, fmt.Errorf("NALU too short")
	}

	if NALUType(nalu[0]&0x1F) != NALUTypeSEI {
		return 0, nil, 0, fmt.Errorf("expected a SEI NALU, got a %s one", NALUType(nalu[0]&0x1F))
	}

	rbsp := EmulationPreventionRemove(nalu[1:])

	if len(rbsp) == 0 {
		return 0, nil, 0, fmt.Errorf("missing RBSP trailing bits")
	}

	// SEI messages are byte-aligned, therefore rbsp_trailing_bits
	// must consist of a single 0x80 byte.
	trailing := rbsp[len(rbsp)-1]
	if trailing != 0x80 {
		return 0, nil, 0, fmt.Errorf("invalid RBSP trailing bits")
	}

	buf := rbsp[:len(rbsp)-1]
	var msgs []seiRawMessage
	pos := 0

	for pos < len(buf) {
		start := pos

		payloadType, next, ok := readSEIByteValue(buf, pos)
		if !ok {
			return 0, nil, 0, fmt.Errorf("invalid SEI payload type")
		}
		pos = next

		payloadSize, next, ok := readSEIByteValue(buf, pos)
		if !ok {
			return 0, nil, 0, fmt.Errorf("invalid SEI payload size")
		}
		pos = next

		if payloadSize > len(buf)-pos {
			return 0, nil, 0, fmt.Errorf("SEI payload size %d exceeds remaining data (%d bytes)",
				payloadSize, len(buf)-pos)
		}

		msgs = append(msgs, seiRawMessage{
			payloadType: payloadType,
			payload:     buf[pos : pos+payloadSize],
			raw:         buf[start : pos+payloadSize],
		})
		pos += payloadSize
	}

	return nalu[0], msgs, trailing, nil
}

// appendSEIUserDataUnregistered encodes a user_data_unregistered message
// and appends it to a SEI RBSP.
func appendSEIUserDataUnregistered(rbsp []byte, m SEIUserDataUnregistered) ([]byte, error) {
	if len(m.Payload) > SEIUserDataUnregisteredMaxPayloadSize {
		return nil, fmt.Errorf("payload size %d exceeds maximum %d",
			len(m.Payload), SEIUserDataUnregisteredMaxPayloadSize)
	}

	rbsp = append(rbsp, seiPayloadTypeUserDataUnregistered)

	size := 16 + len(m.Payload)
	for size >= 0xFF {
		rbsp = append(rbsp, 0xFF)
		size -= 0xFF
	}
	rbsp = append(rbsp, byte(size))

	rbsp = append(rbsp, m.UUID[:]...)
	rbsp = append(rbsp, m.Payload...)

	return rbsp, nil
}

// SEIUserDataUnregisteredExtract extracts the user_data_unregistered messages
// contained in a SEI NALU.
func SEIUserDataUnregisteredExtract(nalu []byte) ([]SEIUserDataUnregistered, error) {
	_, msgs, _, err := seiParseMessages(nalu)
	if err != nil {
		return nil, err
	}

	var out []SEIUserDataUnregistered

	for _, m := range msgs {
		if m.payloadType != seiPayloadTypeUserDataUnregistered {
			continue
		}

		if len(m.payload) < 16 {
			return nil, fmt.Errorf("user_data_unregistered payload is %d bytes long, cannot contain a UUID",
				len(m.payload))
		}

		var mout SEIUserDataUnregistered
		copy(mout.UUID[:], m.payload[:16])
		mout.Payload = append([]byte(nil), m.payload[16:]...)
		out = append(out, mout)
	}

	return out, nil
}

// SEIUserDataUnregisteredMarshal encodes a SEI NALU that contains
// the given user_data_unregistered messages.
func SEIUserDataUnregisteredMarshal(msgs []SEIUserDataUnregistered) ([]byte, error) {
	var rbsp []byte

	for _, m := range msgs {
		var err error
		rbsp, err = appendSEIUserDataUnregistered(rbsp, m)
		if err != nil {
			return nil, err
		}
	}

	// rbsp_trailing_bits
	rbsp = append(rbsp, 0x80)

	out := make([]byte, 0, 1+len(rbsp))
	out = append(out, byte(NALUTypeSEI))
	out = append(out, EmulationPreventionAdd(rbsp)...)

	return out, nil
}

// SEIUserDataUnregisteredReplace rewrites a SEI NALU, replacing its
// user_data_unregistered messages with the given ones. Other SEI messages,
// the NALU header and the RBSP trailing bits are preserved.
func SEIUserDataUnregisteredReplace(nalu []byte, msgs []SEIUserDataUnregistered) ([]byte, error) {
	header, existing, trailing, err := seiParseMessages(nalu)
	if err != nil {
		return nil, err
	}

	var rbsp []byte

	for _, m := range existing {
		if m.payloadType != seiPayloadTypeUserDataUnregistered {
			rbsp = append(rbsp, m.raw...)
		}
	}

	for _, m := range msgs {
		var err error
		rbsp, err = appendSEIUserDataUnregistered(rbsp, m)
		if err != nil {
			return nil, err
		}
	}

	// rbsp_trailing_bits
	rbsp = append(rbsp, trailing)

	out := make([]byte, 0, 1+len(rbsp))
	out = append(out, header)
	out = append(out, EmulationPreventionAdd(rbsp)...)

	return out, nil
}
