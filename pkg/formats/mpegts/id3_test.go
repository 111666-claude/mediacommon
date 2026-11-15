package mpegts_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/asticode/go-astits"
	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/codecs"
)

func id3Tag(version byte, framesPayload []byte) []byte {
	size := len(framesPayload)

	tag := make([]byte, 0, 10+len(framesPayload))

	tag = append(tag,
		'I', 'D', '3', version, 0x00, 0x00,
		byte(size>>21&0x7f),
		byte(size>>14&0x7f),
		byte(size>>7&0x7f),
		byte(size&0x7f),
	)

	return append(tag, framesPayload...)
}

func id3Frame(id string, content []byte) []byte {
	size := len(content)
	return append([]byte{
		id[0], id[1], id[2], id[3],
		byte(size >> 21 & 0x7f),
		byte(size >> 14 & 0x7f),
		byte(size >> 7 & 0x7f),
		byte(size & 0x7f),
		0x00, 0x00,
	}, content...)
}

func TestWriterReaderID3(t *testing.T) {
	tag1 := id3Tag(4, id3Frame("TIT2", []byte{0x03, 'c', 'u', 'e', '1'}))
	tag2 := id3Tag(3, id3Frame("TIT2", []byte{0x03, 'c', 'u', 'e', '2'}))

	var buf bytes.Buffer
	w := &mpegts.Writer{
		W: &buf,
		Tracks: []*mpegts.Track{
			{Codec: &codecs.H264{}},
			{Codec: &codecs.ID3{}},
		},
	}
	err := w.Initialize()
	require.NoError(t, err)

	err = w.WriteH264(w.Tracks[0], 90000, 90000, [][]byte{{1, 2, 3, 4}})
	require.NoError(t, err)

	err = w.WriteID3(w.Tracks[1], 90000, tag1)
	require.NoError(t, err)

	err = w.WriteH264(w.Tracks[0], 180000, 180000, [][]byte{{5, 6, 7, 8}})
	require.NoError(t, err)

	err = w.WriteID3(w.Tracks[1], 120000, tag2)
	require.NoError(t, err)

	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	err = r.Initialize()
	require.NoError(t, err)

	require.Len(t, r.Tracks(), 2)
	require.IsType(t, &codecs.H264{}, r.Tracks()[0].Codec)
	require.IsType(t, &codecs.ID3{}, r.Tracks()[1].Codec)
	require.NotZero(t, r.Tracks()[1].PID)

	var id3Events []struct {
		pts int64
		tag []byte
	}

	r.OnDataID3(r.Tracks()[1], func(pts int64, tag []byte) error {
		id3Events = append(id3Events, struct {
			pts int64
			tag []byte
		}{pts, append([]byte(nil), tag...)})
		return nil
	})

	var videoPTS []int64

	r.OnDataH264(r.Tracks()[0], func(pts, _ int64, _ [][]byte) error {
		videoPTS = append(videoPTS, pts)
		return nil
	})

	for {
		err = r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, []int64{90000, 180000}, videoPTS)
	require.Len(t, id3Events, 2)
	require.Equal(t, int64(90000), id3Events[0].pts)
	require.Equal(t, tag1, id3Events[0].tag)
	require.Equal(t, int64(120000), id3Events[1].pts)
	require.Equal(t, tag2, id3Events[1].tag)
}

func TestWriterReaderID3Consecutive(t *testing.T) {
	const count = 50

	var buf bytes.Buffer
	w := &mpegts.Writer{
		W: &buf,
		Tracks: []*mpegts.Track{
			{Codec: &codecs.H264{}},
			{Codec: &codecs.ID3{}},
		},
	}
	err := w.Initialize()
	require.NoError(t, err)

	expected := make([][]byte, 0, count)

	for i := range count {
		tag := id3Tag(4, id3Frame("TIT2", []byte{0x03, byte(i)}))
		expected = append(expected, tag)

		err = w.WriteH264(w.Tracks[0], int64(i)*90000, int64(i)*90000, [][]byte{{1}})
		require.NoError(t, err)

		// consecutive events with the same PTS must not be merged or dropped
		err = w.WriteID3(w.Tracks[1], int64(i)*90000, tag)
		require.NoError(t, err)
	}

	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	err = r.Initialize()
	require.NoError(t, err)

	var events [][]byte

	r.OnDataID3(r.Tracks()[1], func(_ int64, tag []byte) error {
		events = append(events, append([]byte(nil), tag...))
		return nil
	})

	for {
		err = r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, expected, events)
}

