package opus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func loadIDHeaderFixture(t *testing.T, name string) *IDHeader {
	t.Helper()

	buf, err := os.ReadFile(filepath.Join("testdata", "idheader", name+".bin"))
	require.NoError(t, err)

	var h IDHeader
	err = h.Unmarshal(buf)
	require.NoError(t, err)

	return &h
}

func TestChannelMappingUnmarshalFixtures(t *testing.T) {
	for _, ca := range []struct {
		name string
		m    ChannelMapping
	}{
		{
			"mono",
			ChannelMapping{
				Family:             0,
				StreamCount:        1,
				CoupledStreamCount: 0,
				Layout:             []Channel{ChannelCenter},
			},
		},
		{
			"stereo",
			ChannelMapping{
				Family:             0,
				StreamCount:        1,
				CoupledStreamCount: 1,
				Layout:             []Channel{ChannelLeft, ChannelRight},
			},
		},
		{
			"5.1",
			ChannelMapping{
				Family:             1,
				StreamCount:        4,
				CoupledStreamCount: 2,
				Mapping:            []uint8{0, 4, 1, 2, 3, 5},
				Layout: []Channel{
					ChannelLeft, ChannelCenter, ChannelRight,
					ChannelRearLeft, ChannelRearRight, ChannelLFE,
				},
			},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			h := loadIDHeaderFixture(t, ca.name)

			m, err := h.ChannelMapping()
			require.NoError(t, err)
			require.Equal(t, &ca.m, m)

			// rebuilding the header must produce the same family and table.
			var rebuilt IDHeader
			rebuilt.ChannelCount = h.ChannelCount
			m.ToIDHeader(&rebuilt)
			require.Equal(t, h.ChannelMappingFamily, rebuilt.ChannelMappingFamily)
			if len(h.ChannelMappingTable) == 0 {
				require.Empty(t, rebuilt.ChannelMappingTable)
			} else {
				require.Equal(t, h.ChannelMappingTable, rebuilt.ChannelMappingTable)
			}
		})
	}
}

func TestChannelMappingLayout(t *testing.T) {
	for _, ca := range []struct {
		name        string
		channelCnt  uint8
		streamCnt   int
		coupledCnt  int
		mapping     []uint8
		expected    []Channel
		silentCount int
	}{
		{
			"mono",
			1, 1, 0, []uint8{0},
			[]Channel{ChannelCenter}, 0,
		},
		{
			"stereo",
			2, 1, 1, []uint8{0, 1},
			[]Channel{ChannelLeft, ChannelRight}, 0,
		},
		{
			"5.1",
			6, 4, 2, []uint8{0, 4, 1, 2, 3, 5},
			[]Channel{
				ChannelLeft, ChannelCenter, ChannelRight,
				ChannelRearLeft, ChannelRearRight, ChannelLFE,
			},
			0,
		},
		{
			"5.1 with silent center",
			6, 3, 2, []uint8{0, 255, 1, 4, 2, 3},
			[]Channel{
				ChannelLeft, ChannelSilence, ChannelRight,
				ChannelRearLeft, ChannelRearRight, ChannelLFE,
			},
			1,
		},
		{
			"7.1",
			8, 5, 3, []uint8{0, 6, 1, 4, 5, 2, 3, 7},
			[]Channel{
				ChannelLeft, ChannelCenter, ChannelRight,
				ChannelSideLeft, ChannelSideRight,
				ChannelRearLeft, ChannelRearRight, ChannelLFE,
			},
			0,
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			m, err := NewChannelMapping(ca.streamCnt, ca.coupledCnt, ca.mapping)
			require.NoError(t, err)
			require.Equal(t, ca.expected, m.Layout)

			silent := 0
			for _, c := range m.Layout {
				if c == ChannelSilence {
					silent++
				}
			}
			require.Equal(t, ca.silentCount, silent)

			h := &IDHeader{
				Version:         1,
				ChannelCount:    ca.channelCnt,
				InputSampleRate: 48000,
			}
			m.ToIDHeader(h)

			enc, err := h.Marshal()
			require.NoError(t, err)

			var decoded IDHeader
			err = decoded.Unmarshal(enc)
			require.NoError(t, err)

			roundTrip, err := decoded.ChannelMapping()
			require.NoError(t, err)
			require.Equal(t, m, roundTrip)
		})
	}
}

