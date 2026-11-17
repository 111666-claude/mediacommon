package h265

import (
	"bytes"
	"errors"
	"fmt"
)

// Errors returned by AccessUnitChecker.Check.
var (
	// ErrEmptyAccessUnit is returned when the access unit doesn't contain any NALU.
	ErrEmptyAccessUnit = errors.New("access unit is empty")

	// ErrTruncatedNALU is returned when a NALU is shorter than the H265 NALU header.
	ErrTruncatedNALU = errors.New("truncated NALU")

	// ErrContradictoryParameterSets is returned when an access unit carries multiple
	// different parameter sets of the same type.
	ErrContradictoryParameterSets = errors.New("access unit contains contradictory parameter sets")

	// ErrInvalidParameterSet is returned when a parameter set can't be parsed.
	ErrInvalidParameterSet = errors.New("invalid parameter set")
)

// RandomAccessType is the random access type of an access unit.
// Specification: ITU-T Rec. H.265, section 7.4.2.2.
type RandomAccessType uint8

// Random access types, from the weakest to the strongest one.
// When an access unit contains multiple types, the strongest one is reported.
const (
	// RandomAccessTypeNone indicates that the access unit is not a random access point.
	RandomAccessTypeNone RandomAccessType = iota

	// RandomAccessTypeCRA indicates a Clean Random Access (CRA) picture.
	RandomAccessTypeCRA

	// RandomAccessTypeBLA indicates a Broken Link Access (BLA) picture.
	RandomAccessTypeBLA

	// RandomAccessTypeIDR indicates an Instantaneous Decoding Refresh (IDR) picture.
	RandomAccessTypeIDR
)

// String implements fmt.Stringer.
func (t RandomAccessType) String() string {
	switch t {
	case RandomAccessTypeNone:
		return "none"

	case RandomAccessTypeCRA:
		return "CRA"

	case RandomAccessTypeBLA:
		return "BLA"

	case RandomAccessTypeIDR:
		return "IDR"
	}

	return fmt.Sprintf("unknown (%d)", t)
}

func randomAccessTypeOfNALUType(typ NALUType) RandomAccessType {
	switch typ {
	case NALUType_IDR_W_RADL, NALUType_IDR_N_LP:
		return RandomAccessTypeIDR

	case NALUType_BLA_W_LP, NALUType_BLA_W_RADL, NALUType_BLA_N_LP:
		return RandomAccessTypeBLA

	case NALUType_CRA_NUT, NALUType_RSV_IRAP_VCL22, NALUType_RSV_IRAP_VCL23:
		// NALU types 16-23 are all IRAP types (ITU-T Rec. H.265, Table 7-1).
		// The reserved IRAP types are reported as the weakest IRAP type.
		return RandomAccessTypeCRA
	}

	return RandomAccessTypeNone
}

// AccessUnitInfo contains the information about an access unit
// computed by AccessUnitChecker.
type AccessUnitInfo struct {
	// HasVCL reports whether the access unit contains at least one VCL NALU,
	// hence a coded picture. Access units that only contain non-VCL NALUs
	// (AUD, SEI, parameter sets, ...) have this flag set to false.
	HasVCL bool

	// RandomAccessType is the random access type of the access unit.
	RandomAccessType RandomAccessType

	// VPS, SPS and PPS are the parameter sets carried by the access unit, or nil when absent.
	// They point to the buffers given to AccessUnitChecker.Check and must not be modified.
	VPS []byte
	SPS []byte
	PPS []byte

	// VPSChanged, SPSChanged and PPSChanged report whether the access unit carries a
	// parameter set whose content differs from the last valid one of the same type.
	// The first occurrence of a parameter set is reported as a change.
	VPSChanged bool
	SPSChanged bool
	PPSChanged bool

	// MissingParameterSets reports whether the access unit contains a coded picture but,
	// after processing it, the checker still doesn't know all the parameter sets (VPS, SPS
	// and PPS). This happens when a slice precedes the parameter sets that configure it and
	// no previous access unit provided them, or when only part of them is available.
	// Such an access unit is reported, not rejected, since a following access unit can
	// still complete the configuration.
	MissingParameterSets bool
}

// IsRandomAccess reports whether the access unit is a random access point.
func (i AccessUnitInfo) IsRandomAccess() bool {
	return i.RandomAccessType != RandomAccessTypeNone
}

// ParameterSetsChanged reports whether the access unit carries at least one
// parameter set whose content changed.
func (i AccessUnitInfo) ParameterSetsChanged() bool {
	return i.VPSChanged || i.SPSChanged || i.PPSChanged
}

// AccessUnitChecker inspects the NALUs of the access units of an H265 stream.
//
// It reports whether an access unit contains a coded picture, its random access type and
// the parameter sets carried by it. Moreover, it keeps the last valid VPS, SPS and PPS
// seen across access units and reports when their content changes, which is required to
// write the index of a recorder.
//
// The zero value is ready to use. AccessUnitChecker is not safe for concurrent use.
type AccessUnitChecker struct {
	vps []byte
	sps []byte
	pps []byte
}