func TestWriterReaderID3WrapAround(t *testing.T) {
	const maxTS = int64(0x1FFFFFFFF)

	tags := [][]byte{
		id3Tag(4, id3Frame("TIT2", []byte{'a'})),
		id3Tag(4, id3Frame("TIT2", []byte{'b'})),
		id3Tag(4, id3Frame("TIT2", []byte{'c'})),
	}

	ptss := []int64{
		maxTS - 90000, // right before wraparound
		maxTS - 1,
		90000, // after wraparound
	}

	var buf bytes.Buffer
	w := &mpegts.Writer{
		W: &buf,
		Tracks: []*mpegts.Track{
			{Codec: &codecs.H264{}},
			{Codec: &codecs.ID3{}},
		},
	}
	err := w.Initialize()
	require.NoError(t, err)

	for i, tag := range tags {
		err = w.WriteH264(w.Tracks[0], ptss[i], ptss[i], [][]byte{{1}})
		require.NoError(t, err)

		err = w.WriteID3(w.Tracks[1], ptss[i], tag)
		require.NoError(t, err)
	}

	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	err = r.Initialize()
	require.NoError(t, err)

	var gotPTS []int64
	var gotTags [][]byte

	r.OnDataID3(r.Tracks()[1], func(pts int64, tag []byte) error {
		gotPTS = append(gotPTS, pts)
		gotTags = append(gotTags, append([]byte(nil), tag...))
		return nil
	})

	for {
		err = r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	// the PES layer stores 33-bit timestamps; events must not be lost or misassigned
	require.Len(t, gotTags, len(tags))
	require.Equal(t, tags, gotTags)
	require.Equal(t, []int64{ptss[0], ptss[1], ptss[2]}, gotPTS)
}

func TestWriterReaderID3RoundTrip(t *testing.T) {
	tag := id3Tag(4, id3Frame("TIT2", []byte{0x03, 'h', 'i'}))

	var buf1 bytes.Buffer
	w1 := &mpegts.Writer{
		W: &buf1,
		Tracks: []*mpegts.Track{
			{Codec: &codecs.H264{}},
			{Codec: &codecs.ID3{}},
		},
	}
	err := w1.Initialize()
	require.NoError(t, err)

	err = w1.WriteH264(w1.Tracks[0], 90000, 90000, [][]byte{{1, 2}})
	require.NoError(t, err)

	err = w1.WriteID3(w1.Tracks[1], 90000, tag)
	require.NoError(t, err)

	// read back and remux into a second stream
	var buf2 bytes.Buffer
	w2 := &mpegts.Writer{
		W: &buf2,
		Tracks: []*mpegts.Track{
			{Codec: &codecs.H264{}},
			{Codec: &codecs.ID3{}},
		},
	}
	err = w2.Initialize()
	require.NoError(t, err)

	r := &mpegts.Reader{R: bytes.NewReader(buf1.Bytes())}
	err = r.Initialize()
	require.NoError(t, err)

	var readTags [][]byte

	r.OnDataID3(r.Tracks()[1], func(pts int64, data []byte) error {
		readTags = append(readTags, append([]byte(nil), data...))
		return w2.WriteID3(w2.Tracks[1], pts, data)
	})

	r.OnDataH264(r.Tracks()[0], func(pts, dts int64, au [][]byte) error {
		return w2.WriteH264(w2.Tracks[0], pts, dts, au)
	})

	for {
		err = r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, [][]byte{tag}, readTags)

	// the demuxed payload must be byte-identical to the original tag
	require.True(t, bytes.Contains(buf2.Bytes(), tag))

	// remuxing must preserve the tag and its presentation time exactly
	r2 := &mpegts.Reader{R: bytes.NewReader(buf2.Bytes())}
	err = r2.Initialize()
	require.NoError(t, err)

	var (
		gotPTS  int64
		gotTag  []byte
		gotTags int
	)

	r2.OnDataID3(r2.Tracks()[1], func(pts int64, data []byte) error {
		gotPTS = pts
		gotTag = append([]byte(nil), data...)
		gotTags++
		return nil
	})

	for {
		err = r2.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, 1, gotTags)
	require.Equal(t, int64(90000), gotPTS)
	require.Equal(t, tag, gotTag)
}

func TestWriterReaderID3Interleaved(t *testing.T) {
	// two ID3 tracks interleaved with each other and with video must not cross-assign
	tagA := id3Tag(4, id3Frame("TIT2", []byte("a")))
	tagB := id3Tag(4, id3Frame("TIT2", []byte("b")))

	var buf bytes.Buffer
	w := &mpegts.Writer{
		W: &buf,
		Tracks: []*mpegts.Track{
			{Codec: &codecs.H264{}},
			{Codec: &codecs.ID3{}},
			{Codec: &codecs.ID3{}},
		},
	}
	err := w.Initialize()
	require.NoError(t, err)

	type writing struct {
		track int
		pts   int64
		tag   []byte
	}

	writings := []writing{
		{0, 90000, nil},
		{1, 90000, tagA},
		{2, 93000, tagB},
		{0, 180000, nil},
		{2, 180000, tagB},
		{1, 120000, tagA},
		{1, 210000, tagA},
		{2, 240000, tagB},
	}

	for _, wr := range writings {
		if wr.track == 0 {
			err = w.WriteH264(w.Tracks[0], wr.pts, wr.pts, [][]byte{{1}})
		} else {
			err = w.WriteID3(w.Tracks[wr.track], wr.pts, wr.tag)
		}
		require.NoError(t, err)
	}

	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	err = r.Initialize()
	require.NoError(t, err)
	require.Len(t, r.Tracks(), 3)

	got := map[uint16][][2]any{}

	for i := 1; i <= 2; i++ {
		pid := r.Tracks()[i].PID
		got[pid] = nil

		r.OnDataID3(r.Tracks()[i], func(pts int64, tag []byte) error {
			got[pid] = append(got[pid], [2]any{pts, append([]byte(nil), tag...)})
			return nil
		})
	}

	for {
		err = r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, [][2]any{
		{int64(90000), tagA},
		{int64(120000), tagA},
		{int64(210000), tagA},
	}, got[r.Tracks()[1].PID])

	require.Equal(t, [][2]any{
		{int64(93000), tagB},
		{int64(180000), tagB},
		{int64(240000), tagB},
	}, got[r.Tracks()[2].PID])
}

func TestID3Invalid(t *testing.T) {
	valid := id3Tag(4, id3Frame("TIT2", []byte{0x03, 'h', 'i'}))

	for _, ca := range []struct {
		name string
		tag  []byte
		msg  string
	}{
		{
			"empty",
			nil,
			"truncated header at offset 0",
		},
		{
			"truncated header",
			valid[:5],
			"truncated header at offset 0",
		},
		{
			"invalid identifier",
			[]byte{'X', 'D', '3', 4, 0, 0, 0, 0, 0, 0},
			"invalid identifier at offset 0",
		},
		{
			"invalid version",
			[]byte{'I', 'D', '3', 0xFF, 0, 0, 0, 0, 0, 0},
			"invalid version at offset 3",
		},
		{
			"unsupported version",
			[]byte{'I', 'D', '3', 5, 0, 0, 0, 0, 0, 0},
			"unsupported version at offset 3",
		},
		{
			"invalid size",
			[]byte{'I', 'D', '3', 4, 0, 0, 0x80, 0, 0, 0},
			"invalid size at offset 6",
		},
		{
			"truncated tag",
			valid[:len(valid)-1],
			"truncated tag",
		},
		{
			"trailing data",
			append(append([]byte(nil), valid...), 0x01),
			"unexpected data",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &mpegts.Writer{
				W: &buf,
				Tracks: []*mpegts.Track{
					{Codec: &codecs.H264{}},
					{Codec: &codecs.ID3{}},
				},
			}
			err := w.Initialize()
			require.NoError(t, err)

			err = w.WriteID3(w.Tracks[1], 90000, ca.tag)
			require.ErrorContains(t, err, ca.msg)
		})
	}
}

