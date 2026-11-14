package mpeg4audio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var streamMuxConfigCases = []struct {
	name string
	enc  []byte
	dec  StreamMuxConfig
}{
	{
		"lc",
		[]byte{0x40, 0x00, 0x26, 0x20, 0x3f, 0xc0},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   24000,
						ChannelCount: 2,
					},
					LatmBufferFullness: 255,
				}},
			}},
		},
	},
	{
		"he-aac",
		[]byte{0x40, 0x00, 0x26, 0x10, 0x3f, 0xc0},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   24000,
						ChannelCount: 1,
					},
					LatmBufferFullness: 255,
				}},
			}},
		},
	},
	{
		"sbr",
		[]byte{0x40, 0x00, 0x56, 0x23, 0x10, 0x1f, 0xe0},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:                2,
						ExtensionType:       5,
						ExtensionSampleRate: 48000,
						SampleRate:          24000,
						ChannelCount:        2,
					},
					LatmBufferFullness: 255,
				}},
			}},
		},
	},
	{
		"ps",
		[]byte{0x40, 0x01, 0xd6, 0x13, 0x10, 0x1f, 0xe0},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:                2,
						ExtensionType:       29,
						ExtensionSampleRate: 48000,
						SampleRate:          24000,
						ChannelCount:        1,
					},
					LatmBufferFullness: 255,
				}},
			}},
		},
	},
	{
		"other data and checksum",
		[]byte{0x40, 0x00, 0x24, 0x10, 0xad, 0xca, 0x00},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   44100,
						ChannelCount: 1,
					},
					FrameLengthType: 2,
				}},
			}},
			OtherDataPresent: true,
			OtherDataLenBits: 220,
			CRCCheckPresent:  true,
			CRCCheckSum:      64,
		},
	},
	{
		"other data > 256 and checksum",
		[]byte{0x40, 0x00, 0x24, 0x10, 0xb0, 0x33, 0x85, 0x00},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   44100,
						ChannelCount: 1,
					},
					FrameLengthType: 2,
				}},
			}},
			OtherDataPresent: true,
			OtherDataLenBits: 880,
			CRCCheckPresent:  true,
			CRCCheckSum:      64,
		},
	},
	{
		"other data without checksum",
		[]byte{0x40, 0x00, 0x24, 0x10, 0xad, 0xc0},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   44100,
						ChannelCount: 1,
					},
					FrameLengthType: 2,
				}},
			}},
			OtherDataPresent: true,
			OtherDataLenBits: 220,
		},
	},
	{
		"checksum without other data",
		[]byte{0x40, 0x00, 0x24, 0x10, 0x94, 0x00},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   44100,
						ChannelCount: 1,
					},
					FrameLengthType: 2,
				}},
			}},
			CRCCheckPresent: true,
			CRCCheckSum:     64,
		},
	},
	{
		"no other data nor checksum",
		[]byte{0x40, 0x00, 0x24, 0x10, 0x80},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   44100,
						ChannelCount: 1,
					},
					FrameLengthType: 2,
				}},
			}},
		},
	},
	{
		"other data length with three bytes",
		[]byte{0x40, 0x00, 0x24, 0x10, 0xb0, 0x19, 0x19, 0x16, 0x06},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   44100,
						ChannelCount: 1,
					},
					FrameLengthType: 2,
				}},
			}},
			OtherDataPresent: true,
			OtherDataLenBits: 74565,
			CRCCheckPresent:  true,
			CRCCheckSum:      3,
		},
	},
}

func TestStreamMuxConfigUnmarshal(t *testing.T) {
	for _, ca := range streamMuxConfigCases {
		t.Run(ca.name, func(t *testing.T) {
			var dec StreamMuxConfig
			err := dec.Unmarshal(ca.enc)
			require.NoError(t, err)
			require.Equal(t, ca.dec, dec)
		})
	}
}

func TestStreamMuxConfigUnmarshalTruncated(t *testing.T) {
	var dec StreamMuxConfig
	err := dec.Unmarshal([]byte{0x40, 0x00, 0x23, 0x10})
	require.NoError(t, err)
	require.Equal(t, StreamMuxConfig{
		Programs: []*StreamMuxConfigProgram{{
			Layers: []*StreamMuxConfigLayer{{
				AudioSpecificConfig: &AudioSpecificConfig{
					Type:         2,
					SampleRate:   48000,
					ChannelCount: 1,
				},
				LatmBufferFullness: 255,
			}},
		}},
	}, dec)
}

func TestStreamMuxConfigUnmarshalTruncatedSuffix(t *testing.T) {
	for _, ca := range []struct {
		name string
		enc  []byte
	}{
		{"missing other data length", []byte{0x40, 0x00, 0x24, 0x10, 0xad}},
		{"incomplete other data length", []byte{0x40, 0x00, 0x24, 0x10, 0xb0, 0x33}},
		{"missing checksum", []byte{0x40, 0x00, 0x24, 0x10, 0xad, 0xca}},
		{"incomplete checksum", []byte{0x40, 0x00, 0x24, 0x10, 0xb0, 0x33, 0x85}},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var dec StreamMuxConfig
			err := dec.Unmarshal(ca.enc)
			require.Error(t, err)
		})
	}
}

func TestStreamMuxConfigRoundTrip(t *testing.T) {
	for _, ca := range streamMuxConfigCases {
		t.Run(ca.name, func(t *testing.T) {
			var dec StreamMuxConfig
			err := dec.Unmarshal(ca.enc)
			require.NoError(t, err)

			enc, err := dec.Marshal()
			require.NoError(t, err)
			require.Equal(t, ca.enc, enc)

			err = dec.Unmarshal(enc)
			require.NoError(t, err)
			require.Equal(t, ca.dec, dec)
		})
	}
}

func TestStreamMuxConfigRoundTripRepeated(t *testing.T) {
	for _, ca := range streamMuxConfigCases {
		t.Run(ca.name, func(t *testing.T) {
			var dec StreamMuxConfig
			var enc []byte

			for i := 0; i < 1000; i++ {
				err := dec.Unmarshal(ca.enc)
				require.NoError(t, err)

				enc, err = dec.Marshal()
				require.NoError(t, err)
				require.Equal(t, ca.enc, enc)
			}
		})
	}
}

func TestStreamMuxConfigUnmarshalReusedDecoder(t *testing.T) {
	var dec StreamMuxConfig

	for i := 0; i < 1000; i++ {
		for _, ca := range streamMuxConfigCases {
			err := dec.Unmarshal(ca.enc)
			require.NoError(t, err)
			require.Equal(t, ca.dec, dec)

			enc, err := dec.Marshal()
			require.NoError(t, err)
			require.Equal(t, ca.enc, enc)
		}
	}
}

func TestStreamMuxConfigMarshal(t *testing.T) {
	for _, ca := range streamMuxConfigCases {
		t.Run(ca.name, func(t *testing.T) {
			enc, err := ca.dec.Marshal()
			require.NoError(t, err)
			require.Equal(t, ca.enc, enc)
		})
	}
}

func FuzzStreamMuxConfigUnmarshal(f *testing.F) {
	for _, ca := range streamMuxConfigCases {
		f.Add(ca.enc)
	}

	f.Fuzz(func(t *testing.T, b []byte) {
		var conf StreamMuxConfig
		conf.Unmarshal(b)
	})
}
