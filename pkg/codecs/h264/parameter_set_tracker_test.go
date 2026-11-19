package h264_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
)

// Fixed NAL sequence, validated against SPS.Unmarshal / PPS.Unmarshal.
var (
	// SPS id 0, Baseline profile, 320x240
	trackerSPS0 = []byte{0x67, 0x42, 0x00, 0x1e, 0xda, 0x05, 0x07, 0xe0}
	// SPS id 1, Baseline profile, 640x368
	trackerSPS1 = []byte{0x67, 0x42, 0x00, 0x1e, 0x56, 0x80, 0xa0, 0x2f, 0x80}
	// PPS id 0 -> SPS id 0
	trackerPPS0 = []byte{0x68, 0xce, 0x3c}
	// PPS id 1 -> SPS id 1
	trackerPPS1 = []byte{0x68, 0x48, 0xe3, 0xc0}
	// PPS id 2 -> SPS id 0
	trackerPPS2 = []byte{0x68, 0x73, 0x8f, 0x00}
	// IDR slice, ignored by the tracker
	trackerIDR = []byte{0x65, 0x88, 0x80}
)

func TestParameterSetTrackerSequence(t *testing.T) {
	var tr h264.ParameterSetTracker
	tr.Initialize()

	// 1. nothing received: not ready, no event
	changed, err := tr.Process(nil)
	require.NoError(t, err)
	require.False(t, changed)
	require.False(t, tr.Ready())
	require.False(t, tr.Pending())

	// 2. SPS alone: still not ready, no decodable config reported
	changed, err = tr.Process([][]byte{trackerSPS0})
	require.NoError(t, err)
	require.False(t, changed)
	require.False(t, tr.Ready())
	require.False(t, tr.Pending())

	// 3. SPS0 + PPS0: first complete config -> change #1
	changed, err = tr.Process([][]byte{trackerSPS0, trackerPPS0, trackerIDR})
	require.NoError(t, err)
	require.True(t, changed)
	require.True(t, tr.Ready())
	require.False(t, tr.Pending())
	require.Equal(t, h264.ParameterSetConfig{SPS: trackerSPS0, PPS: trackerPPS0}, tr.Config())

	// 4. identical bytes resent: no event
	changed, err = tr.Process([][]byte{trackerSPS0, trackerPPS0})
	require.NoError(t, err)
	require.False(t, changed)
	require.True(t, tr.Ready())
	require.Equal(t, h264.ParameterSetConfig{SPS: trackerSPS0, PPS: trackerPPS0}, tr.Config())

	// 5. out-of-order PPS before its SPS: pending, previous config stays in effect
	changed, err = tr.Process([][]byte{trackerPPS1})
	require.NoError(t, err)
	require.False(t, changed)
	require.True(t, tr.Pending())
	require.True(t, tr.Ready()) // previous config not clobbered
	require.Equal(t, h264.ParameterSetConfig{SPS: trackerSPS0, PPS: trackerPPS0}, tr.Config())

	// 6. missing SPS arrives: chain resolves -> change #2
	changed, err = tr.Process([][]byte{trackerSPS1})
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, tr.Pending())
	require.Equal(t, h264.ParameterSetConfig{SPS: trackerSPS1, PPS: trackerPPS1}, tr.Config())

	// 7. PPS2 reuses SPS0: switching to it changes the PPS byte only -> change #3
	changed, err = tr.Process([][]byte{trackerPPS2})
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, h264.ParameterSetConfig{SPS: trackerSPS0, PPS: trackerPPS2}, tr.Config())

	// 8. ID reuse: PPS0 re-sent with identical bytes selects the old
	// combination back -> change #4, and SPS0 is still cached
	changed, err = tr.Process([][]byte{trackerPPS0})
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, h264.ParameterSetConfig{SPS: trackerSPS0, PPS: trackerPPS0}, tr.Config())

	// exactly four changes over the whole sequence
}

func TestParameterSetTrackerReset(t *testing.T) {
	var tr h264.ParameterSetTracker
	tr.Initialize()

	changed, err := tr.Process([][]byte{trackerSPS0, trackerPPS0})
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
	changed, err = tr.Process([][]byte{trackerSPS0, trackerPPS0})
	require.NoError(t, err)
	require.True(t, changed)
	require.True(t, tr.Ready())
	require.False(t, tr.Pending())
}

