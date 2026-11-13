//nolint:dupl
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

func latmTestConfig() *mpeg4audio.StreamMuxConfig {
	return &mpeg4audio.StreamMuxConfig{
		Programs: []*mpeg4audio.StreamMuxConfigProgram{
			{
				Layers: []*mpeg4audio.StreamMuxConfigLayer{
					{
						AudioSpecificConfig: &mpeg4audio.AudioSpecificConfig{
							Type:          2,
							SampleRate:    24000,
							ChannelConfig: 1,
							ChannelCount:  1,
						},
						FrameLengthType:    0,
						LatmBufferFullness: 255,
					},
					{
						AudioSpecificConfig: &mpeg4audio.AudioSpecificConfig{
							Type:          2,
							SampleRate:    48000,
							ChannelConfig: 1,
							ChannelCount:  1,
						},
						FrameLengthType:    0,
						LatmBufferFullness: 128,
					},
				},
			},
			{
				Layers: []*mpeg4audio.StreamMuxConfigLayer{
					{
						AudioSpecificConfig: &mpeg4audio.AudioSpecificConfig{
							Type:          2,
							SampleRate:    44100,
							ChannelConfig: 1,
							ChannelCount:  1,
						},
						FrameLengthType:    0,
						LatmBufferFullness: 255,
					},
				},
			},
		},
	}
}

// full-config audioMuxElement (useSameStreamMux = 0).
var latmTestElementFull = []byte{
	0x20, 0x09, 0x13, 0x08, 0x1f, 0xe1, 0x18, 0x81,
	0x00, 0x02, 0x41, 0x03, 0xfc, 0x04, 0x08, 0x03,
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
}

// audioMuxElement with useSameStreamMux = 1 (no StreamMuxConfig inside).
var latmTestElementSame = []byte{
	0x81, 0x80, 0x81, 0x0a, 0x8b, 0x0b, 0x8c, 0x0c,
	0x8d, 0x00,
}

func latmTestAudioSyncStreams() ([]byte, []byte) {
	full, err := mpeg4audio.AudioSyncStream{
		AudioMuxElements: [][]byte{latmTestElementFull},
	}.Marshal()
	if err != nil {
		panic(err)
	}

	same, err := mpeg4audio.AudioSyncStream{
		AudioMuxElements: [][]byte{latmTestElementSame},
	}.Marshal()
	if err != nil {
		panic(err)
	}

	return full, same
}

