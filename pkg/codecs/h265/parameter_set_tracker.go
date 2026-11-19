package h265

import (
	"bytes"
	"fmt"
)

const (
	maxVPSCount         = 16
	maxSPSCount         = 16
	maxPPSCount         = 64
	maxParameterSetSize = MaxAccessUnitSize
)

// ParameterSetConfig is the parameter set combination (VPS + SPS + PPS)
// that is currently in effect.
type ParameterSetConfig struct {
	VPS []byte
	SPS []byte
	PPS []byte
}

// ParameterSetTracker tracks H265 parameter sets (VPS, SPS, PPS) across
// a stream of access units and detects when the active combination
// changes.
//
// Parameter sets are associated by their reference IDs, as required by
// ITU-T Rec. H.265: a PPS references a SPS through
// pic_parameter_set_id / sps_id, and the SPS references a VPS through
// sps_video_parameter_set_id.
type ParameterSetTracker struct {
	vpsMap     map[uint32][]byte
	spsMap     map[uint32][]byte
	spsVPSID   map[uint32]uint32
	ppsMap     map[uint32][]byte
	ppsSPSID   map[uint32]uint32
	latestPPS  uint32
	hasLatest  bool
	active     ParameterSetConfig
	hasActive  bool
	pending    bool
	totalBytes int
}

// Initialize initializes a ParameterSetTracker.
func (t *ParameterSetTracker) Initialize() {
	t.vpsMap = make(map[uint32][]byte)
	t.spsMap = make(map[uint32][]byte)
	t.spsVPSID = make(map[uint32]uint32)
	t.ppsMap = make(map[uint32][]byte)
	t.ppsSPSID = make(map[uint32]uint32)
}

// NewParameterSetTracker allocates a ParameterSetTracker.
//
// Deprecated: replaced by ParameterSetTracker.Initialize.
func NewParameterSetTracker() *ParameterSetTracker {
	t := &ParameterSetTracker{}
	t.Initialize()
	return t
}

// Reset clears the tracker state. It must be called when the stream is
// reset (for example after a discontinuity): parameter sets received
// before the reset are no longer valid.
func (t *ParameterSetTracker) Reset() {
	clear(t.vpsMap)
	clear(t.spsMap)
	clear(t.spsVPSID)
	clear(t.ppsMap)
	clear(t.ppsSPSID)
	t.latestPPS = 0
	t.hasLatest = false
	t.active = ParameterSetConfig{}
	t.hasActive = false
	t.pending = false
	t.totalBytes = 0
}

// Ready returns true if a complete, decodable parameter set combination
// (a PPS, the SPS it references and the VPS referenced by the SPS) is
// currently in effect.
func (t *ParameterSetTracker) Ready() bool {
	return t.hasActive
}

// Pending returns true if the most recently received PPS cannot be
// resolved yet because its referenced SPS (or the VPS referenced by
// that SPS) is missing. The previous configuration (if any) stays in
// effect while the tracker is pending.
func (t *ParameterSetTracker) Pending() bool {
	return t.pending
}

// Config returns the parameter set combination currently in effect.
// It must be called only if Ready returns true. The returned slices must
// not be modified.
func (t *ParameterSetTracker) Config() ParameterSetConfig {
	return t.active
}

