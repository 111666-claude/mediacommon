package codecs

import "github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"

// MPEG4AudioLATM is a MPEG-4 Audio LATM codec.
// Specification: ISO 13818-1
type MPEG4AudioLATM struct {
	// StreamMuxConfig is the configuration found in the stream.
	// It is filled by the Reader from the first audioMuxElement that
	// contains a StreamMuxConfig (useSameStreamMux = 0), and it can
	// contain multiple programs and layers.
	// It can be nil when the codec is instantiated manually for writing.
	StreamMuxConfig *mpeg4audio.StreamMuxConfig

	// in Go, empty structs share the same pointer,
	// therefore they cannot be used as map keys
	// or in equality operations. Prevent this.
	unused int //nolint:unused
}

// IsVideo implements Codec.
func (*MPEG4AudioLATM) IsVideo() bool {
	return false
}

func (*MPEG4AudioLATM) isCodec() {}