func TestReaderID3InvalidDoesNotStop(t *testing.T) {
	invalid := id3Tag(4, id3Frame("TIT2", []byte{0x03, 'x', 'x'}))
	valid := id3Tag(4, id3Frame("TIT2", []byte{0x03, 'h', 'i'}))

	var buf bytes.Buffer
	w := &mpegts.Writer{
		W: &buf,
		Tracks: []*mpegts.Track{
			{Codec: &codecs.H264{}},
			{Codec: &codecs.ID3{}},
		},
	}
	err := w.Initialize()
	require.NoError(t, err)

	err = w.WriteH264(w.Tracks[0], 90000, 90000, [][]byte{{1, 2, 3, 4}})
	require.NoError(t, err)

	err = w.WriteID3(w.Tracks[1], 90000, invalid)
	require.NoError(t, err)

	err = w.WriteH264(w.Tracks[0], 180000, 180000, [][]byte{{5, 6, 7, 8}})
	require.NoError(t, err)

	err = w.WriteID3(w.Tracks[1], 180000, valid)
	require.NoError(t, err)

	// corrupt the version field of the first tag in the emitted stream
	raw := buf.Bytes()
	index := bytes.Index(raw, invalid)
	require.NotEqual(t, -1, index)
	raw[index+3] = 0xFF

	r := &mpegts.Reader{R: bytes.NewReader(raw)}
	err = r.Initialize()
	require.NoError(t, err)

	var decErrs []string

	r.OnDecodeError(func(err error) {
		decErrs = append(decErrs, err.Error())
	})

	var gotTags [][]byte

	r.OnDataID3(r.Tracks()[1], func(_ int64, tag []byte) error {
		gotTags = append(gotTags, append([]byte(nil), tag...))
		return nil
	})

	var videoCount int

	r.OnDataH264(r.Tracks()[0], func(_, _ int64, _ [][]byte) error {
		videoCount++
		return nil
	})

	for {
		err = r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, [][]byte{valid}, gotTags)
	require.Equal(t, 2, videoCount)
	require.Len(t, decErrs, 1)
	require.Contains(t, decErrs[0], "invalid ID3 tag")
	require.Contains(t, decErrs[0], "invalid version at offset 3")
}

