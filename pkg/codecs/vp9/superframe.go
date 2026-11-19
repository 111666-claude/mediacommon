package vp9

import (
	"errors"
	"fmt"
)

const (
	// maxSuperFrameFrames is the maximum number of frames that can be stored
	// inside a superframe, as encoded by the 3-bit frames-minus-1 field.
	maxSuperFrameFrames = 8
)

// ErrEmptySuperFrame is returned when a superframe doesn't contain any frame.
var ErrEmptySuperFrame = errors.New("superframe doesn't contain any frame")

// SuperFrame is a VP9 superframe: one or more VP9 frames that can be
// decoded/encoded from/to the superframe index format.
// Specification: VP9 Bitstream & Decoding Process Specification v0.6,
// section 4.3.4 (RFC 9128, section 4.2.3)
type SuperFrame [][]byte

// Unmarshal decodes a SuperFrame.
// A buffer without a superframe index is treated as a single frame.
// The returned frames are sub-slices of buf and therefore share its memory.
func (sf *SuperFrame) Unmarshal(buf []byte) error {
	*sf = nil

	if len(buf) == 0 {
		return ErrEmptySuperFrame
	}

	// The superframe index, when present, is at the end of the buffer and
	// starts with a marker whose two most significant bits are both 1.
	if buf[len(buf)-1]&0xC0 != 0xC0 {
		if len(buf) > MaxFrameSize {
			return fmt.Errorf("frame size (%d) is too big, maximum is %d", len(buf), MaxFrameSize)
		}

		*sf = SuperFrame{buf}
		return nil
	}

	marker := buf[len(buf)-1]
	width := int((marker&0x30)>>4) + 1
	framesCount := int(marker&0x07) + 1
	indexSize := 2 + framesCount*width

	if len(buf) < indexSize {
		return fmt.Errorf("truncated superframe index: buffer has %d bytes, index needs %d",
			len(buf), indexSize)
	}

	indexStart := len(buf) - indexSize
	if buf[indexStart] != marker {
		return fmt.Errorf("superframe index marker mismatch: first byte 0x%.2x, last byte 0x%.2x",
			buf[indexStart], marker)
	}

	frames := make(SuperFrame, 0, framesCount)
	pos := indexStart + 1
	dataEnd := indexStart
	dataPos := 0

	for range framesCount {
		var size int
		for range width {
			size = size<<8 | int(buf[pos])
			pos++
		}

		if size == 0 {
			return errors.New("superframe contains an empty frame")
		}

		if size > MaxFrameSize {
			return fmt.Errorf("frame size (%d) is too big, maximum is %d", size, MaxFrameSize)
		}

		if dataPos+size > dataEnd {
			return fmt.Errorf("superframe frame length sum (%d) exceeds available data (%d)",
				dataPos+size, dataEnd)
		}

		frames = append(frames, buf[dataPos:dataPos+size])
		dataPos += size
	}

	if dataPos != dataEnd {
		return fmt.Errorf("superframe frame length sum (%d) doesn't match available data (%d)",
			dataPos, dataEnd)
	}

	*sf = frames
	return nil
}

// Marshal encodes a SuperFrame.
// When the superframe contains a single frame, the frame is returned as-is
// and no superframe index is written.
func (sf SuperFrame) Marshal() ([]byte, error) {
	if len(sf) == 0 {
		return nil, ErrEmptySuperFrame
	}

	if len(sf) > maxSuperFrameFrames {
		return nil, fmt.Errorf("frame count (%d) exceeds maximum allowed (%d)",
			len(sf), maxSuperFrameFrames)
	}

	if len(sf) == 1 {
		if len(sf[0]) == 0 {
			return nil, errors.New("superframe contains an empty frame")
		}

		if len(sf[0]) > MaxFrameSize {
			return nil, fmt.Errorf("frame size (%d) is too big, maximum is %d",
				len(sf[0]), MaxFrameSize)
		}

		return sf[0], nil
	}

	// Find the minimum width (1 to 4 bytes) able to hold every frame size.
	width := 1
	for _, frame := range sf {
		if len(frame) == 0 {
			return nil, errors.New("superframe contains an empty frame")
		}

		if len(frame) > MaxFrameSize {
			return nil, fmt.Errorf("frame size (%d) is too big, maximum is %d",
				len(frame), MaxFrameSize)
		}

		for w := width; w < 4 && len(frame) > (1<<(8*w))-1; w++ {
			width = w + 1
		}
	}
	marker := byte(0xC0 | ((width - 1) << 4) | (len(sf) - 1))

	dataSize := 0
	for _, frame := range sf {
		dataSize += len(frame)
	}

	buf := make([]byte, dataSize+2+len(sf)*width)
	pos := 0
	for _, frame := range sf {
		pos += copy(buf[pos:], frame)
	}

	buf[pos] = marker
	pos++

	for _, frame := range sf {
		size := len(frame)
		for shift := (width - 1) * 8; shift >= 0; shift -= 8 {
			buf[pos] = byte(size >> shift)
			pos++
		}
	}

	buf[pos] = marker

	return buf, nil
}

// IsRandomAccess checks whether every frame of the superframe can be
// randomly accessed.
func (sf SuperFrame) IsRandomAccess() bool {
	if len(sf) == 0 {
		return false
	}

	for _, frame := range sf {
		if !IsRandomAccess(frame) {
			return false
		}
	}

	return true
}
