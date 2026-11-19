package h265

import (
	"fmt"

	"github.com/bluenviron/mediacommon/v2/pkg/bits"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
)

// VPS is a H265 video parameter set.
// Specification: ITU-T Rec. H.265, 7.3.2.1
type VPS struct {
	ID uint8
}

// Unmarshal decodes a VPS. Only the fields needed to identify the
// parameter set are decoded.
func (v *VPS) Unmarshal(buf []byte) error {
	if len(buf) < 2 {
		return fmt.Errorf("not enough bits")
	}

	if NALUType((buf[0]>>1)&0b111111) != NALUType_VPS_NUT {
		return fmt.Errorf("not a VPS")
	}

	buf = h264.EmulationPreventionRemove(buf[1:])
	pos := 8

	err := bits.HasSpace(buf, pos, 12)
	if err != nil {
		return err
	}

	v.ID = uint8(bits.ReadBitsUnsafe(buf, &pos, 4))

	return nil
}