func TestWriterID3Packets(t *testing.T) {
	tag := []byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0, 6, 'T', 'I', 'T', '2', 0, 0}

	var buf bytes.Buffer
	w := &mpegts.Writer{
		W: &buf,
		Tracks: []*mpegts.Track{
			{PID: 256, Codec: &codecs.H264{}},
			{PID: 257, Codec: &codecs.ID3{}},
		},
	}
	err := w.Initialize()
	require.NoError(t, err)

	err = w.WriteH264(w.Tracks[0], 30*90000, 30*90000, [][]byte{{1, 2, 3, 4}})
	require.NoError(t, err)

	err = w.WriteID3(w.Tracks[1], 30*90000, tag)
	require.NoError(t, err)

	expected := []*astits.Packet{
		{ // PAT
			Header: astits.PacketHeader{
				HasPayload:                true,
				PayloadUnitStartIndicator: true,
			},
			Payload: append([]byte{
				0x00, 0x00, 0xb0, 0x0d, 0x00, 0x00, 0xc1, 0x00,
				0x00, 0x00, 0x01, 0xf0, 0x00, 0x71, 0x10, 0xd8,
				0x78,
			}, bytes.Repeat([]byte{0xff}, 167)...),
		},
		{ // PMT with a metadata_descriptor identifying an "ID3 " track
			Header: astits.PacketHeader{
				HasPayload:                true,
				PayloadUnitStartIndicator: true,
				PID:                       4096,
			},
			Payload: append([]byte{
				0x00, 0x02, 0xb0, 0x26, 0x00, 0x01, 0xc1, 0x00,
				0x00, 0xe1, 0x00, 0xf0, 0x00, 0x1b, 0xe1, 0x00,
				0xf0, 0x00, 0x15, 0xe1, 0x01, 0xf0, 0x0f, 0x26,
				0x0d, 0xff, 0xff, 0x49, 0x44, 0x33, 0x20, 0xff,
				0x49, 0x44, 0x33, 0x20, 0x00, 0x0f, 0x6c, 0xfb,
				0xab, 0xc7,
			}, bytes.Repeat([]byte{0xff}, 142)...),
		},
		{ // video PES
			AdaptationField: &astits.PacketAdaptationField{
				PCR: &astits.ClockReference{
					Base: 2691000,
				},
				Length:         155,
				StuffingLength: 148,
				HasPCR:         true,
			},
			Header: astits.PacketHeader{
				HasAdaptationField:        true,
				HasPayload:                true,
				PayloadUnitStartIndicator: true,
				PID:                       256,
			},
			Payload: []byte{
				0x00, 0x00, 0x01, 0xe0, 0x00, 0x00, 0x80, 0x80,
				0x05, 0x21, 0x00, 0xa5, 0x65, 0xc1, 0x00, 0x00,
				0x00, 0x01, 0x09, 0xf0, 0x00, 0x00, 0x00, 0x01,
				0x01, 0x02, 0x03, 0x04,
			},
		},
		{ // ID3 PES
			Header: astits.PacketHeader{
				HasAdaptationField:        true,
				HasPayload:                true,
				PayloadUnitStartIndicator: true,
				PID:                       257,
			},
			AdaptationField: &astits.PacketAdaptationField{
				Length:                153,
				StuffingLength:        152,
				RandomAccessIndicator: true,
			},
			Payload: []byte{
				0x00, 0x00, 0x01, 0xbd, 0x00, 0x18, 0x80, 0x80,
				0x05, 0x21, 0x00, 0xa5, 0x65, 0xc1, 0x49, 0x44,
				0x33, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06,
				0x54, 0x49, 0x54, 0x32, 0x00, 0x00,
			},
		},
	}

	dem := astits.NewDemuxer(
		context.Background(),
		&buf,
		astits.DemuxerOptPacketSize(188))

	var pkts []*astits.Packet
	for {
		var pkt *astits.Packet
		pkt, err = dem.NextPacket()
		if errors.Is(err, astits.ErrNoMorePackets) {
			break
		}
		require.NoError(t, err)
		pkts = append(pkts, pkt)
	}

	require.Equal(t, expected, pkts)
}