func TestParameterSetTrackerInvalid(t *testing.T) {
	t.Run("invalid SPS", func(t *testing.T) {
		var tr h264.ParameterSetTracker
		tr.Initialize()
		// NAL header says SPS but the payload is truncated
		_, err := tr.Process([][]byte{{0x67, 0x42}})
		require.Error(t, err)
		require.False(t, tr.Ready())
	})

	t.Run("invalid PPS", func(t *testing.T) {
		var tr h264.ParameterSetTracker
		tr.Initialize()
		// NAL header says PPS but there is no payload
		_, err := tr.Process([][]byte{{0x68}})
		require.Error(t, err)
	})

	t.Run("wrong NAL type in PPS parser", func(t *testing.T) {
		var p h264.PPS
		require.Error(t, p.Unmarshal(trackerSPS0))
	})

	t.Run("empty NALU", func(t *testing.T) {
		var tr h264.ParameterSetTracker
		tr.Initialize()
		_, err := tr.Process([][]byte{{}})
		require.Error(t, err)
	})

	t.Run("state unchanged after error", func(t *testing.T) {
		var tr h264.ParameterSetTracker
		tr.Initialize()
		_, err := tr.Process([][]byte{trackerSPS0, trackerPPS0})
		require.NoError(t, err)

		_, err = tr.Process([][]byte{{0x67, 0x42}, trackerPPS0})
		require.Error(t, err)
		// active config is untouched
		require.True(t, tr.Ready())
		require.Equal(t, h264.ParameterSetConfig{SPS: trackerSPS0, PPS: trackerPPS0}, tr.Config())
	})
}

func TestParameterSetTrackerOversizedCache(t *testing.T) {
	var tr h264.ParameterSetTracker
	tr.Initialize()

	// a single NALU larger than the allowed cache is rejected
	big := make([]byte, h264.MaxAccessUnitSize+1)
	big[0] = 0x67 // claim to be a SPS
	_, err := tr.Process([][]byte{big})
	require.Error(t, err)
	require.False(t, tr.Ready())
}

func TestParameterSetTrackerChangeCount(t *testing.T) {
	// fixed end-to-end sequence with an explicit change counter
	sequence := [][][]byte{
		{trackerIDR},                           // no parameter sets
		{trackerSPS0},                          // incomplete
		{trackerSPS0, trackerPPS0, trackerIDR}, // change 1
		{trackerIDR},                           // no parameter sets
		{trackerSPS0, trackerPPS0},             // identical, no change
		{trackerPPS1},                          // pending (SPS1 missing)
		{trackerSPS1},                          // change 2
		{trackerPPS2},                          // change 3 (SPS0 reused)
		{trackerSPS1, trackerPPS1},             // change 4
	}

	var tr h264.ParameterSetTracker
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
	require.Equal(t, h264.ParameterSetConfig{SPS: trackerSPS1, PPS: trackerPPS1}, tr.Config())
}

func TestParameterSetTrackerPendingPersists(t *testing.T) {
	var tr h264.ParameterSetTracker
	tr.Initialize()

	// establish a working config
	changed, err := tr.Process([][]byte{trackerSPS0, trackerPPS0})
	require.NoError(t, err)
	require.True(t, changed)

	// a PPS whose SPS is missing leaves the tracker pending
	changed, err = tr.Process([][]byte{trackerPPS1})
	require.NoError(t, err)
	require.False(t, changed)
	require.True(t, tr.Pending())

	// access units carrying no parameter set do not clear the pending state
	changed, err = tr.Process([][]byte{trackerIDR})
	require.NoError(t, err)
	require.False(t, changed)
	require.True(t, tr.Pending())
	require.True(t, tr.Ready())
	require.Equal(t, h264.ParameterSetConfig{SPS: trackerSPS0, PPS: trackerPPS0}, tr.Config())

	// the missing SPS finally resolves the chain
	changed, err = tr.Process([][]byte{trackerSPS1})
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, tr.Pending())
	require.Equal(t, h264.ParameterSetConfig{SPS: trackerSPS1, PPS: trackerPPS1}, tr.Config())
}