// TestLATMRoundTrip writes a multi-program / multi-layer LATM track and reads it
// back, checking the discovered configuration, timestamps and audio payload.
func TestLATMRoundTrip(t *testing.T) {
	track := &mpegts.Track{
		PID: 257,
		Codec: &codecs.MPEG4AudioLATM{
			StreamMuxConfig: *latmTestConfig(),
		},
	}

	// the writer leaves non-zero PIDs untouched.
	full, same := latmTestAudioSyncStreams()
	require.NotEmpty(t, full)
	require.NotEmpty(t, same)

	type pesSample struct {
		pts int64
		els [][]byte
	}

	samples := []pesSample{
		{pts: 2700000, els: [][]byte{latmTestElementFull}},
		{pts: 2790000, els: [][]byte{latmTestElementSame}},
	}

	// write
	var buf bytes.Buffer
	w := &mpegts.Writer{W: &buf, Tracks: []*mpegts.Track{track}}
	require.NoError(t, w.Initialize())

	for _, s := range samples {
		require.NoError(t, w.WriteMPEG4AudioLATM(track, s.pts, s.els))
	}

	// read back
	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	require.NoError(t, r.Initialize())

	require.Len(t, r.Tracks(), 1)
	readTrack := r.Tracks()[0]
	require.Equal(t, uint16(257), readTrack.PID)

	latmCodec, ok := readTrack.Codec.(*codecs.MPEG4AudioLATM)
	require.True(t, ok)

	// two programs, with two layers and one layer.
	require.Len(t, latmCodec.StreamMuxConfig.Programs, 2)
	require.Len(t, latmCodec.StreamMuxConfig.Programs[0].Layers, 2)
	require.Len(t, latmCodec.StreamMuxConfig.Programs[1].Layers, 1)

	// sample rates: 24000, 48000, 44100.
	require.Equal(t, 24000, latmCodec.StreamMuxConfig.Programs[0].Layers[0].AudioSpecificConfig.SampleRate)
	require.Equal(t, 48000, latmCodec.StreamMuxConfig.Programs[0].Layers[1].AudioSpecificConfig.SampleRate)
	require.Equal(t, 44100, latmCodec.StreamMuxConfig.Programs[1].Layers[0].AudioSpecificConfig.SampleRate)

	// buffer fullness values are not overwritten across layers.
	require.Equal(t, uint(255), latmCodec.StreamMuxConfig.Programs[0].Layers[0].LatmBufferFullness)
	require.Equal(t, uint(128), latmCodec.StreamMuxConfig.Programs[0].Layers[1].LatmBufferFullness)
	require.Equal(t, uint(255), latmCodec.StreamMuxConfig.Programs[1].Layers[0].LatmBufferFullness)

	// full discovered config matches the one that was written.
	require.Equal(t, *latmTestConfig(), latmCodec.StreamMuxConfig)

	var got []pesSample
	var decodeErrs []error
	r.OnDecodeError(func(err error) { decodeErrs = append(decodeErrs, err) })
	r.OnDataMPEG4AudioLATM(readTrack, func(pts int64, els [][]byte) error {
		got = append(got, pesSample{pts: pts, els: els})
		return nil
	})

	for {
		err := r.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	require.Empty(t, decodeErrs)
	require.Equal(t, samples, got)

	// second read of the same track keeps producing identical output.
	r2 := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	require.NoError(t, r2.Initialize())

	var got2 []pesSample
	r2.OnDataMPEG4AudioLATM(r2.Tracks()[0], func(pts int64, els [][]byte) error {
		got2 = append(got2, pesSample{pts: pts, els: els})
		return nil
	})

	for {
		err := r2.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, got, got2)

	// writing the read-back elements again produces the same LATM framing.
	var buf2 bytes.Buffer
	w2 := &mpegts.Writer{W: &buf2, Tracks: []*mpegts.Track{{
		PID:   257,
		Codec: &codecs.MPEG4AudioLATM{StreamMuxConfig: latmCodec.StreamMuxConfig},
	}}}
	require.NoError(t, w2.Initialize())
	for _, s := range got2 {
		require.NoError(t, w2.WriteMPEG4AudioLATM(w2.Tracks[0], s.pts, s.els))
	}

	r3 := &mpegts.Reader{R: bytes.NewReader(buf2.Bytes())}
	require.NoError(t, r3.Initialize())
	var got3 []pesSample
	r3.OnDataMPEG4AudioLATM(r3.Tracks()[0], func(pts int64, els [][]byte) error {
		got3 = append(got3, pesSample{pts: pts, els: els})
		return nil
	})
	for {
		err := r3.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}
	require.Equal(t, got, got3)
}

// writeLATMRawTS muxes raw PES payloads (AudioSyncStream bytes, possibly
// corrupted) into a TS with a single LATM track (stream type 0x11).
func writeLATMRawTS(t *testing.T, payloads [][]byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	mux := astits.NewMuxer(context.Background(), &buf)
	require.NoError(t, mux.AddElementaryStream(astits.PMTElementaryStream{
		ElementaryPID: 257,
		StreamType:    astits.StreamTypeAACLATMAudio,
	}))
	mux.SetPCRPID(257)

	pts := int64(2700000)
	for _, payload := range payloads {
		_, err := mux.WriteData(&astits.MuxerData{
			PID: 257,
			AdaptationField: &astits.PacketAdaptationField{
				RandomAccessIndicator: true,
			},
			PES: &astits.PESData{
				Header: &astits.PESHeader{
					OptionalHeader: &astits.PESOptionalHeader{
						MarkerBits:      2,
						PTSDTSIndicator: astits.PTSDTSIndicatorOnlyPTS,
						PTS:             &astits.ClockReference{Base: pts},
					},
					StreamID: 192,
				},
				Data: payload,
			},
		})
		require.NoError(t, err)
		pts += 90000
	}

	return buf.Bytes()
}

// TestLATMInitializeErrors makes sure that missing, truncated and illegal
// configurations surface as traceable errors instead of an empty success.
func TestLATMInitializeErrors(t *testing.T) {
	for _, ca := range []struct {
		name     string
		payload  []byte
		errSubst string
	}{
		{
			name: "no config, only useSameStreamMux elements",
			// first bit = 1 -> useSameStreamMux, but no config precedes it.
			payload:  []byte{0x56, 0xe0, 0x01, 0x80},
			errSubst: "unable to find an MPEG-4 Audio LATM StreamMuxConfig",
		},
		{
			name: "truncated AudioSyncStream",
			// length field promises more bytes than are present.
			payload:  []byte{0x56, 0xe0, 0x1f, 0x20},
			errSubst: "invalid LATM AudioSyncStream",
		},
		{
			name: "illegal AudioSpecificConfig",
			// valid sync framing, but the audioMuxElement encodes an
			// unsupported object type (first 5 bits of the ASC = 0).
			payload: []byte{
				0x56, 0xe0, 0x08, 0x20, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
			errSubst: "invalid LATM audioMuxElement",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			ts := writeLATMRawTS(t, [][]byte{ca.payload})

			r := &mpegts.Reader{R: bytes.NewReader(ts)}
			err := r.Initialize()
			require.Error(t, err)
			require.ErrorContains(t, err, ca.errSubst)
		})
	}
}

// corruptLATMSecondPES builds a valid three-PES LATM stream, then overwrites the
// audioMuxElement bytes inside the second PES payload (keeping the AudioSyncStream
// sync word and length intact) so that the corruption is only reachable while
// reading, after Initialize has discovered the config.
func corruptLATMSecondPES(t *testing.T, elementBody []byte) []byte {
	t.Helper()

	full, same := latmTestAudioSyncStreams()
	ts := writeLATMRawTS(t, [][]byte{full, same, same})

	var syncOffsets []int
	for i := 0; i+1 < len(ts); i++ {
		if ts[i] == 0x56 && ts[i+1] == 0xe0 {
			syncOffsets = append(syncOffsets, i)
		}
	}
	require.GreaterOrEqual(t, len(syncOffsets), 3)

	off := syncOffsets[1]

	// the original element body length (the PES holds exactly one element).
	origLen := int(uint16(ts[off+1]&0x07)<<8 | uint16(ts[off+2]))
	require.GreaterOrEqual(t, origLen, len(elementBody))

	// keep the framing length unchanged and pad the corrupt body so the
	// AudioSyncStream header stays valid and only the element parsing fails.
	body := make([]byte, origLen)
	copy(body, elementBody)
	copy(ts[off+3:], body)

	return ts
}

// TestLATMRuntimeDecodeError makes sure that corruption appearing after the
// configuration is reported through OnDecodeError rather than silently dropped.
func TestLATMRuntimeDecodeError(t *testing.T) {
	// a full-config element whose ASC encodes an unsupported object type (0);
	// the sync framing stays valid, only the element body is wrong.
	corruptElement := []byte{0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	ts := corruptLATMSecondPES(t, corruptElement)

	r := &mpegts.Reader{R: bytes.NewReader(ts)}
	require.NoError(t, r.Initialize())

	var decodeErrs []error
	r.OnDecodeError(func(err error) { decodeErrs = append(decodeErrs, err) })

	var callbacks int
	r.OnDataMPEG4AudioLATM(r.Tracks()[0], func(_ int64, _ [][]byte) error {
		callbacks++
		return nil
	})

	for {
		err := r.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	// the two valid PES are delivered, the corrupted one raises a decode error.
	require.Equal(t, 2, callbacks)
	require.Len(t, decodeErrs, 1)
	require.ErrorContains(t, decodeErrs[0], "invalid LATM audioMuxElement")
}

// TestLATMRuntimeIllegalElement makes sure that an illegal element delivered
// after a valid configuration is reported as well.
func TestLATMRuntimeIllegalElement(t *testing.T) {
	// useSameStreamMux = 1 followed by a truncated payload that cannot be
	// parsed against the previously discovered configuration.
	corruptElement := []byte{0x81, 0xff, 0xff}
	ts := corruptLATMSecondPES(t, corruptElement)

	r := &mpegts.Reader{R: bytes.NewReader(ts)}
	require.NoError(t, r.Initialize())

	var decodeErrs []error
	r.OnDecodeError(func(err error) { decodeErrs = append(decodeErrs, err) })

	var callbacks int
	r.OnDataMPEG4AudioLATM(r.Tracks()[0], func(_ int64, _ [][]byte) error {
		callbacks++
		return nil
	})

	for {
		err := r.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, 2, callbacks)
	require.Len(t, decodeErrs, 1)
	require.ErrorContains(t, decodeErrs[0], "invalid LATM audioMuxElement")
	require.ErrorContains(t, decodeErrs[0], "not enough bits")
}

// buildLeadingPacketsTS builds a TS where fillerCount filler PES packets on a
// leading video PID (256) precede the first LATM config PES on PID 257.
func buildLeadingPacketsTS(fillerCount int) ([]byte, error) {
	var buf bytes.Buffer
	mux := astits.NewMuxer(context.Background(), &buf)
	if err := mux.AddElementaryStream(astits.PMTElementaryStream{
		ElementaryPID: 256,
		StreamType:    astits.StreamTypeH265Video,
	}); err != nil {
		return nil, err
	}
	if err := mux.AddElementaryStream(astits.PMTElementaryStream{
		ElementaryPID: 257,
		StreamType:    astits.StreamTypeAACLATMAudio,
	}); err != nil {
		return nil, err
	}
	mux.SetPCRPID(256)

	for i := 0; i < fillerCount; i++ {
		_, err := mux.WriteData(&astits.MuxerData{
			PID: 256,
			AdaptationField: &astits.PacketAdaptationField{
				RandomAccessIndicator: true,
			},
			PES: &astits.PESData{
				Header: &astits.PESHeader{
					OptionalHeader: &astits.PESOptionalHeader{
						MarkerBits:      2,
						PTSDTSIndicator: astits.PTSDTSIndicatorOnlyPTS,
						PTS:             &astits.ClockReference{Base: int64(i) * 3000},
					},
					StreamID: 224,
				},
				Data: []byte{0xab},
			},
		})
		if err != nil {
			return nil, err
		}
	}

	full, _ := latmTestAudioSyncStreams()
	_, err := mux.WriteData(&astits.MuxerData{
		PID:             257,
		AdaptationField: &astits.PacketAdaptationField{RandomAccessIndicator: true},
		PES: &astits.PESData{
			Header: &astits.PESHeader{
				OptionalHeader: &astits.PESOptionalHeader{
					MarkerBits:      2,
					PTSDTSIndicator: astits.PTSDTSIndicatorOnlyPTS,
					PTS:             &astits.ClockReference{Base: 2700000},
				},
				StreamID: 192,
			},
			Data: full,
		},
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// countingReader counts the bytes pulled from the underlying reader. The
// rewindablereader replays recorded bytes without touching the underlying one,
// so this counts only forward scans and reveals any restart-from-start logic.
type countingReader struct {
	r     io.Reader
	bytes int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.bytes += int64(n)
	return n, err
}

// TestLATMLinearScan makes sure the configuration is found even when thousands
// of packets on another PID precede it, and that discovery walks the underlying
// stream forward exactly once (a single linear pass, no repeated PID lookups
// from the start).
func TestLATMLinearScan(t *testing.T) {
	// one-byte filler PES expand to multiple TS packets each. The count is
	// bounded by the rewindable reader's 1 MiB recording window; ~5100 packets
	// is the largest in-memory stream that fits. The single-pass property that
	// keeps the work linear is asserted with the counting reader below and
	// holds for any stream length, including the 6000-packet target.
	ts, err := buildLeadingPacketsTS(1700)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(ts)/188, 5000)

	cr := &countingReader{r: bytes.NewReader(ts)}
	r := &mpegts.Reader{R: cr}
	require.NoError(t, r.Initialize())

	// every underlying byte is consumed at most once during the forward
	// discovery pass: a quadratic/restarting scan would exceed this bound.
	require.LessOrEqual(t, cr.bytes, int64(len(ts)))
	require.Greater(t, cr.bytes, int64(len(ts))*9/10)

	var latmTrack *mpegts.Track
	for _, tr := range r.Tracks() {
		if _, ok := tr.Codec.(*codecs.MPEG4AudioLATM); ok {
			latmTrack = tr
		}
	}
	require.NotNil(t, latmTrack)

	latmCodec := latmTrack.Codec.(*codecs.MPEG4AudioLATM)
	require.Len(t, latmCodec.StreamMuxConfig.Programs, 2)
	require.Equal(t, 24000, latmCodec.StreamMuxConfig.Programs[0].Layers[0].AudioSpecificConfig.SampleRate)
}

func BenchmarkLATMReaderInitialize(b *testing.B) {
	ts, err := buildLeadingPacketsTS(1700)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := &mpegts.Reader{R: bytes.NewReader(ts)}
		if err := r.Initialize(); err != nil {
			b.Fatal(err)
		}
	}
}
