package mpegts_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/asticode/go-astits"
	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/codecs"
)

func latmASC(rate int) *mpeg4audio.AudioSpecificConfig {
	return &mpeg4audio.AudioSpecificConfig{
		Type:          mpeg4audio.ObjectTypeAACLC,
		SampleRate:    rate,
		ChannelConfig: 1,
		ChannelCount:  1,
	}
}

// latmMultiProgramConfig is the anchor configuration:
// 2 programs, the first one with 2 layers, sample rates
// 24000 / 48000 (program 0) and 44100 (program 1).
func latmMultiProgramConfig() *mpeg4audio.StreamMuxConfig {
	return &mpeg4audio.StreamMuxConfig{
		Programs: []*mpeg4audio.StreamMuxConfigProgram{
			{
				Layers: []*mpeg4audio.StreamMuxConfigLayer{
					{
						AudioSpecificConfig: latmASC(24000),
						FrameLengthType:     0,
						LatmBufferFullness:  255,
					},
					{
						AudioSpecificConfig: latmASC(48000),
						FrameLengthType:     0,
						LatmBufferFullness:  100,
					},
				},
			},
			{
				Layers: []*mpeg4audio.StreamMuxConfigLayer{
					{
						AudioSpecificConfig: latmASC(44100),
						FrameLengthType:     0,
						LatmBufferFullness:  50,
					},
				},
			},
		},
	}
}

func latmMarshalElement(t *testing.T, el *mpeg4audio.AudioMuxElement) []byte {
	t.Helper()
	b, err := el.Marshal()
	require.NoError(t, err)
	return b
}

func latmLOAS(t *testing.T, els ...[]byte) []byte {
	t.Helper()
	b, err := mpeg4audio.AudioSyncStream{AudioMuxElements: els}.Marshal()
	require.NoError(t, err)
	return b
}

func TestLATMRoundTripMultiProgram(t *testing.T) {
	cfg := latmMultiProgramConfig()

	payloads := [][][][]byte{
		{
			{{0x01, 0x02, 0x03, 0x04}, {0x05, 0x06, 0x07, 0x08}},
			{{0x09, 0x0a, 0x0b, 0x0c}},
		},
	}

	// first element carries the full StreamMuxConfig
	fullEl := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
		MuxConfigPresent: true,
		UseSameStreamMux: false,
		StreamMuxConfig:  cfg,
		Payloads:         payloads,
	})

	// later elements only set useSameStreamMux and reuse the config
	sameEl1 := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
		MuxConfigPresent: true,
		UseSameStreamMux: true,
		StreamMuxConfig:  cfg,
		Payloads:         payloads,
	})

	sameEl2 := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
		MuxConfigPresent: true,
		UseSameStreamMux: true,
		StreamMuxConfig:  cfg,
		Payloads:         payloads,
	})

	track := &mpegts.Track{
		PID:   257,
		Codec: &codecs.MPEG4AudioLATM{StreamMuxConfig: cfg},
	}

	written := [][][]byte{
		{fullEl, sameEl1},
		{sameEl2},
		{fullEl},
	}

	var buf bytes.Buffer
	w := mpegts.NewWriter(&buf, []*mpegts.Track{track})

	for i, els := range written {
		err := w.WriteMPEG4AudioLATM(track, int64(i+1)*90000, els)
		require.NoError(t, err)
	}

	// read back
	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	require.NoError(t, r.Initialize())

	require.Len(t, r.Tracks(), 1)
	latmCodec, ok := r.Tracks()[0].Codec.(*codecs.MPEG4AudioLATM)
	require.True(t, ok)
	require.NotNil(t, latmCodec.StreamMuxConfig)

	gotCfg := latmCodec.StreamMuxConfig
	require.Len(t, gotCfg.Programs, 2)
	require.Len(t, gotCfg.Programs[0].Layers, 2)
	require.Len(t, gotCfg.Programs[1].Layers, 1)

	require.Equal(t, 24000, gotCfg.Programs[0].Layers[0].AudioSpecificConfig.SampleRate)
	require.Equal(t, 48000, gotCfg.Programs[0].Layers[1].AudioSpecificConfig.SampleRate)
	require.Equal(t, 44100, gotCfg.Programs[1].Layers[0].AudioSpecificConfig.SampleRate)

	// distinct buffer fullness values must not overwrite one another
	require.Equal(t, uint(255), gotCfg.Programs[0].Layers[0].LatmBufferFullness)
	require.Equal(t, uint(100), gotCfg.Programs[0].Layers[1].LatmBufferFullness)
	require.Equal(t, uint(50), gotCfg.Programs[1].Layers[0].LatmBufferFullness)

	type callback struct {
		pts int64
		els [][]byte
	}
	var callbacks []callback

	r.OnDataMPEG4AudioLATM(r.Tracks()[0], func(pts int64, els [][]byte) error {
		// the callback must hand back independent, but identical, bytes
		elsCopy := make([][]byte, len(els))
		for i, el := range els {
			elsCopy[i] = append([]byte(nil), el...)
		}
		callbacks = append(callbacks, callback{pts: pts, els: elsCopy})
		return nil
	})

	for {
		err := r.Read()
		if err != nil {
			require.ErrorIs(t, io.EOF, err)
			break
		}
	}

	// timestamps and raw audioMuxElements must be preserved exactly
	require.Len(t, callbacks, len(written))
	for i, want := range written {
		require.Equal(t, int64(i+1)*90000, callbacks[i].pts)
		require.Equal(t, want, callbacks[i].els)
	}

	// the first element must decode to the original payloads, including
	// every program and layer
	var dec mpeg4audio.AudioMuxElement
	dec.MuxConfigPresent = true
	require.NoError(t, dec.Unmarshal(callbacks[0].els[0]))
	require.False(t, dec.UseSameStreamMux)
	require.Equal(t, payloads, dec.Payloads)

	// a useSameStreamMux element must decode against the persisted config
	var decSame mpeg4audio.AudioMuxElement
	decSame.MuxConfigPresent = true
	decSame.StreamMuxConfig = gotCfg
	require.NoError(t, decSame.Unmarshal(callbacks[0].els[1]))
	require.True(t, decSame.UseSameStreamMux)
	require.Equal(t, payloads, decSame.Payloads)
}

