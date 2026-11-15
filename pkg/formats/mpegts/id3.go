package mpegts

import "fmt"

const (
	id3HeaderSize = 10
	id3FooterSize = 10

	// ID3v2.4, section 3.1: bit 4 of the flags byte signals a footer.
	id3FlagFooter = 0x10
)

// validateID3Tag checks that data contains exactly one complete ID3v2 tag.
// All errors report the absolute offset at which the problem has been detected.
func validateID3Tag(data []byte) error {
	if len(data) < id3HeaderSize {
		return fmt.Errorf("truncated header at offset 0: %d bytes available, %d required",
			len(data), id3HeaderSize)
	}

	if data[0] != 'I' || data[1] != 'D' || data[2] != '3' {
		return fmt.Errorf("invalid identifier at offset 0: %02x %02x %02x, expected 49 44 33",
			data[0], data[1], data[2])
	}

	major := data[3]
	revision := data[4]

	if major == 0xFF || revision == 0xFF {
		return fmt.Errorf("invalid version at offset 3: %d.%d", major, revision)
	}

	if major < 2 || major > 4 {
		return fmt.Errorf("unsupported version at offset 3: %d.%d", major, revision)
	}

	// ID3v2.2, 3 and 4 encode the tag size as a syncsafe integer.
	size := 0
	for i, b := range data[6:10] {
		if b&0x80 != 0 {
			return fmt.Errorf("invalid size at offset %d: most significant bit is set", 6+i)
		}
		size = size<<7 | int(b)
	}

	totalSize := id3HeaderSize + size
	if major >= 4 && data[5]&id3FlagFooter != 0 {
		totalSize += id3FooterSize
	}

	if len(data) < totalSize {
		return fmt.Errorf("truncated tag: %d bytes available at offset %d, %d declared by the size field",
			len(data), len(data), totalSize)
	}

	if len(data) > totalSize {
		return fmt.Errorf("unexpected data at offset %d: %d bytes after the end of the tag",
			totalSize, len(data)-totalSize)
	}

	return nil
}
