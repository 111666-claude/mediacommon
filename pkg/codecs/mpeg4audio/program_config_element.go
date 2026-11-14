package mpeg4audio

import (
	"fmt"

	"github.com/bluenviron/mediacommon/v2/pkg/bits"
)

// ProgramConfigElement is a program_config_element.
// It carries the channel layout when the AudioSpecificConfig
// channelConfiguration is set to 0, meaning that the channel
// information is explicitly provided in-band instead of being
// described by one of the fixed channel configurations.
// Specification: ISO 14496-3, 4.5.1.1 (Table 4.2)
type ProgramConfigElement struct {
	ElementInstanceTag   uint8
	Profile              uint8
	SampleRateIndex      uint8
	NumFrontElements     uint8
	NumSideElements      uint8
	NumBackElements      uint8
	NumLFEElements       uint8
	NumAssocDataElements uint8
	NumValidCCElements   uint8

	MonoMixdownPresent   bool
	MonoMixdownElement   uint8
	StereoMixdownPresent bool
	StereoMixdownElement uint8
	MatrixMixdownPresent bool
	MatrixMixdownIndex   uint8
	MatrixPseudoSurround bool

	FrontElements     []ProgramConfigElementItem
	SideElements      []ProgramConfigElementItem
	BackElements      []ProgramConfigElementItem
	LFEElements       []ProgramConfigElementItem
	AssocDataElements []ProgramConfigElementItem
	ValidCCElements   []ProgramConfigElementItem

	// Comments is the raw comment_field_bytes payload.
	Comments []byte
}

// ProgramConfigElementItem is an entry of an element list
// inside a ProgramConfigElement.
type ProgramConfigElementItem struct {
	IsCPE      bool
	ElementTag uint8
}

// ChannelCount returns the number of output channels described by the PCE.
// A channel pair element (CPE) contributes two channels, any other
// element contributes one. LFE, association data and valid CC elements
// are counted as well since they occupy a channel slot.
func (e ProgramConfigElement) ChannelCount() int {
	n := 0
	for _, list := range [][]ProgramConfigElementItem{
		e.FrontElements,
		e.SideElements,
		e.BackElements,
		e.LFEElements,
		e.AssocDataElements,
		e.ValidCCElements,
	} {
		for _, item := range list {
			if item.IsCPE {
				n += 2
			} else {
				n++
			}
		}
	}
	return n
}

