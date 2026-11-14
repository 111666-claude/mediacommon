package mpegts_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/codecs"
	"github.com/bluenviron/mediacommon/v2/pkg/rewindablereader"
)

// largeFrameNALU builds a H264 NAL unit of the requested size. The payload
// contains no zero byte (and therefore no Annex-B start code), so it cannot
// be split or escaped and a byte-for-byte comparison is possible.
func largeFrameNALU(t *testing.T, naluType byte, size int) []byte {
	t.Helper()

	nalu := make([]byte, size)
	nalu[0] = naluType & 0x1f
	for i := 1; i < size; i++ {
		nalu[i] = byte(1 + (i % 255))
	}

	require.NotContains(t, nalu, []byte{0x00, 0x00, 0x01})

	return nalu
}

// largeFrameSize returns a frame size that exceeds the 1 MiB generic
// rewindable-reader default. Under the race detector a smaller size is used
// to keep the test suite within its time budget.
func largeFrameSize() int {
	if raceEnabled() {
		return 1100 * 1024
	}
	return 2 * 1024 * 1024
}

// boundaryFrameSize returns a frame size that, once muxed, produces data
// above the 1 MiB generic default while staying below the 8 MiB
// initialization limit.
func boundaryFrameSize() int {
	if raceEnabled() {
		return 1500 * 1024
	}
	return 7 * 1024 * 1024
}

// exceededFrameSize returns the size of each frame written before the first
// audio packet. enoughExceededFrames() of them push the recorded data beyond
// the 8 MiB initialization limit.
func exceededFrameSize() int {
	if raceEnabled() {
		return 1100 * 1024
	}
	return 5 * 1024 * 1024
}

// enoughExceededFrames returns how many frames of exceededFrameSize() are
// needed for the muxed data to cross the 8 MiB initialization limit.
// TS muxing adds about 2% overhead, hence the conservative divisor.
func enoughExceededFrames() int {
	return (8 * 1024 * 1024 / exceededFrameSize()) + 1
}

func newLargeFrameWriter(t *testing.T, buf *bytes.Buffer) (*mpegts.Writer, *mpegts.Track, *mpegts.Track) {
	t.Helper()

	videoTrack := &mpegts.Track{Codec: &codecs.H264{}}
	audioTrack := &mpegts.Track{
		Codec: &codecs.MPEG4Audio{
			Config: mpeg4audio.AudioSpecificConfig{
				Type:          mpeg4audio.ObjectTypeAACLC,
				SampleRate:    48000,
				ChannelConfig: 2,
				ChannelCount:  2, //nolint:staticcheck
			},
		},
	}

	w := &mpegts.Writer{
		W:      buf,
		Tracks: []*mpegts.Track{videoTrack, audioTrack},
	}
	require.NoError(t, w.Initialize())

	return w, videoTrack, audioTrack
}

// verifyRewoundStream reads the whole stream and checks that every packet is
// delivered again with unchanged bytes and timestamps.
func verifyRewoundStream(t *testing.T, data []byte, expectedVideo []sample, expectedAudio [][]byte) {
	t.Helper()

	r := &mpegts.Reader{R: bytes.NewReader(data)}
	require.NoError(t, r.Initialize())
	require.Len(t, r.Tracks(), 2)

	videoTrack := r.Tracks()[0]
	audioTrack := r.Tracks()[1]
	require.IsType(t, &codecs.H264{}, videoTrack.Codec)
	require.Equal(t, &codecs.MPEG4Audio{
		Config: mpeg4audio.AudioSpecificConfig{
			Type:          mpeg4audio.ObjectTypeAACLC,
			SampleRate:    48000,
			ChannelConfig: 2,
			ChannelCount:  2, //nolint:staticcheck
		},
	}, audioTrack.Codec)

	var gotVideo []sample
	r.OnDataH264(videoTrack, func(pts int64, dts int64, au [][]byte) error {
		gotVideo = append(gotVideo, sample{pts: pts, dts: dts, data: au})
		return nil
	})

	var gotAudio [][]byte
	r.OnDataMPEG4Audio(audioTrack, func(_ int64, aus [][]byte) error {
		gotAudio = append(gotAudio, aus...)
		return nil
	})

	for {
		err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, expectedVideo, gotVideo)
	require.Equal(t, expectedAudio, gotAudio)
}

func TestReaderLargeInitialFrame(t *testing.T) {
	// the first video PES is much larger than the generic 1 MiB default and
	// the audio parameters only arrive in a packet after it
	frameSize := largeFrameSize()
	idr := largeFrameNALU(t, 5, frameSize)

	var buf bytes.Buffer
	w, videoTrack, audioTrack := newLargeFrameWriter(t, &buf)

	pts0 := int64(30 * 90000)
	require.NoError(t, w.WriteH264(videoTrack, pts0, pts0, [][]byte{{7}, {8}, idr}))

	audioAU := []byte{0x11, 0x22, 0x33}
	ptsAudio := int64(31 * 90000)
	require.NoError(t, w.WriteMPEG4Audio(audioTrack, ptsAudio, [][]byte{audioAU}))

	frame2 := largeFrameNALU(t, 1, 1024)
	pts1 := int64(60 * 90000)
	dts1 := int64(59 * 90000)
	require.NoError(t, w.WriteH264(videoTrack, pts1, dts1, [][]byte{frame2}))

	// sanity check: the recorded amount really exceeds the generic default
	require.Greater(t, buf.Len(), rewindablereader.MaxRecordedSize)

	expectedVideo := []sample{
		{pts: pts0, dts: pts0, data: [][]byte{{7}, {8}, idr}},
		{pts: pts1, dts: dts1, data: [][]byte{frame2}},
	}
	verifyRewoundStream(t, buf.Bytes(), expectedVideo, [][]byte{audioAU})
}