// Process analyzes an access unit made of NALUs, updates the cached
// parameter sets and reports whether the active parameter set
// combination changed. NALUs that are not VPS, SPS or PPS are ignored.
//
// An invalid VPS/SPS/PPS NALU, an ID out of range or a cache that
// would exceed MaxAccessUnitSize results in an error and leaves the
// tracker state unchanged.
func (t *ParameterSetTracker) Process(au [][]byte) (bool, error) {
	type vpsEntry struct {
		id  uint32
		buf []byte
	}

	type spsEntry struct {
		id    uint32
		vpsID uint32
		buf   []byte
	}

	type ppsEntry struct {
		id    uint32
		spsID uint32
		buf   []byte
	}

	newVPS := map[uint32]vpsEntry{}
	newSPS := map[uint32]spsEntry{}
	newPPS := map[uint32]ppsEntry{}
	var lastPPSID *uint32

	for _, nalu := range au {
		if len(nalu) == 0 {
			return false, fmt.Errorf("empty NALU")
		}

		if len(nalu) > maxParameterSetSize {
			return false, fmt.Errorf("NALU size (%d) is too big, maximum is %d",
				len(nalu), maxParameterSetSize)
		}

		switch NALUType((nalu[0] >> 1) & 0b111111) {
		case NALUType_VPS_NUT:
			var vpsp VPS
			err := vpsp.Unmarshal(nalu)
			if err != nil {
				return false, fmt.Errorf("invalid VPS: %w", err)
			}

			if vpsp.ID >= maxVPSCount {
				return false, fmt.Errorf("vps_id out of range (%d)", vpsp.ID)
			}

			newVPS[uint32(vpsp.ID)] = vpsEntry{id: uint32(vpsp.ID), buf: bytes.Clone(nalu)}

		case NALUType_SPS_NUT:
			var spsp SPS
			err := spsp.Unmarshal(nalu)
			if err != nil {
				return false, fmt.Errorf("invalid SPS: %w", err)
			}

			if uint32(spsp.ID) >= maxSPSCount {
				return false, fmt.Errorf("sps_id out of range (%d)", spsp.ID)
			}

			newSPS[uint32(spsp.ID)] = spsEntry{
				id:    uint32(spsp.ID),
				vpsID: uint32(spsp.VPSID),
				buf:   bytes.Clone(nalu),
			}

		case NALUType_PPS_NUT:
			var ppsp PPS
			err := ppsp.Unmarshal(nalu)
			if err != nil {
				return false, fmt.Errorf("invalid PPS: %w", err)
			}

			if ppsp.ID >= maxPPSCount {
				return false, fmt.Errorf("pps_id out of range (%d)", ppsp.ID)
			}

			if ppsp.SPSID >= maxSPSCount {
				return false, fmt.Errorf("sps_id out of range (%d)", ppsp.SPSID)
			}

			newPPS[ppsp.ID] = ppsEntry{id: ppsp.ID, spsID: ppsp.SPSID, buf: bytes.Clone(nalu)}
			id := ppsp.ID
			lastPPSID = &id
		}
	}

	// validate the total cache size before committing anything
	added := 0

	for id, e := range newVPS {
		if old, ok := t.vpsMap[id]; !ok {
			added += len(e.buf)
		} else {
			added += len(e.buf) - len(old)
		}
	}

	for id, e := range newSPS {
		if old, ok := t.spsMap[id]; !ok {
			added += len(e.buf)
		} else {
			added += len(e.buf) - len(old)
		}
	}

	for id, e := range newPPS {
		if old, ok := t.ppsMap[id]; !ok {
			added += len(e.buf)
		} else {
			added += len(e.buf) - len(old)
		}
	}

	if t.totalBytes+added > maxParameterSetSize {
		return false, fmt.Errorf("parameter set cache is too big, maximum is %d", maxParameterSetSize)
	}

	// commit VPS/SPS/PPS updates; identical bytes do not replace the entry
	for id, e := range newVPS {
		if old, ok := t.vpsMap[id]; ok && bytes.Equal(old, e.buf) {
			continue
		}

		if old, ok := t.vpsMap[id]; ok {
			t.totalBytes -= len(old)
		}

		t.vpsMap[id] = e.buf
		t.totalBytes += len(e.buf)
	}

	for id, e := range newSPS {
		if old, ok := t.spsMap[id]; ok && bytes.Equal(old, e.buf) {
			continue
		}

		if old, ok := t.spsMap[id]; ok {
			t.totalBytes -= len(old)
		}

		t.spsMap[id] = e.buf
		t.spsVPSID[id] = e.vpsID
		t.totalBytes += len(e.buf)
	}

	for id, e := range newPPS {
		if old, ok := t.ppsMap[id]; ok && bytes.Equal(old, e.buf) {
			continue
		}

		if old, ok := t.ppsMap[id]; ok {
			t.totalBytes -= len(old)
		}

		t.ppsMap[id] = e.buf
		t.ppsSPSID[id] = e.spsID
		t.totalBytes += len(e.buf)
	}

	// the last PPS of the access unit is the candidate active one
	if lastPPSID != nil {
		if !t.hasLatest || t.latestPPS != *lastPPSID {
			t.latestPPS = *lastPPSID
			t.hasLatest = true
		}
	}

	if !t.hasLatest {
		return false, nil
	}

	// resolve the candidate chain PPS -> SPS -> VPS
	ppsBuf, ok := t.ppsMap[t.latestPPS]
	if !ok {
		t.pending = true
		return false, nil
	}

	spsID, ok := t.ppsSPSID[t.latestPPS]
	if !ok {
		t.pending = true
		return false, nil
	}

	spsBuf, ok := t.spsMap[spsID]
	if !ok {
		t.pending = true
		return false, nil
	}

	vpsID, ok := t.spsVPSID[spsID]
	if !ok {
		t.pending = true
		return false, nil
	}

	vpsBuf, ok := t.vpsMap[vpsID]
	if !ok {
		t.pending = true
		return false, nil
	}

	t.pending = false

	cfg := ParameterSetConfig{VPS: vpsBuf, SPS: spsBuf, PPS: ppsBuf}
	if t.hasActive &&
		bytes.Equal(t.active.VPS, cfg.VPS) &&
		bytes.Equal(t.active.SPS, cfg.SPS) &&
		bytes.Equal(t.active.PPS, cfg.PPS) {
		return false, nil
	}

	t.active = cfg
	t.hasActive = true

	return true, nil
}