func TestLATMInitializeErrors(t *testing.T) {
	for _, ca := range []string{
		"no loas sync",
		"only useSameStreamMux",
		"invalid AudioSpecificConfig",
	} {
		t.Run(ca, func(t *testing.T) {
			var buf bytes.Buffer
			mux := astits.NewMuxer(context.Background(), &buf)

			err := mux.AddElementaryStream(astits.PMTElementaryStream{
				ElementaryPID: 257,
				StreamType:    astits.StreamTypeAACLATMAudio,
			})
			require.NoError(t, err)
			mux.SetPCRPID(257)

			var data []byte

			switch ca {
			case "no loas sync":
				// valid length, but the LOAS syncword is never present
				data = make([]byte, 32)

			case "only useSameStreamMux":
				el := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
					MuxConfigPresent: true,
					UseSameStreamMux: true,
					StreamMuxConfig: &mpeg4audio.StreamMuxConfig{
						Programs: []*mpeg4audio.StreamMuxConfigProgram{{
							Layers: []*mpeg4audio.StreamMuxConfigLayer{{
								AudioSpecificConfig: latmASC(24000),
								FrameLengthType:     0,
								LatmBufferFullness:  255,
							}},
						}},
					},
					Payloads: [][][][]byte{{{{0x01}}}},
				})
				data = latmLOAS(t, el)

			case "invalid AudioSpecificConfig":
				badCfg := &mpeg4audio.StreamMuxConfig{
					Programs: []*mpeg4audio.StreamMuxConfigProgram{{
						Layers: []*mpeg4audio.StreamMuxConfigLayer{{
							// object type 8 is rejected by the parser
							AudioSpecificConfig: &mpeg4audio.AudioSpecificConfig{
								Type:          8,
								SampleRate:    24000,
								ChannelConfig: 1,
								ChannelCount:  1,
							},
							FrameLengthType:    0,
							LatmBufferFullness: 255,
						}},
					}},
				}
				el := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
					MuxConfigPresent: true,
					StreamMuxConfig:  badCfg,
					Payloads:         [][][][]byte{{{{0x01}}}},
				})
				data = latmLOAS(t, el)
			}

			_, err = mux.WriteData(&astits.MuxerData{
				PID: 257,
				PES: &astits.PESData{
					Header: &astits.PESHeader{
						OptionalHeader: &astits.PESOptionalHeader{
							MarkerBits:      2,
							PTSDTSIndicator: astits.PTSDTSIndicatorOnlyPTS,
							PTS:             &astits.ClockReference{Base: 90000},
						},
						StreamID: 192,
					},
					Data: data,
				},
			})
			require.NoError(t, err)

			r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
			err = r.Initialize()
			require.Error(t, err)
			require.ErrorContains(t, err, "unable to find LATM StreamMuxConfig")
		})
	}
}

