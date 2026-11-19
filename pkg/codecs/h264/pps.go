package h264

import (
	"fmt"

	"github.com/bluenviron/mediacommon/v2/pkg/bits"
)

// PPS is a H264 picture parameter set.
// Specification: ITU-T Rec. H.264, 7.3.2.2
type PPS struct {
	ID    uint32
	SPSID uint32
}

// Unmarshal decodes a PPS.
func (p *PPS) Unmarshal(buf []byte) error {
	if len(buf) < 1 {
		return fmt.Errorf("not enough bits")
	}

	if NALUType(buf[0]&0x1F) != NALUTypePPS {
		return fmt.Errorf("not a PPS")
	}

	buf = EmulationPreventionRemove(buf[1:])
	pos := 0

	var err error
	p.ID, err = bits.ReadGolombUnsigned(buf, &pos)
	if err != nil {
		return err
	}

	p.SPSID, err = bits.ReadGolombUnsigned(buf, &pos)
	if err != nil {
		return err
	}

	if p.ID > 254 {
		return fmt.Errorf("pps_id out of range (%d)", p.ID)
	}

	if p.SPSID > 31 {
		return fmt.Errorf("sps_id out of range (%d)", p.SPSID)
	}

	return nil
}
