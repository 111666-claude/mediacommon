package mpegts

import (
	"errors"
	"fmt"
	"io"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
)

// errLATMNoConfig is returned when the whole stream was scanned without
// finding any packet carrying a StreamMuxConfig.
var errLATMNoConfig = errors.New("LATM stream mux configuration not found in any packet")

// findLATMSyncWord returns the offset of the first AudioSyncStream syncword
// (0x56, followed by the 3 high bits 0b111) strictly after start, or -1 if
// none is found.
func findLATMSyncWord(buf []byte, start int) int {
	for i := start; i+1 < len(buf); i++ {
		if buf[i] == 0x56 && buf[i+1]&0xe0 == 0xe0 {
			return i
		}
	}
	return -1
}

// parseLATMAudioSyncStream decodes the AudioMuxElements found in a PES payload.
//
// A PES may start with a truncated LATM frame left over by the capture start
// or a packet loss. In that case the leading bytes do not contain a valid
// syncword: the parser resynchronizes on a syncword instead of failing. Every
// candidate syncword is tried until one decodes successfully, which makes
// resync robust against a syncword-looking byte sequence inside the truncated
// prefix. When no candidate can be decoded, the payload is unusable and an
// error is returned.
//
// Elements that reuse a previously signaled configuration
// (useSameStreamMux = 1) cannot be decoded on their own and are forwarded as
// raw bytes. Every other element is fully decoded so that an invalid
// AudioSpecificConfig or a truncated payload is reported instead of being
// silently accepted. config is set to the StreamMuxConfig carried by the
// first configuration-bearing element and stays nil when none is present.
func parseLATMAudioSyncStream(data []byte) (elements [][]byte, config *mpeg4audio.StreamMuxConfig, err error) {
	var stream mpeg4audio.AudioSyncStream
	err = stream.Unmarshal(data)
	if err != nil {
		firstErr := err

		// resynchronize: try every candidate syncword.
		offset := findLATMSyncWord(data, 0)
		for offset >= 0 {
			stream = mpeg4audio.AudioSyncStream{}
			err = stream.Unmarshal(data[offset:])
			if err == nil {
				break
			}
			firstErr = err
			offset = findLATMSyncWord(data, offset+1)
		}

		if offset < 0 {
			return nil, nil, fmt.Errorf("unable to decode LATM: no resync possible in PES payload: %w", firstErr)
		}
	}

	elements = stream.AudioMuxElements

	for _, element := range elements {
		if len(element) == 0 {
			return nil, nil, fmt.Errorf("unable to decode LATM: empty audio mux element")
		}

		// first bit of an audio mux element is useSameStreamMux
		useSameStreamMux := element[0]&0x80 != 0
		if useSameStreamMux {
			// the element reuses a previously signaled StreamMuxConfig and is
			// forwarded untouched; there is nothing to validate here.
			continue
		}

		var ame mpeg4audio.AudioMuxElement
		ame.MuxConfigPresent = true
		err = ame.Unmarshal(element)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to decode LATM audio mux element: %w", err)
		}

		if config == nil && ame.StreamMuxConfig != nil {
			config = ame.StreamMuxConfig
		}
	}

	return elements, config, nil
}

// findMPEG4AudioLATMConfig scans the packets of a LATM track until it finds
// the first packet carrying a complete StreamMuxConfig.
//
// Packets made only of truncated frames, packets that cannot be resynchronized
// and packets that only reuse a previous configuration (useSameStreamMux) are
// skipped so that a single corrupted packet never prevents a track from being
// recognized. A traceable error is returned when the end of the stream is
// reached without any configuration (the last parse error encountered is
// joined for traceability), or when an AudioSpecificConfig is invalid.
//
// The demuxer is consumed once and forward only, so scanning a stream stays
// linear in the number of TS packets even with several tracks: each packet is
// visited a single time instead of being searched again from the beginning.
func findMPEG4AudioLATMConfig(dem *robustDemuxer, pid uint16) (*mpeg4audio.StreamMuxConfig, error) {
	var lastErr error

	for {
		data, err := dem.nextData()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if lastErr != nil {
					return nil, fmt.Errorf("%w (last packet error: %v)", errLATMNoConfig, lastErr)
				}
				return nil, errLATMNoConfig
			}
			return nil, err
		}

		if data.PES == nil || data.PID != pid {
			continue
		}

		_, config, err := parseLATMAudioSyncStream(data.PES.Data)
		if err != nil {
			// the packet is unusable but later packets may carry a valid config.
			lastErr = err
			continue
		}

		if config != nil {
			return config, nil
		}
	}
}