// Unmarshal decodes a program_config_element from buf starting at pos.
func (e *ProgramConfigElement) unmarshalBits(buf []byte, pos *int) error {
	var err error

	e.ElementInstanceTag, err = readBits8(buf, pos, 4)
	if err != nil {
		return err
	}

	e.Profile, err = readBits8(buf, pos, 2)
	if err != nil {
		return err
	}

	e.SampleRateIndex, err = readBits8(buf, pos, 4)
	if err != nil {
		return err
	}

	e.NumFrontElements, err = readBits8(buf, pos, 4)
	if err != nil {
		return err
	}

	e.NumSideElements, err = readBits8(buf, pos, 4)
	if err != nil {
		return err
	}

	e.NumBackElements, err = readBits8(buf, pos, 4)
	if err != nil {
		return err
	}

	e.NumLFEElements, err = readBits8(buf, pos, 2)
	if err != nil {
		return err
	}

	e.NumAssocDataElements, err = readBits8(buf, pos, 3)
	if err != nil {
		return err
	}

	e.NumValidCCElements, err = readBits8(buf, pos, 3)
	if err != nil {
		return err
	}

	e.MonoMixdownPresent, err = bits.ReadFlag(buf, pos)
	if err != nil {
		return err
	}
	if e.MonoMixdownPresent {
		e.MonoMixdownElement, err = readBits8(buf, pos, 4)
		if err != nil {
			return err
		}
	}

	e.StereoMixdownPresent, err = bits.ReadFlag(buf, pos)
	if err != nil {
		return err
	}
	if e.StereoMixdownPresent {
		e.StereoMixdownElement, err = readBits8(buf, pos, 4)
		if err != nil {
			return err
		}
	}

	e.MatrixMixdownPresent, err = bits.ReadFlag(buf, pos)
	if err != nil {
		return err
	}
	if e.MatrixMixdownPresent {
		e.MatrixMixdownIndex, err = readBits8(buf, pos, 3)
		if err != nil {
			return err
		}
		e.MatrixPseudoSurround, err = bits.ReadFlag(buf, pos)
		if err != nil {
			return err
		}
	}

	e.FrontElements, err = readProgramConfigElementItems(buf, pos, int(e.NumFrontElements), true)
	if err != nil {
		return err
	}

	e.SideElements, err = readProgramConfigElementItems(buf, pos, int(e.NumSideElements), true)
	if err != nil {
		return err
	}

	e.BackElements, err = readProgramConfigElementItems(buf, pos, int(e.NumBackElements), true)
	if err != nil {
		return err
	}

	e.LFEElements, err = readProgramConfigElementItems(buf, pos, int(e.NumLFEElements), false)
	if err != nil {
		return err
	}

	e.AssocDataElements, err = readProgramConfigElementItems(buf, pos, int(e.NumAssocDataElements), false)
	if err != nil {
		return err
	}

	e.ValidCCElements, err = readProgramConfigElementItems(buf, pos, int(e.NumValidCCElements), false)
	if err != nil {
		return err
	}

	// byteAlign(): pad with zero bits up to the next byte boundary
	if (*pos % 8) != 0 {
		padding := 8 - (*pos % 8)
		_, err = bits.ReadBits(buf, pos, padding)
		if err != nil {
			return err
		}
	}

	commentFieldBytes, err := bits.ReadBits(buf, pos, 8)
	if err != nil {
		return fmt.Errorf("unable to read program_config_element comment length: %w", err)
	}

	if commentFieldBytes > 0 {
		err = bits.HasSpace(buf, *pos, int(commentFieldBytes)*8)
		if err != nil {
			return err
		}

		e.Comments = make([]byte, commentFieldBytes)
		for i := range commentFieldBytes {
			e.Comments[i], err = readBits8(buf, pos, 8)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func readBits8(buf []byte, pos *int, n int) (uint8, error) {
	v, err := bits.ReadBits(buf, pos, n)
	return uint8(v), err
}

func readProgramConfigElementItems(
	buf []byte,
	pos *int,
	count int,
	withIsCPE bool,
) ([]ProgramConfigElementItem, error) {
	if count == 0 {
		return nil, nil
	}

	items := make([]ProgramConfigElementItem, count)

	for i := range count {
		if withIsCPE {
			isCPE, err := bits.ReadFlag(buf, pos)
			if err != nil {
				return nil, err
			}
			items[i].IsCPE = isCPE
		}

		tag, err := readBits8(buf, pos, 4)
		if err != nil {
			return nil, err
		}
		items[i].ElementTag = tag
	}

	return items, nil
}

func (e ProgramConfigElement) marshalSizeBits() int {
	return e.marshalSizeBitsAt(0)
}

// marshalSizeBitsAt returns the size in bits when the element starts at
// startPos in the surrounding bitstream. The program_config_element ends
// with a byteAlign relative to the whole bitstream, so the amount of
// padding depends on the position it is written at, not only on its own
// content.
func (e ProgramConfigElement) marshalSizeBitsAt(startPos int) int {
	// element_instance_tag(4) + profile(2) + sampling_frequency_index(4)
	// + num_front(4) + num_side(4) + num_back(4) + num_lfe(2)
	// + num_assoc(3) + num_valid_cc(3) + mono_mixdown_present(1)
	// + stereo_mixdown_present(1) + matrix_mixdown_present(1)
	n := 4 + 2 + 4 + 4 + 4 + 4 + 2 + 3 + 3 + 1 + 1 + 1

	if e.MonoMixdownPresent {
		n += 4
	}
	if e.StereoMixdownPresent {
		n += 4
	}
	if e.MatrixMixdownPresent {
		n += 3 + 1
	}

	n += programConfigElementItemsSizeBits(e.FrontElements, true)
	n += programConfigElementItemsSizeBits(e.SideElements, true)
	n += programConfigElementItemsSizeBits(e.BackElements, true)
	n += programConfigElementItemsSizeBits(e.LFEElements, false)
	n += programConfigElementItemsSizeBits(e.AssocDataElements, false)
	n += programConfigElementItemsSizeBits(e.ValidCCElements, false)

	// byteAlign relative to the surrounding bitstream
	endPos := startPos + n
	if endPos%8 != 0 {
		n += 8 - (endPos % 8)
	}

	// comment_field_bytes(8) + comments
	n += 8 + len(e.Comments)*8

	return n
}

func programConfigElementItemsSizeBits(items []ProgramConfigElementItem, withIsCPE bool) int {
	n := len(items) * 4
	if withIsCPE {
		n += len(items)
	}
	return n
}

func (e ProgramConfigElement) marshalToBits(buf []byte, pos *int) error {
	bits.WriteBitsUnsafe(buf, pos, uint64(e.ElementInstanceTag), 4)
	bits.WriteBitsUnsafe(buf, pos, uint64(e.Profile), 2)
	bits.WriteBitsUnsafe(buf, pos, uint64(e.SampleRateIndex), 4)
	bits.WriteBitsUnsafe(buf, pos, uint64(len(e.FrontElements)), 4)
	bits.WriteBitsUnsafe(buf, pos, uint64(len(e.SideElements)), 4)
	bits.WriteBitsUnsafe(buf, pos, uint64(len(e.BackElements)), 4)
	bits.WriteBitsUnsafe(buf, pos, uint64(len(e.LFEElements)), 2)
	bits.WriteBitsUnsafe(buf, pos, uint64(len(e.AssocDataElements)), 3)
	bits.WriteBitsUnsafe(buf, pos, uint64(len(e.ValidCCElements)), 3)

	bits.WriteFlagUnsafe(buf, pos, e.MonoMixdownPresent)
	if e.MonoMixdownPresent {
		bits.WriteBitsUnsafe(buf, pos, uint64(e.MonoMixdownElement), 4)
	}

	bits.WriteFlagUnsafe(buf, pos, e.StereoMixdownPresent)
	if e.StereoMixdownPresent {
		bits.WriteBitsUnsafe(buf, pos, uint64(e.StereoMixdownElement), 4)
	}

	bits.WriteFlagUnsafe(buf, pos, e.MatrixMixdownPresent)
	if e.MatrixMixdownPresent {
		bits.WriteBitsUnsafe(buf, pos, uint64(e.MatrixMixdownIndex), 3)
		bits.WriteFlagUnsafe(buf, pos, e.MatrixPseudoSurround)
	}

	writeProgramConfigElementItems(buf, pos, e.FrontElements, true)
	writeProgramConfigElementItems(buf, pos, e.SideElements, true)
	writeProgramConfigElementItems(buf, pos, e.BackElements, true)
	writeProgramConfigElementItems(buf, pos, e.LFEElements, false)
	writeProgramConfigElementItems(buf, pos, e.AssocDataElements, false)
	writeProgramConfigElementItems(buf, pos, e.ValidCCElements, false)

	// byteAlign(): zero padding up to the next byte boundary
	if (*pos % 8) != 0 {
		padding := 8 - (*pos % 8)
		bits.WriteBitsUnsafe(buf, pos, 0, padding)
	}

	bits.WriteBitsUnsafe(buf, pos, uint64(len(e.Comments)), 8)
	for _, c := range e.Comments {
		bits.WriteBitsUnsafe(buf, pos, uint64(c), 8)
	}

	return nil
}

func writeProgramConfigElementItems(
	buf []byte,
	pos *int,
	items []ProgramConfigElementItem,
	withIsCPE bool,
) {
	for _, item := range items {
		if withIsCPE {
			bits.WriteFlagUnsafe(buf, pos, item.IsCPE)
		}
		bits.WriteBitsUnsafe(buf, pos, uint64(item.ElementTag), 4)
	}
}
