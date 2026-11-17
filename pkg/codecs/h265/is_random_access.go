package h265

// IsRandomAccess checks whether the access unit can be randomly accessed.
// AU is assumed to contain byte slices each with at least 1 byte.
func IsRandomAccess(au [][]byte) bool {
	for _, nalu := range au {
		if len(nalu) == 0 {
			continue
		}

		typ := NALUType((nalu[0] >> 1) & 0b111111)
		switch typ {
		case NALUType_IDR_W_RADL, NALUType_IDR_N_LP, NALUType_CRA_NUT:
			return true
		}
	}
	return false
}
