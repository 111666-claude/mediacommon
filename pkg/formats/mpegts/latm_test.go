package mpegts_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/asticode/go-astits"
	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/codecs"
)

// anchorStreamMuxConfig reproduces the shape described in the issue:
// 2 programs, the first one with 2 layers, with sample rates 24000, 48000
// and 44100. The layers use distinct buffer fullness values to ensure they
// cannot overwrite each other.
func anchorStreamMuxConfig() *mpeg4audio.StreamMuxConfig {
	asc := func(rate int) *mpeg4audio.AudioSpecificConfig {
		return &mpeg4audio.AudioSpecificConfig{
			Type:          mpeg4audio.ObjectTypeAACLC,
			SampleRate:    rate,
			ChannelConfig: 1,
			ChannelCount:  1, //nolint:staticcheck
		}
	}

	return &mpeg4audio.StreamMuxConfig{
		Programs: []*mpeg4audio.StreamMuxConfigProgram{
			{
				Layers: []*mpeg4audio.StreamMuxConfigLayer{
					{AudioSpecificConfig: asc(24000), FrameLengthType: 0, LatmBufferFullness: 255},
					{AudioSpecificConfig: asc(48000), FrameLengthType: 0, LatmBufferFullness: 200},
				},
			},
			{
				Layers: []*mpeg4audio.StreamMuxConfigLayer{
					{AudioSpecificConfig: asc(44100), FrameLengthType: 0, LatmBufferFullness: 128},
				},
			},
		},
	}
}

func anchorPayloads() [][][][]byte {
	return [][][][]byte{
		{
			{
				{0x01, 0x02, 0x03, 0x04}, // program 0, layer 0
				{0x05, 0x06, 0x07, 0x08}, // program 0, layer 1
			},
			{
				{0x09, 0x0a, 0x0b, 0x0c}, // program 1, layer 0
			},
		},
	}
}

// buildLATMElement builds the raw bytes of an audio mux element.
func buildLATMElement(
	t *testing.T,
	conf *mpeg4audio.StreamMuxConfig,
	useSame bool,
	payloads [][][][]byte,
) []byte {
	t.Helper()

	enc, err := mpeg4audio.AudioMuxElement{
		MuxConfigPresent: true,
		UseSameStreamMux: useSame,
		StreamMuxConfig:  conf,
		Payloads:         payloads,
	}.Marshal()
	require.NoError(t, err)
	return enc
}

// frameLATM wraps raw audio mux elements in an AudioSyncStream.
func frameLATM(t *testing.T, elements ...[]byte) []byte {
	t.Helper()

	enc, err := mpeg4audio.AudioSyncStream{AudioMuxElements: elements}.Marshal()
	require.NoError(t, err)
	return enc
}

// muxLATMTS builds a MPEG-TS buffer containing a single AAC LATM elementary
// stream (PID 257) with the given PES payloads, each in its own PES.
func muxLATMTS(t *testing.T, payloads [][]byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	mux := astits.NewMuxer(context.Background(), &buf)

	err := mux.AddElementaryStream(astits.PMTElementaryStream{
		ElementaryPID: 257,
		StreamType:    astits.StreamTypeAACLATMAudio,
	})
	require.NoError(t, err)
	mux.SetPCRPID(257)

	for i, payload := range payloads {
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
						PTS:             &astits.ClockReference{Base: int64(i+1) * 90000},
					},
					StreamID: 192,
				},
				Data: payload,
			},
		})
		require.NoError(t, err)
	}

	return buf.Bytes()
}

func requireAnchorConfig(t *testing.T, track *mpegts.Track) {
	t.Helper()

	codec, ok := track.Codec.(*codecs.MPEG4AudioLATM)
	require.True(t, ok)
	require.Equal(t, anchorStreamMuxConfig(), &codec.StreamMuxConfig)

	// explicit anchor assertions: 2 programs, 2 then 1 layers, rates in order
	require.Len(t, codec.Programs, 2)
	require.Len(t, codec.Programs[0].Layers, 2)
	require.Len(t, codec.Programs[1].Layers, 1)
	require.Equal(t, 24000, codec.Programs[0].Layers[0].AudioSpecificConfig.SampleRate)
	require.Equal(t, 48000, codec.Programs[0].Layers[1].AudioSpecificConfig.SampleRate)
	require.Equal(t, 44100, codec.Programs[1].Layers[0].AudioSpecificConfig.SampleRate)
	require.Equal(t, uint(255), codec.Programs[0].Layers[0].LatmBufferFullness)
	require.Equal(t, uint(200), codec.Programs[0].Layers[1].LatmBufferFullness)
	require.Equal(t, uint(128), codec.Programs[1].Layers[0].LatmBufferFullness)
}

