package h264

import (
	"bytes"
	"fmt"

	"github.com/bluenviron/mediacommon/v2/pkg/bits"
)

const (
	maxReorderedFrames = 10
	/*
		(max_size(first_mb_in_slice) + max_size(slice_type) + max_size(pic_parameter_set_id) +
		max_size(frame_num) + max_size(pic_order_cnt_lsb)) * 4 / 3 =
		(3 * max_size(golomb) + (max(Log2MaxFrameNumMinus4) + 4) / 8 + (max(Log2MaxPicOrderCntLsbMinus4) + 4) / 8) * 4 / 3 =
		(3 * 4 + 2 + 2) * 4 / 3 = 22
	*/
	maxBytesToGetPOC = 22
)

func getPictureOrderCount(buf []byte, sps *SPS, idr bool) (uint32, error) {
	buf = buf[1:]
	lb := min(len(buf), maxBytesToGetPOC)

	buf = EmulationPreventionRemove(buf[:lb])
	pos := 0

	_, err := bits.ReadGolombUnsigned(buf, &pos) // first_mb_in_slice
	if err != nil {
		return 0, err
	}

	_, err = bits.ReadGolombUnsigned(buf, &pos) // slice_type
	if err != nil {
		return 0, err
	}

	_, err = bits.ReadGolombUnsigned(buf, &pos) // pic_parameter_set_id
	if err != nil {
		return 0, err
	}

	_, err = bits.ReadBits(buf, &pos, int(sps.Log2MaxFrameNumMinus4+4)) // frame_num
	if err != nil {
		return 0, err
	}

	if idr {
		_, err = bits.ReadGolombUnsigned(buf, &pos) // idr_pic_id
		if err != nil {
			return 0, err
		}
	}

	picOrderCntLsb, err := bits.ReadBits(buf, &pos, int(sps.Log2MaxPicOrderCntLsbMinus4+4))
	if err != nil {
		return 0, err
	}

	return uint32(picOrderCntLsb), nil
}

func pictureOrderCountDiff(a uint32, b uint32, sps *SPS) int32 {
	maxVal := uint32(1 << (sps.Log2MaxPicOrderCntLsbMinus4 + 4))
	d := (a - b) & (maxVal - 1)
	if d > (maxVal / 2) {
		return int32(d) - int32(maxVal)
	}
	return int32(d)
}

// DTSExtractor computes DTS from PTS.
type DTSExtractor struct {
	sps             []byte
	spsp            *SPS
	prevDTSFilled   bool
	prevDTS         int64
	randomReceived  bool
	imageSeen       bool
	expectedPOC     uint32
	reorderedFrames int
	pause           int
	auCount         int
	pocIncrement    int
	prevPOC         uint32
	prevPrevPOC     uint32
}

// Initialize initializes a DTSExtractor.
func (d *DTSExtractor) Initialize() {
	d.pocIncrement = 2
}

// NewDTSExtractor allocates a DTSExtractor.
//
// Deprecated: replaced by DTSExtractor.Initialize.
func NewDTSExtractor() *DTSExtractor {
	d := &DTSExtractor{}
	d.Initialize()
	return d
}

