package codecs

// ID3 is a timed ID3 metadata codec.
// Specification: ID3v2.4.0, ID3 Tag Version 2.4.0
// Specification: ISO 13818-1, table 2-86
type ID3 struct {
	// in Go, empty structs share the same pointer,
	// therefore they cannot be used as map keys
	// or in equality operations. Prevent this.
	unused int //nolint:unused
}

// IsVideo implements Codec.
func (*ID3) IsVideo() bool {
	return false
}

func (*ID3) isCodec() {}