func TestLATMRoundTrip(t *testing.T) {
	conf := anchorStreamMuxConfig()
	payloads := anchorPayloads()

	configElement := buildLATMElement(t, conf, false, payloads)
	sameElement := buildLATMElement(t, conf, true, payloads)

	track := &mpegts.Track{
		Codec: &codecs.MPEG4AudioLATM{StreamMuxConfig: *conf},
	}

	var buf bytes.Buffer
	w := &mpegts.Writer{
		W:      &buf,
		Tracks: []*mpegts.Track{track},
	}
	require.NoError(t, w.Initialize())

	type written struct {
		pts int64
		els [][]byte
	}
	writtenSamples := []written{
		{pts: 90000, els: [][]byte{configElement}},
		{pts: 180000, els: [][]byte{sameElement}},
		{pts: 270000, els: [][]byte{configElement, sameElement}},
	}
	for _, s := range writtenSamples {
		require.NoError(t, w.WriteMPEG4AudioLATM(track, s.pts, s.els))
	}

	r, err := mpegts.NewReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Len(t, r.Tracks(), 1)
	requireAnchorConfig(t, r.Tracks()[0])

	var got []written
	r.OnDataMPEG4AudioLATM(r.Tracks()[0], func(pts int64, els [][]byte) error {
		got = append(got, written{pts: pts, els: els})
		return nil
	})

	for {
		err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Len(t, got, len(writtenSamples))
	for i, s := range writtenSamples {
		require.Equal(t, s.pts, got[i].pts)
		require.Equal(t, s.els, got[i].els)
	}
}

func TestLATMNoConfigUseSameOnly(t *testing.T) {
	conf := anchorStreamMuxConfig()
	// every element reuses a configuration that is never present in the stream
	sameElement := buildLATMElement(t, conf, true, anchorPayloads())

	ts := muxLATMTS(t, [][]byte{frameLATM(t, sameElement)})

	r := &mpegts.Reader{R: bytes.NewReader(ts)}
	err := r.Initialize()
	require.Error(t, err)
	require.Contains(t, err.Error(), "stream mux configuration not found")
}

func TestLATMPesWithoutSyncwordThenRecovers(t *testing.T) {
	conf := anchorStreamMuxConfig()
	configElement := buildLATMElement(t, conf, false, anchorPayloads())

	// first PES: no LATM syncword at all (truncated frame from capture start)
	garbage := bytes.Repeat([]byte{0xaa}, 64)
	// second PES: valid, carries the configuration
	valid := frameLATM(t, configElement)

	ts := muxLATMTS(t, [][]byte{garbage, valid})

	r := &mpegts.Reader{R: bytes.NewReader(ts)}
	err := r.Initialize()
	require.NoError(t, err)
	require.Len(t, r.Tracks(), 1)
	requireAnchorConfig(t, r.Tracks()[0])
}

func TestLATMPesWithTruncatedPrefixResyncs(t *testing.T) {
	conf := anchorStreamMuxConfig()
	configElement := buildLATMElement(t, conf, false, anchorPayloads())

	// the PES starts with a truncated frame (no syncword) followed by valid frames
	payload := append(append([]byte{}, bytes.Repeat([]byte{0x12}, 40)...), frameLATM(t, configElement)...)
	ts := muxLATMTS(t, [][]byte{payload})

	r := &mpegts.Reader{R: bytes.NewReader(ts)}
	err := r.Initialize()
	require.NoError(t, err)
	require.Len(t, r.Tracks(), 1)
	requireAnchorConfig(t, r.Tracks()[0])
}

func TestLATMPesWithFakeSyncwordResyncs(t *testing.T) {
	conf := anchorStreamMuxConfig()
	configElement := buildLATMElement(t, conf, false, anchorPayloads())

	// the truncated prefix contains a syncword-looking sequence that does not
	// decode (a zero-length element), followed by filler and the real frame.
	prefix := []byte{0x56, 0xe0, 0x00}
	prefix = append(prefix, bytes.Repeat([]byte{0x34}, 16)...)
	payload := append(prefix, frameLATM(t, configElement)...)
	ts := muxLATMTS(t, [][]byte{payload})

	r := &mpegts.Reader{R: bytes.NewReader(ts)}
	err := r.Initialize()
	require.NoError(t, err)
	require.Len(t, r.Tracks(), 1)
	requireAnchorConfig(t, r.Tracks()[0])
}

func TestLATMInvalidAudioSpecificConfig(t *testing.T) {
	// Craft the bytes of an element whose config carries a reserved object
	// type (0): Marshal() would reject it, so the malformed config has to be
	// produced by mutating a valid element.
	validConf := anchorStreamMuxConfig()
	raw := buildLATMElement(t, validConf, false, anchorPayloads())

	// The element starts with the useSameStreamMux flag (1 bit), then the
	// StreamMuxConfig: audioMuxVersion(1) + allStreamsSameTimeFraming(1) +
	// numSubFrames(6) + numProgram(4) + numLayer(3) = 15 bits. So the ASC
	// 5-bit object type occupies bits 16..20, i.e. the top 5 bits of byte 2.
	// Clear them to force the reserved object type 0.
	raw[2] &^= 0xf8

	ts := muxLATMTS(t, [][]byte{frameLATM(t, raw)})

	r := &mpegts.Reader{R: bytes.NewReader(ts)}
	err := r.Initialize()
	require.Error(t, err)
	require.Contains(t, err.Error(), "stream mux configuration not found")
	require.Contains(t, err.Error(), "unsupported object type")
}

func TestLATMRuntimeDecodeErrorOnCorruptedPes(t *testing.T) {
	conf := anchorStreamMuxConfig()
	configElement := buildLATMElement(t, conf, false, anchorPayloads())

	ts := muxLATMTS(t, [][]byte{
		frameLATM(t, configElement),    // valid, lets the track be recognized
		bytes.Repeat([]byte{0xaa}, 32), // corrupted: no syncword
	})

	r, err := mpegts.NewReader(bytes.NewReader(ts))
	require.NoError(t, err)
	require.Len(t, r.Tracks(), 1)
	requireAnchorConfig(t, r.Tracks()[0])

	var decodeErrs []error
	r.OnDecodeError(func(err error) {
		decodeErrs = append(decodeErrs, err)
	})

	callbacks := 0
	r.OnDataMPEG4AudioLATM(r.Tracks()[0], func(pts int64, els [][]byte) error {
		callbacks++
		return nil
	})

	for {
		err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, 1, callbacks)
	require.Len(t, decodeErrs, 1)
	require.Contains(t, decodeErrs[0].Error(), "no resync possible")
}

// countingReader counts the bytes read from the underlying source. Because the
// rewindable reader records the first forward scan once, a linear
// implementation makes the total number of underlying reads equal to the
// stream size: every TS packet is visited once instead of being re-searched
// from the beginning for each track.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func TestLATMLinearScan(t *testing.T) {
	conf := anchorStreamMuxConfig()
	payloads := anchorPayloads()
	configElement := buildLATMElement(t, conf, false, payloads)
	sameElement := buildLATMElement(t, conf, true, payloads)

	track0 := &mpegts.Track{
		Codec: &codecs.MPEG4AudioLATM{StreamMuxConfig: *conf},
	}
	track1 := &mpegts.Track{
		Codec: &codecs.MPEG4AudioLATM{StreamMuxConfig: *conf},
	}

	var buf bytes.Buffer
	w := &mpegts.Writer{
		W:      &buf,
		Tracks: []*mpegts.Track{track0, track1},
	}
	require.NoError(t, w.Initialize())

	// each track must expose a configuration-bearing packet first so that both
	// are discovered during the single forward scan
	require.NoError(t, w.WriteMPEG4AudioLATM(track0, 90000, [][]byte{configElement}))
	require.NoError(t, w.WriteMPEG4AudioLATM(track1, 180000, [][]byte{configElement}))

	// then alternate the two tracks, reusing the config on half of the packets
	for i := 0; i < 4000; i++ {
		tr := track0
		el := sameElement
		if i%2 == 0 {
			tr = track1
			el = configElement
		}
		require.NoError(t, w.WriteMPEG4AudioLATM(tr, int64(i+3)*90000, [][]byte{el}))
	}

	require.GreaterOrEqual(t, buf.Len()/188, 6000, "fixture must contain at least 6000 TS packets")

	cr := &countingReader{r: bytes.NewReader(buf.Bytes())}
	r := &mpegts.Reader{R: cr}
	require.NoError(t, r.Initialize())

	// discovering both tracks must never pull more bytes than the stream size:
	// the forward scan visits every TS packet once, there is no per-track
	// re-search from the beginning of the underlying source.
	require.LessOrEqual(t, cr.n, int64(buf.Len()))

	require.Len(t, r.Tracks(), 2)
	for _, tr := range r.Tracks() {
		requireAnchorConfig(t, tr)
	}

	for _, tr := range r.Tracks() {
		r.OnDataMPEG4AudioLATM(tr, func(int64, [][]byte) error { return nil })
	}
	for {
		err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	// after a full read, the underlying source has been consumed exactly once.
	require.Equal(t, int64(buf.Len()), cr.n)
}