func (d *DTSExtractor) extractInner(au [][]byte, pts int64) (int64, bool, error) {
	var idr []byte
	var nonIDR []byte
	recoveryPoint := false

	// first pass: update parameter sets and detect a recovery point.
	// every NALU is inspected so that the ordering of SPS, SEI and
	// slice NALUs within the access unit cannot pollute the state.
	for _, nalu := range au {
		if len(nalu) == 0 {
			return 0, false, fmt.Errorf("empty NALU")
		}

		switch NALUType(nalu[0] & 0x1F) {
		case NALUTypeSPS:
			if !bytes.Equal(d.sps, nalu) {
				var spsp SPS
				err := spsp.Unmarshal(nalu)
				if err != nil {
					return 0, false, fmt.Errorf("invalid SPS: %w", err)
				}
				d.sps = nalu
				d.spsp = &spsp

				// reset state
				d.randomReceived = false
				d.imageSeen = false
				d.expectedPOC = 0
				d.reorderedFrames = 0
				d.pause = 0
				d.auCount = 0
				d.pocIncrement = 2
				d.prevPOC = 0
				d.prevPrevPOC = 0
			}

		case NALUTypeSEI:
			var sei SEI
			err := sei.Unmarshal(nalu)
			if err != nil {
				return 0, false, fmt.Errorf("invalid SEI: %w", err)
			}

			// a recovery point can only establish a new random
			// access point if no image was decoded before it;
			// otherwise it belongs to the ongoing stream and must
			// not replace the existing reference state.
			if sei.RecoveryPoint && !d.imageSeen {
				recoveryPoint = true
			}
		}
	}

	// second pass: only one image NALU participates in the DTS
	// computation of an access unit; an IDR takes precedence.
	for _, nalu := range au {
		if len(nalu) == 0 {
			return 0, false, fmt.Errorf("empty NALU")
		}

		switch NALUType(nalu[0] & 0x1F) {
		case NALUTypeIDR:
			idr = nalu
		case NALUTypeNonIDR:
			if nonIDR == nil {
				nonIDR = nalu
			}
		}
	}

	if idr != nil {
		// an IDR is itself a random access point and takes
		// precedence over a recovery point in the same unit.
		recoveryPoint = false
	}

	// an access unit without an image only carries parameter sets
	// and/or SEIs: it cannot produce a DTS and must not leave any
	// flag that would affect the next access unit.
	hasImage := idr != nil || nonIDR != nil
	if !hasImage {
		return 0, false, fmt.Errorf("access unit doesn't contain an IDR or non-IDR NALU")
	}

	if d.spsp == nil {
		return 0, false, fmt.Errorf("SPS not received yet")
	}

	if d.spsp.PicOrderCntType == 2 || !d.spsp.FrameMbsOnlyFlag {
		d.imageSeen = true
		return pts, false, nil
	}

	if d.spsp.PicOrderCntType == 1 {
		return 0, false, fmt.Errorf("pic_order_cnt_type = 1 is not supported yet")
	}

	if !d.randomReceived {
		if idr == nil && !recoveryPoint {
			return 0, false, fmt.Errorf("random access frame not received yet")
		}
		d.randomReceived = true
	}

	var ptsDTSDiff int

	switch {
	case idr != nil:
		var err error
		d.expectedPOC, err = getPictureOrderCount(idr, d.spsp, true)
		if err != nil {
			return 0, false, err
		}

		if d.pocIncrement == 2 {
			if (d.expectedPOC % 2) != 0 {
				d.pocIncrement = 1
			}

			d.auCount = 1
			d.prevPrevPOC = 0
			d.prevPOC = d.expectedPOC
		}

		ptsDTSDiff = 0

	case nonIDR != nil && recoveryPoint:
		// first image after a recovery point SEI: it is the
		// synchronization sample and rebuilds the DTS baseline
		// from its PTS, exactly like an IDR would.
		var err error
		d.expectedPOC, err = getPictureOrderCount(nonIDR, d.spsp, false)
		if err != nil {
			return 0, false, err
		}

		if d.pocIncrement == 2 {
			if (d.expectedPOC % 2) != 0 {
				d.pocIncrement = 1
			}

			d.auCount = 1
			d.prevPrevPOC = 0
			d.prevPOC = d.expectedPOC
		}

		ptsDTSDiff = 0

	case nonIDR != nil:
		poc, err := getPictureOrderCount(nonIDR, d.spsp, false)
		if err != nil {
			return 0, false, err
		}

		if d.auCount < 5 && d.pocIncrement == 2 {
			if (poc % 2) != 0 {
				d.pocIncrement = 1
				d.expectedPOC /= 2

				if d.reorderedFrames != 0 {
					increase := d.reorderedFrames
					if (d.reorderedFrames + increase) > maxReorderedFrames {
						return 0, false, fmt.Errorf("too many reordered frames (%d)", d.reorderedFrames+increase)
					}

					d.reorderedFrames += increase
					d.pause += increase
				}
			} else if d.auCount >= 2 && (poc%4) == 0 && poc == (d.prevPOC+4) && d.prevPOC == (d.prevPrevPOC+4) {
				d.pocIncrement = 4
				d.expectedPOC *= 2
			}

			d.auCount++
			d.prevPrevPOC = d.prevPOC
			d.prevPOC = poc
		}

		d.expectedPOC += uint32(d.pocIncrement)
		d.expectedPOC &= ((1 << (d.spsp.Log2MaxPicOrderCntLsbMinus4 + 4)) - 1)

		ptsDTSDiff = int(pictureOrderCountDiff(poc, d.expectedPOC, d.spsp)) / d.pocIncrement
	}

	// the image of this access unit has now been consumed.
	d.imageSeen = true

	ptsDTSDiff += d.reorderedFrames

	switch {
	case ptsDTSDiff > (2*d.reorderedFrames + 1):
		increase := ptsDTSDiff - (2*d.reorderedFrames + 1)
		if (d.reorderedFrames + increase) > maxReorderedFrames {
			return 0, false, fmt.Errorf("too many reordered frames (%d)", d.reorderedFrames+increase)
		}

		d.reorderedFrames += increase
		d.pause += increase
		ptsDTSDiff += increase

	case ptsDTSDiff < 0:
		increase := -ptsDTSDiff
		if (d.reorderedFrames + increase) > maxReorderedFrames {
			return 0, false, fmt.Errorf("too many reordered frames (%d)", d.reorderedFrames+increase)
		}

		d.reorderedFrames += increase
		d.pause += increase
		ptsDTSDiff += increase
	}

	if d.pause > 0 {
		d.pause--
		if !d.prevDTSFilled {
			return pts, true, nil
		}
		return d.prevDTS + 90, true, nil
	}

	if !d.prevDTSFilled {
		return pts, true, nil
	}

	return d.prevDTS + (pts-d.prevDTS)/(int64(ptsDTSDiff)+1), false, nil
}

// Extract extracts the DTS of an access unit.
func (d *DTSExtractor) Extract(au [][]byte, pts int64) (int64, error) {
	dts, skipChecks, err := d.extractInner(au, pts)
	if err != nil {
		return 0, err
	}

	if !skipChecks && dts > pts {
		return 0, fmt.Errorf("DTS is greater than PTS")
	}

	if d.prevDTSFilled && dts < d.prevDTS {
		return 0, fmt.Errorf("DTS is not monotonically increasing, was %v, now is %v",
			d.prevDTS, dts)
	}

	d.prevDTSFilled = true
	d.prevDTS = dts

	return dts, err
}