func TestLATMDecodeErrors(t *testing.T) {
	cfg := &mpeg4audio.StreamMuxConfig{
		Programs: []*mpeg4audio.StreamMuxConfigProgram{{
			Layers: []*mpeg4audio.StreamMuxConfigLayer{{
				AudioSpecificConfig: latmASC(24000),
				FrameLengthType:     0,
				LatmBufferFullness:  255,
			}},
		}},
	}

	validEl := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
		MuxConfigPresent: true,
		StreamMuxConfig:  cfg,
		Payloads:         [][][][]byte{{{{0x01, 0x02}}}},
	})

	for _, ca := range []string{
		"truncated PES",
		"invalid AudioSpecificConfig",
	} {
		t.Run(ca, func(t *testing.T) {
			var buf bytes.Buffer
			mux := astits.NewMuxer(context.Background(), &buf)

			err := mux.AddElementaryStream(astits.PMTElementaryStream{
				ElementaryPID: 257,
				StreamType:    astits.StreamTypeAACLATMAudio,
			})
			require.NoError(t, err)
			mux.SetPCRPID(257)

			writePES := func(data []byte) {
				_, err := mux.WriteData(&astits.MuxerData{
					PID: 257,
					PES: &astits.PESData{
						Header: &astits.PESHeader{
							OptionalHeader: &astits.PESOptionalHeader{
								MarkerBits:      2,
								PTSDTSIndicator: astits.PTSDTSIndicatorOnlyPTS,
								PTS:             &astits.ClockReference{Base: 90000},
							},
							StreamID: 192,
						},
						Data: data,
					},
				})
				require.NoError(t, err)
			}

			// valid packet first, so that initialization succeeds
			writePES(latmLOAS(t, validEl))

			var bad []byte

			switch ca {
			case "truncated PES":
				frame := latmLOAS(t, validEl)
				// drop the last two bytes: the declared length can no
				// longer be satisfied
				bad = frame[:len(frame)-2]

			case "invalid AudioSpecificConfig":
				badCfg := &mpeg4audio.StreamMuxConfig{
					Programs: []*mpeg4audio.StreamMuxConfigProgram{{
						Layers: []*mpeg4audio.StreamMuxConfigLayer{{
							AudioSpecificConfig: &mpeg4audio.AudioSpecificConfig{
								Type:          8,
								SampleRate:    24000,
								ChannelConfig: 1,
								ChannelCount:  1,
							},
							FrameLengthType:    0,
							LatmBufferFullness: 255,
						}},
					}},
				}
				badEl := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
					MuxConfigPresent: true,
					StreamMuxConfig:  badCfg,
					Payloads:         [][][][]byte{{{{0x01}}}},
				})
				bad = latmLOAS(t, badEl)
			}

			writePES(bad)
			// a valid packet after the corruption must still be delivered
			writePES(latmLOAS(t, validEl))

			r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
			require.NoError(t, r.Initialize())

			var decodeErr error
			r.OnDecodeError(func(err error) {
				if decodeErr == nil {
					decodeErr = err
				}
			})

			delivered := 0
			r.OnDataMPEG4AudioLATM(r.Tracks()[0], func(_ int64, _ [][]byte) error {
				delivered++
				return nil
			})

			for {
				err := r.Read()
				if err != nil {
					require.ErrorIs(t, io.EOF, err)
					break
				}
			}

			require.Error(t, decodeErr)
			require.Equal(t, 2, delivered)
		})
	}
}