func TestReaderLargeInitialFrameAudioFirst(t *testing.T) {
	// audio parameters are already available before the large video frame:
	// the result must be the same as with the delayed-audio ordering
	frameSize := largeFrameSize()
	idr := largeFrameNALU(t, 5, frameSize)

	var buf bytes.Buffer
	w, videoTrack, audioTrack := newLargeFrameWriter(t, &buf)

	ptsAudio := int64(30 * 90000)
	audioAU := []byte{0x44, 0x55}
	require.NoError(t, w.WriteMPEG4Audio(audioTrack, ptsAudio, [][]byte{audioAU}))

	pts0 := int64(31 * 90000)
	require.NoError(t, w.WriteH264(videoTrack, pts0, pts0, [][]byte{idr}))

	expectedVideo := []sample{
		{pts: pts0, dts: pts0, data: [][]byte{idr}},
	}
	verifyRewoundStream(t, buf.Bytes(), expectedVideo, [][]byte{audioAU})
}

func TestReaderInitializationRecordedSizeBoundary(t *testing.T) {
	// legal data spanning almost 8 MiB of recorded data must initialize
	frameSize := boundaryFrameSize()
	idr := largeFrameNALU(t, 5, frameSize)

	var buf bytes.Buffer
	w, videoTrack, audioTrack := newLargeFrameWriter(t, &buf)

	pts0 := int64(90000)
	require.NoError(t, w.WriteH264(videoTrack, pts0, pts0, [][]byte{idr}))

	audioAU := []byte{0x66}
	ptsAudio := int64(2 * 90000)
	require.NoError(t, w.WriteMPEG4Audio(audioTrack, ptsAudio, [][]byte{audioAU}))

	// the stream recorded during initialization is above the 1 MiB default
	// but below the 8 MiB MPEG-TS-specific limit
	require.Greater(t, buf.Len(), rewindablereader.MaxRecordedSize)
	require.Less(t, buf.Len(), 8*1024*1024)

	expectedVideo := []sample{
		{pts: pts0, dts: pts0, data: [][]byte{idr}},
	}
	verifyRewoundStream(t, buf.Bytes(), expectedVideo, [][]byte{audioAU})
}

func TestReaderInitializationRecordedSizeExceeded(t *testing.T) {
	// large video frames before the first audio packet push the amount of
	// recorded data beyond the 8 MiB limit: the original error must surface
	frameSize := exceededFrameSize()

	var buf bytes.Buffer
	w, videoTrack, audioTrack := newLargeFrameWriter(t, &buf)

	pts := int64(90000)
	for range enoughExceededFrames() {
		require.NoError(t, w.WriteH264(videoTrack, pts, pts, [][]byte{largeFrameNALU(t, 1, frameSize)}))
		pts += 90000
	}

	require.NoError(t, w.WriteMPEG4Audio(audioTrack, pts, [][]byte{{0x77}}))
	require.Greater(t, buf.Len(), 8*1024*1024)

	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	err := r.Initialize()
	require.Error(t, err)
	require.ErrorIs(t, err, rewindablereader.ErrMaxRecordedSizeExceeded)
	require.EqualError(t, err, "max recorded size exceeded")

	// the failed reader can be discarded and a fresh one succeeds on a
	// different, valid stream: no global state was left behind
	var smallBuf bytes.Buffer
	smallW, smallVideo, smallAudio := newLargeFrameWriter(t, &smallBuf)
	require.NoError(t, smallW.WriteH264(smallVideo, 90000, 90000, [][]byte{{7}, {8}, {5}}))
	require.NoError(t, smallW.WriteMPEG4Audio(smallAudio, 90000, [][]byte{{0x01}}))

	retry := &mpegts.Reader{R: bytes.NewReader(smallBuf.Bytes())}
	require.NoError(t, retry.Initialize())
	require.Len(t, retry.Tracks(), 2)

	// the same Reader instance can also be re-pointed at a valid stream and
	// initialized again: a failed Initialize leaves no internal state behind
	r.R = bytes.NewReader(smallBuf.Bytes())
	require.NoError(t, r.Initialize())
	require.Len(t, r.Tracks(), 2)
}

func TestReaderLargeInitialFrameNoAudio(t *testing.T) {
	// a video-only stream with a large first frame must keep working
	frameSize := largeFrameSize()
	idr := largeFrameNALU(t, 5, frameSize)

	var buf bytes.Buffer
	videoTrack := &mpegts.Track{Codec: &codecs.H264{}}
	w := &mpegts.Writer{
		W:      &buf,
		Tracks: []*mpegts.Track{videoTrack},
	}
	require.NoError(t, w.Initialize())

	pts0 := int64(90000)
	require.NoError(t, w.WriteH264(videoTrack, pts0, pts0, [][]byte{idr}))

	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	require.NoError(t, r.Initialize())
	require.Len(t, r.Tracks(), 1)

	var got []sample
	r.OnDataH264(r.Tracks()[0], func(pts int64, dts int64, au [][]byte) error {
		got = append(got, sample{pts: pts, dts: dts, data: au})
		return nil
	})

	for {
		err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, []sample{{pts: pts0, dts: pts0, data: [][]byte{idr}}}, got)
}