func TestReaderID3FromPackets(t *testing.T) {
	input := []*astits.Packet{
		{ // PAT
			Header: astits.PacketHeader{
				HasPayload:                true,
				PayloadUnitStartIndicator: true,
				PID:                       0,
			},
			Payload: append([]byte{
				0x00, 0x00, 0xb0, 0x0d, 0x00, 0x00, 0xc1, 0x00,
				0x00, 0x00, 0x01, 0xf0, 0x00, 0x71, 0x10, 0xd8,
				0x78,
			}, bytes.Repeat([]byte{0xff}, 167)...),
		},
		{ // PMT with an "ID3 " metadata_descriptor
			Header: astits.PacketHeader{
				HasPayload:                true,
				PayloadUnitStartIndicator: true,
				PID:                       4096,
			},
			Payload: append([]byte{
				0x00, 0x02, 0xb0, 0x21, 0x00, 0x01, 0xc1, 0x00,
				0x00, 0xe1, 0x01, 0xf0, 0x00, 0x15, 0xe1, 0x01,
				0xf0, 0x0f, 0x26, 0x0d, 0xff, 0xff, 0x49, 0x44,
				0x33, 0x20, 0xff, 0x49, 0x44, 0x33, 0x20, 0x00,
				0x0f, 0x4b, 0x07, 0xc6, 0x79,
			}, bytes.Repeat([]byte{0xff}, 143)...),
		},
		{ // ID3 PES with a valid tag
			Header: astits.PacketHeader{
				HasPayload:                true,
				PayloadUnitStartIndicator: true,
				PID:                       257,
			},
			Payload: []byte{
				0x00, 0x00, 0x01, 0xbd, 0x00, 0x15, 0x80, 0x80,
				0x05, 0x21, 0x00, 0x01, 0x51, 0x81, 0x49, 0x44,
				0x33, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
				0x01, 0x02, 0x03,
			},
		},
		{ // ID3 PES with a malformed tag
			Header: astits.PacketHeader{
				HasPayload:                true,
				PayloadUnitStartIndicator: true,
				PID:                       257,
			},
			Payload: []byte{
				0x00, 0x00, 0x01, 0xbd, 0x00, 0x13, 0x80, 0x80,
				0x05, 0x21, 0x00, 0x02, 0xa3, 0x02, 0x49, 0x44,
				0x33, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
				0x01,
			},
		},
	}

	var buf bytes.Buffer
	mux := astits.NewMuxer(context.Background(), &buf)
	for _, pkt := range input {
		_, err := mux.WritePacket(pkt)
		require.NoError(t, err)
	}

	r := &mpegts.Reader{R: bytes.NewReader(buf.Bytes())}
	err := r.Initialize()
	require.NoError(t, err)
	require.Len(t, r.Tracks(), 1)
	require.IsType(t, &codecs.ID3{}, r.Tracks()[0].Codec)
	require.Equal(t, uint16(257), r.Tracks()[0].PID)

	var decErrs []string

	r.OnDecodeError(func(err error) {
		decErrs = append(decErrs, err.Error())
	})

	var gotTags [][]byte

	r.OnDataID3(r.Tracks()[0], func(_ int64, tag []byte) error {
		gotTags = append(gotTags, append([]byte(nil), tag...))
		return nil
	})

	for {
		err = r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	require.Equal(t, [][]byte{{'I', 'D', '3', 4, 0, 0, 0, 0, 0, 3, 1, 2, 3}}, gotTags)
	require.Len(t, decErrs, 1)
	require.Contains(t, decErrs[0], "invalid ID3 tag")
	require.Contains(t, decErrs[0], "truncated tag")
}