func TestLATMLinearScan(t *testing.T) {
	// The initialization scan performs a single forward pass: PID
	// filtering is done on the parsed data units and the stream is never
	// searched from the beginning again. Two LATM tracks interleave
	// invalid packets before each one finds its own configuration.
	const garbagePerPID = 1500

	var buf bytes.Buffer
	mux := astits.NewMuxer(context.Background(), &buf)

	for _, pid := range []uint16{257, 258} {
		err := mux.AddElementaryStream(astits.PMTElementaryStream{
			ElementaryPID: pid,
			StreamType:    astits.StreamTypeAACLATMAudio,
		})
		require.NoError(t, err)
	}
	mux.SetPCRPID(257)

	writePES := func(pid uint16, data []byte) {
		_, err := mux.WriteData(&astits.MuxerData{
			PID: pid,
			PES: &astits.PESData{
				Header: &astits.PESHeader{
					OptionalHeader: &astits.PESOptionalHeader{
						MarkerBits:      2,
						PTSDTSIndicator: astits.PTSDTSIndicatorOnlyPTS,
						PTS:             &astits.ClockReference{Base: 90000},
					},
					StreamID: 192,
				},
				Data: data,
			},
		})
		require.NoError(t, err)
	}

	garbage := make([]byte, 16)
	for range garbagePerPID {
		writePES(257, garbage)
		writePES(258, garbage)
	}

	cfg24 := &mpeg4audio.StreamMuxConfig{
		Programs: []*mpeg4audio.StreamMuxConfigProgram{{
			Layers: []*mpeg4audio.StreamMuxConfigLayer{{
				AudioSpecificConfig: latmASC(24000),
				FrameLengthType:     0,
				LatmBufferFullness:  255,
			}},
		}},
	}
	cfg48 := &mpeg4audio.StreamMuxConfig{
		Programs: []*mpeg4audio.StreamMuxConfigProgram{{
			Layers: []*mpeg4audio.StreamMuxConfigLayer{{
				AudioSpecificConfig: latmASC(48000),
				FrameLengthType:     0,
				LatmBufferFullness:  255,
			}},
		}},
	}

	// track 257 becomes valid while track 258 still contains garbage:
	// the scan must keep going without restarting from the beginning
	for range 500 {
		writePES(258, garbage)
	}

	el24 := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
		MuxConfigPresent: true,
		StreamMuxConfig:  cfg24,
		Payloads:         [][][][]byte{{{{0x01}}}},
	})
	writePES(257, latmLOAS(t, el24))

	for range 500 {
		writePES(257, garbage)
	}

	el48 := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
		MuxConfigPresent: true,
		StreamMuxConfig:  cfg48,
		Payloads:         [][][][]byte{{{{0x02}}}},
	})
	writePES(258, latmLOAS(t, el48))

	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	require.NoError(t, r.Initialize())

	require.Len(t, r.Tracks(), 2)

	rates := map[uint16]int{}
	for _, tr := range r.Tracks() {
		c, ok := tr.Codec.(*codecs.MPEG4AudioLATM)
		require.True(t, ok)
		require.NotNil(t, c.StreamMuxConfig)
		rates[tr.PID] = c.StreamMuxConfig.Programs[0].Layers[0].AudioSpecificConfig.SampleRate
	}
	require.Equal(t, 24000, rates[257])
	require.Equal(t, 48000, rates[258])
}

func TestLATMReadPathLinearScan(t *testing.T) {
	// After initialization, reading well above 6000 TS packets must stay
	// linear: the per-track StreamMuxConfig is reused for every
	// useSameStreamMux element and PIDs are resolved through a map lookup.
	cfg := &mpeg4audio.StreamMuxConfig{
		Programs: []*mpeg4audio.StreamMuxConfigProgram{{
			Layers: []*mpeg4audio.StreamMuxConfigLayer{{
				AudioSpecificConfig: latmASC(24000),
				FrameLengthType:     0,
				LatmBufferFullness:  255,
			}},
		}},
	}

	track := &mpegts.Track{
		PID:   257,
		Codec: &codecs.MPEG4AudioLATM{StreamMuxConfig: cfg},
	}

	fullEl := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
		MuxConfigPresent: true,
		StreamMuxConfig:  cfg,
		Payloads:         [][][][]byte{{{{0x01}}}},
	})
	sameEl := latmMarshalElement(t, &mpeg4audio.AudioMuxElement{
		MuxConfigPresent: true,
		UseSameStreamMux: true,
		StreamMuxConfig:  cfg,
		Payloads:         [][][][]byte{{{{0x02}}}},
	})

	var buf bytes.Buffer
	w := mpegts.NewWriter(&buf, []*mpegts.Track{track})

	const pesCount = 2100 // ~3 TS packets each -> well over 6000 packets
	require.NoError(t, w.WriteMPEG4AudioLATM(track, 0, [][]byte{fullEl}))
	for i := range pesCount - 1 {
		require.NoError(t, w.WriteMPEG4AudioLATM(track, int64(i+1)*90000, [][]byte{sameEl}))
	}

	require.GreaterOrEqual(t, buf.Len()/188, 6000)

	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	require.NoError(t, r.Initialize())

	delivered := 0
	r.OnDataMPEG4AudioLATM(r.Tracks()[0], func(_ int64, els [][]byte) error {
		require.Len(t, els, 1)
		delivered++
		return nil
	})

	for {
		err := r.Read()
		if err != nil {
			require.ErrorIs(t, io.EOF, err)
			break
		}
	}

	require.Equal(t, pesCount, delivered)
}
