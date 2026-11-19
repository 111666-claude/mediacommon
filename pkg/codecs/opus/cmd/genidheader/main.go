// Generator for the OpusHead fixtures used by channel_mapping_test.go.
//
// Run from the repository root with:
//
//	go run ./pkg/codecs/opus/cmd/genidheader
package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/opus"
)

func main() {
	_, thisFile, _, _ := runtime.Caller(0)
	out := filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "idheader")

	cases := []struct {
		name string
		h    opus.IDHeader
	}{
		{
			"mono",
			opus.IDHeader{
				Version:              1,
				ChannelCount:         1,
				PreSkip:              312,
				InputSampleRate:      48000,
				OutputGain:           0,
				ChannelMappingFamily: 0,
			},
		},
		{
			"stereo",
			opus.IDHeader{
				Version:              1,
				ChannelCount:         2,
				PreSkip:              312,
				InputSampleRate:      48000,
				OutputGain:           0,
				ChannelMappingFamily: 0,
			},
		},
		{
			"5.1",
			opus.IDHeader{
				Version:              1,
				ChannelCount:         6,
				PreSkip:              0,
				InputSampleRate:      48000,
				OutputGain:           0,
				ChannelMappingFamily: 1,
				ChannelMappingTable:  []uint8{4, 2, 0, 4, 1, 2, 3, 5},
			},
		},
		{
			// mapping index 5 exceeds N+M (3+0): corrupted header.
			"corrupted",
			opus.IDHeader{
				Version:              1,
				ChannelCount:         2,
				PreSkip:              0,
				InputSampleRate:      48000,
				OutputGain:           0,
				ChannelMappingFamily: 1,
				ChannelMappingTable:  []uint8{3, 0, 0, 5},
			},
		},
	}

	for _, ca := range cases {
		buf, err := ca.h.Marshal()
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(filepath.Join(out, ca.name+".bin"), buf, 0o644)
		if err != nil {
			panic(err)
		}
	}
}
