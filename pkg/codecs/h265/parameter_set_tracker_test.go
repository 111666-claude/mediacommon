package h265_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h265"
)

// Fixed NAL sequence, validated against VPS.Unmarshal / SPS.Unmarshal /
// PPS.Unmarshal. The first payload byte is a placeholder skipped by the
// parsers, which start reading at bit 8 of nalu[1:].
var (
	// VPS id 0
	trackerVPS0 = []byte{
		0x41, 0x00, 0x0c, 0x01, 0xff, 0xff, 0x01, 0x00,
		0x00, 0x00, 0x00, 0xb3, 0x80, 0x00, 0x00, 0x00,
		0x00, 0x1e, 0x7c,
	}
	// VPS id 1
	trackerVPS1 = []byte{
		0x41, 0x00, 0x1c, 0x01, 0xff, 0xff, 0x01, 0x00,
		0x00, 0x00, 0x00, 0xb3, 0x80, 0x00, 0x00, 0x00,
		0x00, 0x1e, 0x7c,
	}
	// SPS id 0 -> VPS id 0, 320x240
	trackerSPS0 = []byte{
		0x43, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
		0xb3, 0x80, 0x00, 0x00, 0x00, 0x00, 0x1e, 0xa0,
		0x0a, 0x08, 0x0f, 0x17, 0x7f, 0xc2, 0x40,
	}
	// SPS id 1 -> VPS id 0, 640x360
	trackerSPS1 = []byte{
		0x43, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
		0xb3, 0x80, 0x00, 0x00, 0x00, 0x00, 0x1e, 0x48,
		0x01, 0x40, 0x80, 0x5a, 0x5d, 0xff, 0x09, 0x00,
	}
	// PPS id 0 -> SPS id 0
	trackerPPS0 = []byte{0x45, 0x00, 0xc0, 0x03, 0x80, 0x04}
	// PPS id 1 -> SPS id 1
	trackerPPS1 = []byte{0x45, 0x00, 0x48, 0x00, 0x38, 0x00, 0x40}
	// IDR_W_RADL slice, ignored by the tracker
	trackerIDR = []byte{0x27, 0x01, 0xaf}
)

func TestParameterSetTrackerSequence(t *testing.T) {
	var tr h265.ParameterSetTracker
	tr.Initialize()

	// 1. nothing received: not ready, no event
	changed, err := tr.Process(nil)
	require.NoError(t, err)
	require.False(t, changed)
	require.False(t, tr.Ready())
	require.False(t, tr.Pending())

	// 2. VPS + SPS but no PPS: not ready, no decodable config
	changed, err = tr.Process([][]byte{trackerVPS0, trackerSPS0})
	require.NoError(t, err)
	require.False(t, changed)
	require.False(t, tr.Ready())

	// 3. full chain VPS0 + SPS0 + PPS0: change #1
	changed, err = tr.Process([][]byte{trackerVPS0, trackerSPS0, trackerPPS0, trackerIDR})
	require.NoError(t, err)
	require.True(t, changed)
	require.True(t, tr.Ready())
	require.False(t, tr.Pending())
	require.Equal(t, h265.ParameterSetConfig{
		VPS: trackerVPS0,
		SPS: trackerSPS0,
		PPS: trackerPPS0,
	}, tr.Config())

	// 4. identical bytes resent: no event
	changed, err = tr.Process([][]byte{trackerVPS0, trackerSPS0, trackerPPS0})
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, h265.ParameterSetConfig{
		VPS: trackerVPS0,
		SPS: trackerSPS0,
		PPS: trackerPPS0,
	}, tr.Config())

	// 5. out-of-order PPS1 before SPS1: pending, previous config stays
	changed, err = tr.Process([][]byte{trackerPPS1})
	require.NoError(t, err)
	require.False(t, changed)
	require.True(t, tr.Pending())
	require.True(t, tr.Ready())
	require.Equal(t, h265.ParameterSetConfig{
		VPS: trackerVPS0,
		SPS: trackerSPS0,
		PPS: trackerPPS0,
	}, tr.Config())

	// 6. SPS1 arrives (VPS0 already cached): chain resolves -> change #2
	changed, err = tr.Process([][]byte{trackerSPS1})
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, tr.Pending())
	require.Equal(t, h265.ParameterSetConfig{
		VPS: trackerVPS0,
		SPS: trackerSPS1,
		PPS: trackerPPS1,
	}, tr.Config())

	// 7. back to PPS0/SPS0: change #3
	changed, err = tr.Process([][]byte{trackerPPS0})
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, h265.ParameterSetConfig{
		VPS: trackerVPS0,
		SPS: trackerSPS0,
		PPS: trackerPPS0,
	}, tr.Config())
}

func TestParameterSetTrackerMissingVPS(t *testing.T) {
	var tr h265.ParameterSetTracker
	tr.Initialize()

	// SPS0 + PPS0 without the referenced VPS: pending, not ready
	changed, err := tr.Process([][]byte{trackerSPS0, trackerPPS0})
	require.NoError(t, err)
	require.False(t, changed)
	require.True(t, tr.Pending())
	require.False(t, tr.Ready())

	// VPS arrives: chain completes -> change
	changed, err = tr.Process([][]byte{trackerVPS0})
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, tr.Pending())
	require.True(t, tr.Ready())
	require.Equal(t, h265.ParameterSetConfig{
		VPS: trackerVPS0,
		SPS: trackerSPS0,
		PPS: trackerPPS0,
	}, tr.Config())
}