// Initialize resets the checker, clearing the stored parameter sets.
func (c *AccessUnitChecker) Initialize() {
	c.vps = nil
	c.sps = nil
	c.pps = nil
}

// VPS returns the last valid VPS received, or nil.
func (c *AccessUnitChecker) VPS() []byte {
	return c.vps
}

// SPS returns the last valid SPS received, or nil.
func (c *AccessUnitChecker) SPS() []byte {
	return c.sps
}

// PPS returns the last valid PPS received, or nil.
func (c *AccessUnitChecker) PPS() []byte {
	return c.pps
}

func mergeParameterSet(dst *[]byte, nalu []byte, name string) error {
	if *dst == nil {
		*dst = nalu
		return nil
	}

	if !bytes.Equal(*dst, nalu) {
		return fmt.Errorf("%w: multiple different %s NALUs", ErrContradictoryParameterSets, name)
	}

	return nil
}

// Check inspects the NALUs of an access unit.
//
// NALUs shorter than the 2-byte H265 NALU header are rejected as truncated.
// An access unit that carries multiple different parameter sets of the same type is
// rejected as contradictory; identical duplicates are accepted.
// Parameter sets are parsed before being stored, so an invalid SPS or PPS rejects the
// whole access unit. A rejected access unit doesn't change the state of the checker.
//
// Parameter sets can appear anywhere inside the access unit, including after the slices
// that reference them. Slices received before any parameter set are not an error: the
// access unit is reported with HasVCL and RandomAccessType set, while the parameter sets
// and the change flags are empty. When the checker doesn't know all the parameter sets
// after processing the access unit, MissingParameterSets is set.
func (c *AccessUnitChecker) Check(au [][]byte) (AccessUnitInfo, error) {
	if len(au) == 0 {
		return AccessUnitInfo{}, ErrEmptyAccessUnit
	}

	var info AccessUnitInfo
	var vps []byte
	var sps []byte
	var pps []byte

	for i, nalu := range au {
		if len(nalu) < 2 {
			return AccessUnitInfo{}, fmt.Errorf("%w: NALU %d has %d bytes, minimum is 2",
				ErrTruncatedNALU, i, len(nalu))
		}

		typ := NALUType((nalu[0] >> 1) & 0b111111)

		switch typ {
		case NALUType_VPS_NUT:
			if err := mergeParameterSet(&vps, nalu, "VPS"); err != nil {
				return AccessUnitInfo{}, err
			}

		case NALUType_SPS_NUT:
			if err := mergeParameterSet(&sps, nalu, "SPS"); err != nil {
				return AccessUnitInfo{}, err
			}

		case NALUType_PPS_NUT:
			if err := mergeParameterSet(&pps, nalu, "PPS"); err != nil {
				return AccessUnitInfo{}, err
			}

		default:
			if typ.IsVCL() {
				info.HasVCL = true

				if rat := randomAccessTypeOfNALUType(typ); rat > info.RandomAccessType {
					info.RandomAccessType = rat
				}
			}
		}
	}

	// validate the parameter sets that differ from the stored ones before committing
	// anything, so that an invalid access unit cannot corrupt the state.
	if vps != nil && !bytes.Equal(vps, c.vps) && len(vps) <= 2 {
		// there is no VPS parser, but a VPS that only contains the 2-byte NALU header
		// has no vps_video_parameter_set_id and is structurally invalid.
		return AccessUnitInfo{}, fmt.Errorf("%w: VPS: not enough bits", ErrInvalidParameterSet)
	}

	if sps != nil && !bytes.Equal(sps, c.sps) {
		var parsed SPS
		if err := parsed.Unmarshal(sps); err != nil {
			return AccessUnitInfo{}, fmt.Errorf("%w: SPS: %w", ErrInvalidParameterSet, err)
		}
	}

	if pps != nil && !bytes.Equal(pps, c.pps) {
		var parsed PPS
		if err := parsed.Unmarshal(pps); err != nil {
			return AccessUnitInfo{}, fmt.Errorf("%w: PPS: %w", ErrInvalidParameterSet, err)
		}
	}

	info.VPS = vps
	info.SPS = sps
	info.PPS = pps
	info.VPSChanged = vps != nil && !bytes.Equal(vps, c.vps)
	info.SPSChanged = sps != nil && !bytes.Equal(sps, c.sps)
	info.PPSChanged = pps != nil && !bytes.Equal(pps, c.pps)

	if info.VPSChanged {
		c.vps = bytes.Clone(vps)
	}
	if info.SPSChanged {
		c.sps = bytes.Clone(sps)
	}
	if info.PPSChanged {
		c.pps = bytes.Clone(pps)
	}

	// a picture whose parameter sets are still unknown can't be decoded on its own.
	info.MissingParameterSets = info.HasVCL &&
		(c.vps == nil || c.sps == nil || c.pps == nil)

	return info, nil
}
