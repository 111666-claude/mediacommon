package h264

import (
	"bytes"
	"fmt"
)

const (
	maxSPSCount         = 32
	maxPPSCount         = 255
	maxParameterSetSize = MaxAccessUnitSize
)

// ParameterSetConfig is the parameter set combination (SPS + PPS)
// that is currently in effect.
type ParameterSetConfig struct {
	SPS []byte
	PPS []byte
}

// ParameterSetTracker tracks H264 parameter sets (SPS, PPS) across a
// stream of access units and detects when the active combination changes.
//
// Parameter sets are associated by their reference IDs, as required by
// ITU-T Rec. H.264: a PPS references a SPS through pic_parameter_set_id
// and seq_parameter_set_id.
type ParameterSetTracker struct {
	spsMap     map[uint32][]byte
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
	t.spsMap = make(map[uint32][]byte)
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
	clear(t.spsMap)
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
// (a PPS and the SPS it references) is currently in effect.
func (t *ParameterSetTracker) Ready() bool {
	return t.hasActive
}

// Pending returns true if the most recently received PPS cannot be
// resolved yet because its referenced SPS is missing. The previous
// configuration (if any) stays in effect while the tracker is pending.
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
// combination changed. NALUs that are not SPS or PPS are ignored.
//
// An invalid SPS/PPS NALU, an ID out of range or a cache that would
// exceed MaxAccessUnitSize results in an error and leaves the tracker
// state unchanged.
func (t *ParameterSetTracker) Process(au [][]byte) (bool, error) {
	type spsEntry struct {
		id  uint32
		buf []byte
	}

	type ppsEntry struct {
		id    uint32
		spsID uint32
		buf   []byte
	}

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

		switch NALUType(nalu[0] & 0x1F) {
		case NALUTypeSPS:
			var spsp SPS
			err := spsp.Unmarshal(nalu)
			if err != nil {
				return false, fmt.Errorf("invalid SPS: %w", err)
			}

			if spsp.ID >= maxSPSCount {
				return false, fmt.Errorf("sps_id out of range (%d)", spsp.ID)
			}

			newSPS[spsp.ID] = spsEntry{id: spsp.ID, buf: bytes.Clone(nalu)}

		case NALUTypePPS:
			var ppsp PPS
			err := ppsp.Unmarshal(nalu)
			if err != nil {
				return false, fmt.Errorf("invalid PPS: %w", err)
			}

			newPPS[ppsp.ID] = ppsEntry{id: ppsp.ID, spsID: ppsp.SPSID, buf: bytes.Clone(nalu)}
			id := ppsp.ID
			lastPPSID = &id
		}
	}

	// validate the total cache size before committing anything
	added := 0

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

	// commit SPS/PPS updates; identical bytes do not replace the entry
	for id, e := range newSPS {
		if old, ok := t.spsMap[id]; ok && bytes.Equal(old, e.buf) {
			continue
		}

		if old, ok := t.spsMap[id]; ok {
			t.totalBytes -= len(old)
		}

		t.spsMap[id] = e.buf
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

	// resolve the candidate chain PPS -> SPS
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

	t.pending = false

	cfg := ParameterSetConfig{SPS: spsBuf, PPS: ppsBuf}
	if t.hasActive && bytes.Equal(t.active.SPS, cfg.SPS) && bytes.Equal(t.active.PPS, cfg.PPS) {
		return false, nil
	}

	t.active = cfg
	t.hasActive = true

	return true, nil
}