func TestParameterSetTrackerReset(t *testing.T) {
	var tr h265.ParameterSetTracker
	tr.Initialize()

	changed, err := tr.Process([][]byte{trackerVPS0, trackerSPS0, trackerPPS0})
	require.NoError(t, err)
	require.True(t, changed)
	require.True(t, tr.Ready())

	// stream reset: all state is dropped
	tr.Reset()
	require.False(t, tr.Ready())
	require.False(t, tr.Pending())

	// a PPS referencing the forgotten SPS must stay pending
	changed, err = tr.Process([][]byte{trackerPPS0})
	require.NoError(t, err)
	require.False(t, changed)
	require.True(t, tr.Pending())
	require.False(t, tr.Ready())

	// rebuilding from scratch produces a change again
	changed, err = tr.Process([][]byte{trackerVPS0, trackerSPS0, trackerPPS0})
	require.NoError(t, err)
	require.True(t, changed)
	require.True(t, tr.Ready())
	require.False(t, tr.Pending())
}

func TestParameterSetTrackerInvalid(t *testing.T) {
	t.Run("invalid VPS", func(t *testing.T) {
		var tr h265.ParameterSetTracker
		tr.Initialize()
		// NAL header says VPS but the payload is too short
		_, err := tr.Process([][]byte{{0x41, 0x00}})
		require.Error(t, err)
		require.False(t, tr.Ready())
	})

	t.Run("invalid SPS", func(t *testing.T) {
		var tr h265.ParameterSetTracker
		tr.Initialize()
		_, err := tr.Process([][]byte{{0x43, 0x00, 0x01}})
		require.Error(t, err)
	})

	t.Run("invalid PPS", func(t *testing.T) {
		var tr h265.ParameterSetTracker
		tr.Initialize()
		_, err := tr.Process([][]byte{{0x45, 0x00}})
		require.Error(t, err)
	})

	t.Run("wrong NAL type in VPS parser", func(t *testing.T) {
		var v h265.VPS
		require.Error(t, v.Unmarshal(trackerSPS0))
	})

	t.Run("empty NALU", func(t *testing.T) {
		var tr h265.ParameterSetTracker
		tr.Initialize()
		_, err := tr.Process([][]byte{{}})
		require.Error(t, err)
	})

	t.Run("state unchanged after error", func(t *testing.T) {
		var tr h265.ParameterSetTracker
		tr.Initialize()
		_, err := tr.Process([][]byte{trackerVPS0, trackerSPS0, trackerPPS0})
		require.NoError(t, err)

		_, err = tr.Process([][]byte{{0x43, 0x00, 0x01}, trackerPPS0})
		require.Error(t, err)
		require.True(t, tr.Ready())
		require.Equal(t, h265.ParameterSetConfig{
			VPS: trackerVPS0,
			SPS: trackerSPS0,
			PPS: trackerPPS0,
		}, tr.Config())
	})
}

func TestParameterSetTrackerOversizedCache(t *testing.T) {
	var tr h265.ParameterSetTracker
	tr.Initialize()

	big := make([]byte, h265.MaxAccessUnitSize+1)
	big[0] = 0x43 // claim to be a SPS
	_, err := tr.Process([][]byte{big})
	require.Error(t, err)
	require.False(t, tr.Ready())
}

func TestParameterSetTrackerChangeCount(t *testing.T) {
	sequence := [][][]byte{
		{trackerIDR},               // nothing
		{trackerVPS0, trackerSPS0}, // no PPS yet
		{trackerVPS0, trackerSPS0, trackerPPS0, trackerIDR}, // change 1
		{trackerIDR},                            // no parameter sets
		{trackerVPS0, trackerSPS0, trackerPPS0}, // identical, no change
		{trackerPPS1},                           // pending (SPS1 missing)
		{trackerSPS1},                           // change 2
		{trackerPPS0},                           // change 3
		{trackerSPS1, trackerPPS1},              // change 4
	}

	var tr h265.ParameterSetTracker
	tr.Initialize()

	changes := 0
	for i, au := range sequence {
		changed, err := tr.Process(au)
		require.NoError(t, err, "AU %d", i)
		if changed {
			changes++
		}
	}

	require.Equal(t, 4, changes)
	require.True(t, tr.Ready())
	require.False(t, tr.Pending())
	require.Equal(t, h265.ParameterSetConfig{
		VPS: trackerVPS0,
		SPS: trackerSPS1,
		PPS: trackerPPS1,
	}, tr.Config())
}

func TestParameterSetTrackerPendingPersists(t *testing.T) {
	var tr h265.ParameterSetTracker
	tr.Initialize()

	changed, err := tr.Process([][]byte{trackerVPS0, trackerSPS0, trackerPPS0})
	require.NoError(t, err)
	require.True(t, changed)

	// PPS1 references the not-yet-seen SPS1: pending
	changed, err = tr.Process([][]byte{trackerPPS1})
	require.NoError(t, err)
	require.False(t, changed)
	require.True(t, tr.Pending())

	// parameter-set-free access units preserve the pending state
	changed, err = tr.Process([][]byte{trackerIDR})
	require.NoError(t, err)
	require.False(t, changed)
	require.True(t, tr.Pending())
	require.True(t, tr.Ready())
	require.Equal(t, h265.ParameterSetConfig{
		VPS: trackerVPS0,
		SPS: trackerSPS0,
		PPS: trackerPPS0,
	}, tr.Config())

	// SPS1 (VPS0 already cached) resolves the chain
	changed, err = tr.Process([][]byte{trackerSPS1})
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, tr.Pending())
	require.Equal(t, h265.ParameterSetConfig{
		VPS: trackerVPS0,
		SPS: trackerSPS1,
		PPS: trackerPPS1,
	}, tr.Config())
}
