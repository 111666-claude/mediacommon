package mpegts

import (
	"fmt"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
)

// latmState keeps the state of an LATM track between PES packets.
// The StreamMuxConfig of an audioMuxElement with useSameStreamMux = 1
// is carried by a previous audioMuxElement, therefore it must be
// remembered until the stream ends or a new configuration is found.
type latmState struct {
	config *mpeg4audio.StreamMuxConfig
}

// parseLATMAudioSyncStream decodes an AudioSyncStream made of LOAS-framed
// audioMuxElements. The first element carrying a StreamMuxConfig
// (useSameStreamMux = 0) establishes the complete configuration, including
// every program and layer, and becomes the state reused afterwards;
// elements with useSameStreamMux = 1 decode against that state. Each track
// owns its own state, therefore programs, layers, sample rates and buffer
// fullness of different tracks can never overwrite one another.
//
// Every element is parsed exactly once, so the cost stays linear in the
// number of elements. The raw elements are returned unchanged, which allows
// writing them back and reproducing the original bytes.
//
// state is updated in place and must not be shared between tracks.
func parseLATMAudioSyncStream(
	state *latmState,
	data []byte,
) ([][]byte, error) {
	var sync mpeg4audio.AudioSyncStream

	err := sync.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("invalid LATM AudioSyncStream: %w", err)
	}

	for _, raw := range sync.AudioMuxElements {
		var el mpeg4audio.AudioMuxElement
		el.MuxConfigPresent = true
		el.StreamMuxConfig = state.config

		err = el.Unmarshal(raw)
		if err != nil {
			// this also reports useSameStreamMux = 1 received before any
			// element carried a StreamMuxConfig, together with truncated
			// PES packets and invalid AudioSpecificConfig values.
			return nil, fmt.Errorf("invalid LATM AudioMuxElement: %w", err)
		}

		if el.UseSameStreamMux {
			continue
		}

		if el.StreamMuxConfig == nil || len(el.StreamMuxConfig.Programs) == 0 {
			return nil, fmt.Errorf("invalid LATM AudioMuxElement: missing StreamMuxConfig")
		}

		state.config = el.StreamMuxConfig
	}

	return sync.AudioMuxElements, nil
}

// findMPEG4AudioLATMConfig scans the track until an audioMuxElement carrying
// a StreamMuxConfig is found. PES packets that cannot be decoded (for example
// truncated PES or packets containing data other than LOAS-framed LATM) are
// skipped: the configuration is located in the first valid packet instead.
// The demuxer is consumed linearly, PID filtering is performed on the parsed
// data units and no packet is read twice.
func findMPEG4AudioLATMConfig(dem *robustDemuxer, pid uint16) (*mpeg4audio.StreamMuxConfig, error) {
	state := &latmState{}

	for {
		data, err := dem.nextData()
		if err != nil {
			return nil, err
		}

		if data.PES == nil || data.PID != pid {
			continue
		}

		_, err = parseLATMAudioSyncStream(state, data.PES.Data)
		if err != nil {
			continue
		}

		if state.config != nil {
			return state.config, nil
		}
	}
}