func TestChannelMappingUnmarshalErrors(t *testing.T) {
	for _, ca := range []struct {
		name string
		h    IDHeader
		err  string
	}{
		{
			"unsupported family",
			IDHeader{
				Version:              1,
				ChannelCount:         2,
				ChannelMappingFamily: 2,
				ChannelMappingTable:  []uint8{1, 1, 0, 1},
			},
			"unsupported channel mapping family: 2",
		},
		{
			"family 0 with non-empty table",
			IDHeader{
				Version:              1,
				ChannelCount:         2,
				ChannelMappingFamily: 0,
				ChannelMappingTable:  []uint8{0},
			},
			"channel mapping table must be empty for family 0, got 1 bytes",
		},
		{
			"family 0 with 3 channels",
			IDHeader{
				Version:              1,
				ChannelCount:         3,
				ChannelMappingFamily: 0,
			},
			"family 0 supports 1 or 2 channels, got 3",
		},
		{
			"family 1 table too short",
			IDHeader{
				Version:              1,
				ChannelCount:         6,
				ChannelMappingFamily: 1,
				ChannelMappingTable:  []uint8{4, 2, 0, 4, 1},
			},
			"channel mapping table length is 5, expected 8 for 6 channels",
		},
		{
			"family 1 table too long",
			IDHeader{
				Version:              1,
				ChannelCount:         2,
				ChannelMappingFamily: 1,
				ChannelMappingTable:  []uint8{1, 1, 0, 1, 2},
			},
			"channel mapping table length is 5, expected 4 for 2 channels",
		},
		{
			"zero stream count",
			IDHeader{
				Version:              1,
				ChannelCount:         2,
				ChannelMappingFamily: 1,
				ChannelMappingTable:  []uint8{0, 0, 0, 1},
			},
			"stream count must not be zero",
		},
		{
			"coupled count exceeds stream count",
			IDHeader{
				Version:              1,
				ChannelCount:         2,
				ChannelMappingFamily: 1,
				ChannelMappingTable:  []uint8{1, 2, 0, 1},
			},
			"coupled stream count (2) exceeds stream count (1)",
		},
		{
			"mapping index out of range",
			IDHeader{
				Version:              1,
				ChannelCount:         2,
				ChannelMappingFamily: 1,
				ChannelMappingTable:  []uint8{2, 0, 0, 5},
			},
			"channel mapping index 5 is out of range (max 1)",
		},
		{
			"duplicated channel mapping index",
			IDHeader{
				Version:              1,
				ChannelCount:         3,
				ChannelMappingFamily: 1,
				ChannelMappingTable:  []uint8{2, 0, 0, 1, 1},
			},
			"duplicated channel mapping index 1 at position 2",
		},
		{
			"unknown channel count for family 1",
			IDHeader{
				Version:              1,
				ChannelCount:         9,
				ChannelMappingFamily: 1,
				ChannelMappingTable: []uint8{
					5, 4, 0, 6, 1, 4, 5, 2, 3, 7, 8,
				},
			},
			"family 1 does not define a layout for 9 channels",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var m ChannelMapping
			err := m.Unmarshal(&ca.h)
			require.EqualError(t, err, ca.err)
		})
	}
}

func TestChannelMappingCorruptedFixture(t *testing.T) {
	h := loadIDHeaderFixture(t, "corrupted")

	var m ChannelMapping
	err := m.Unmarshal(h)
	require.Error(t, err)
	require.Contains(t, err.Error(), "channel mapping index 5 is out of range")
}

func TestNewChannelMappingErrors(t *testing.T) {
	for _, ca := range []struct {
		name       string
		streamCnt  int
		coupledCnt int
		mapping    []uint8
		err        string
	}{
		{
			"zero stream count",
			0, 0, []uint8{0},
			"stream count must not be zero",
		},
		{
			"coupled count conflict",
			1, 2, []uint8{0, 1},
			"coupled stream count (2) exceeds stream count (1)",
		},
		{
			"index out of range",
			2, 0, []uint8{0, 5},
			"channel mapping index 5 is out of range (max 1)",
		},
		{
			"duplicated channel",
			2, 0, []uint8{0, 0},
			"duplicated channel mapping index 0 at position 1",
		},
		{
			"empty mapping",
			1, 0, nil,
			"channel count must be between 1 and 255, got 0",
		},
		{
			"unsupported channel count",
			9, 0, []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8},
			"family 1 does not define a layout for 9 channels",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			_, err := NewChannelMapping(ca.streamCnt, ca.coupledCnt, ca.mapping)
			require.EqualError(t, err, ca.err)
		})
	}
}

func TestChannelString(t *testing.T) {
	require.Equal(t, "L", ChannelLeft.String())
	require.Equal(t, "LFE", ChannelLFE.String())
	require.Equal(t, "silence", ChannelSilence.String())
	require.Equal(t, "unknown(30)", Channel(30).String())
}
