package pmp4

import (
	"bytes"
	"testing"

	amp4 "github.com/abema/go-mp4"
	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/formats/fmp4/seekablebuffer"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mp4/codecs"
)

// boxWriter builds a MP4 box tree used by the sample table tests.
type boxWriter struct {
	w   *amp4.Writer
	buf *seekablebuffer.Buffer
}

func newBoxWriter() *boxWriter {
	buf := &seekablebuffer.Buffer{}
	return &boxWriter{
		w:   amp4.NewWriter(buf),
		buf: buf,
	}
}

func (bw *boxWriter) start(box amp4.IImmutableBox) *boxWriter {
	_, err := bw.w.StartBox(&amp4.BoxInfo{Type: box.GetType()})
	if err != nil {
		panic(err)
	}

	_, err = amp4.Marshal(bw.w, box, amp4.Context{})
	if err != nil {
		panic(err)
	}
	return bw
}

func (bw *boxWriter) end() *boxWriter {
	_, err := bw.w.EndBox()
	if err != nil {
		panic(err)
	}
	return bw
}

func (bw *boxWriter) box(box amp4.IImmutableBox) *boxWriter {
	bw.start(box)
	return bw.end()
}

func (bw *boxWriter) raw(payload []byte) *boxWriter {
	_, err := bw.buf.Write(payload)
	if err != nil {
		panic(err)
	}
	return bw
}

func (bw *boxWriter) bytes() []byte {
	return bw.buf.Bytes()
}

// corruptTrackOptions describes the sample table boxes to emit for a track.
type corruptTrackOptions struct {
	skipStsz       bool
	skipStco       bool
	dupStsz        bool
	dupStco        bool
	stcoBeforeStsz bool
}

// testTrack describes a track and the payload bytes that its samples point at.
type testTrack struct {
	id          int
	sampleSizes []uint32
	chunkSizes  []int // optional; when nil all samples belong to a single chunk
	opts        corruptTrackOptions
}

func (t testTrack) chunks() []int {
	if len(t.sampleSizes) == 0 {
		return nil
	}
	if len(t.chunkSizes) == 0 {
		return []int{len(t.sampleSizes)}
	}
	return t.chunkSizes
}

func addCodecSampleEntry(bw *boxWriter) {
	bw.start(&amp4.AudioSampleEntry{
		SampleEntry: amp4.SampleEntry{
			AnyTypeBox:         amp4.AnyTypeBox{Type: amp4.BoxTypeOpus()},
			DataReferenceIndex: 1,
		},
		ChannelCount: 2,
		SampleSize:   16,
		SampleRate:   48000 << 16,
	})
	bw.box(&amp4.DOps{
		OutputChannelCount: 2,
		PreSkip:            312,
		InputSampleRate:    48000,
	})
	bw.end()
}

func addTrack(bw *boxWriter, track testTrack, chunkOffsets []uint32) {
	bw.start(&amp4.Trak{})

	bw.box(&amp4.Tkhd{TrackID: uint32(track.id)})
	bw.start(&amp4.Edts{})
	bw.box(&amp4.Elst{
		EntryCount: 1,
		Entries: []amp4.ElstEntry{{
			MediaTimeV0:      0,
			MediaRateInteger: 1,
		}},
	})
	bw.end()

	bw.start(&amp4.Mdia{})
	bw.box(&amp4.Mdhd{
		Timescale: 90000,
		Language:  [3]byte{'u', 'n', 'd'},
	})
	bw.start(&amp4.Minf{})
	bw.start(&amp4.Stbl{})
	bw.start(&amp4.Stsd{EntryCount: 1})
	addCodecSampleEntry(bw)
	bw.end()

	var sttsEntries []amp4.SttsEntry
	if len(track.sampleSizes) != 0 {
		sttsEntries = []amp4.SttsEntry{{
			SampleCount: uint32(len(track.sampleSizes)),
			SampleDelta: 100,
		}}
	}
	bw.box(&amp4.Stts{
		EntryCount: uint32(len(sttsEntries)),
		Entries:    sttsEntries,
	})

	var stscEntries []amp4.StscEntry
	for i, size := range track.chunks() {
		stscEntries = append(stscEntries, amp4.StscEntry{
			FirstChunk:             uint32(i + 1),
			SamplesPerChunk:        uint32(size),
			SampleDescriptionIndex: 1,
		})
	}
	bw.box(&amp4.Stsc{
		EntryCount: uint32(len(stscEntries)),
		Entries:    stscEntries,
	})

	stsz := func() {
		bw.box(&amp4.Stsz{
			SampleCount: uint32(len(track.sampleSizes)),
			EntrySize:   append([]uint32{}, track.sampleSizes...),
		})
	}
	stco := func() {
		bw.box(&amp4.Stco{
			EntryCount:  uint32(len(chunkOffsets)),
			ChunkOffset: chunkOffsets,
		})
	}

	switch {
	case track.opts.stcoBeforeStsz:
		stco()
		stsz()
	case track.opts.dupStsz:
		stsz()
		stsz()
		stco()
	case track.opts.dupStco:
		stsz()
		stco()
		stco()
	default:
		if !track.opts.skipStsz {
			stsz()
		}
		if !track.opts.skipStco {
			stco()
		}
	}

	bw.end() // stbl
	bw.end() // minf
	bw.end() // mdia
	bw.end() // trak
}

func rowOf(v []uint32) []uint32 { return v }

// buildPresentation builds a complete MP4 file whose mdat comes before moov.
func buildPresentation(tracks []testTrack) []byte {
	ftyp := newBoxWriter().box(&amp4.Ftyp{
		MajorBrand:   [4]byte{'i', 's', 'o', 'm'},
		MinorVersion: 1,
	}).bytes()

	dataOffset := len(ftyp) + 8 // ftyp + small mdat header

	var payload []byte
	chunkOffsets := make([][]uint32, len(tracks))

	for i, track := range tracks {
		sampleIdx := 0
		for _, chunkSize := range track.chunks() {
			chunkOffsets[i] = append(chunkOffsets[i], uint32(dataOffset+len(payload)))
			for range chunkSize {
				if sampleIdx >= len(track.sampleSizes) {
					panic("chunk sizes exceed sample count")
				}
				payload = append(payload, bytes.Repeat([]byte{byte(track.id)}, int(track.sampleSizes[sampleIdx]))...)
				sampleIdx++
			}
		}
		if sampleIdx != len(track.sampleSizes) {
			panic("chunk sizes do not cover all samples")
		}
	}

	bw := newBoxWriter()
	bw.raw(ftyp)

	mdatSize := uint32(8 + len(payload))
	bw.raw([]byte{byte(mdatSize >> 24), byte(mdatSize >> 16), byte(mdatSize >> 8), byte(mdatSize)})
	bw.raw([]byte{'m', 'd', 'a', 't'})
	bw.raw(payload)

	bw.start(&amp4.Moov{})
	bw.box(&amp4.Mvhd{
		Timescale:   1000,
		Rate:        65536,
		Volume:      256,
		NextTrackID: uint32(len(tracks) + 1),
	})
	for i, track := range tracks {
		addTrack(bw, track, chunkOffsets[i])
	}
	bw.end()

	return bw.bytes()
}

func TestPresentationUnmarshalCorruptedSampleTables(t *testing.T) {
	for _, ca := range []struct {
		name   string
		tracks []testTrack
		error  bool
	}{
		{
			"valid single track",
			[]testTrack{{
				id:          1,
				sampleSizes: []uint32{2, 3},
			}},
			false,
		},
		{
			"valid multiple tracks",
			[]testTrack{
				{id: 1, sampleSizes: []uint32{2, 3}},
				{id: 2, sampleSizes: []uint32{1, 1, 1}},
			},
			false,
		},
		{
			"valid empty sample track",
			[]testTrack{
				{id: 1, sampleSizes: nil},
				{id: 2, sampleSizes: []uint32{2}},
			},
			false,
		},
		{
			"valid interleaved chunks",
			[]testTrack{
				{id: 1, sampleSizes: []uint32{2, 2, 2}, chunkSizes: []int{1, 2}},
				{id: 2, sampleSizes: []uint32{1, 1}, chunkSizes: []int{2}},
				{id: 3, sampleSizes: []uint32{3, 1, 2}, chunkSizes: []int{2, 1}},
			},
			false,
		},
		{
			"missing stco followed by a valid track",
			[]testTrack{
				{id: 1, sampleSizes: []uint32{2, 2}, opts: corruptTrackOptions{skipStco: true}},
				{id: 2, sampleSizes: []uint32{2}},
			},
			true,
		},
		{
			"missing stsz followed by a valid track",
			[]testTrack{
				{id: 1, sampleSizes: []uint32{2, 2}, opts: corruptTrackOptions{skipStsz: true}},
				{id: 2, sampleSizes: []uint32{2}},
			},
			true,
		},
		{
			"missing stco at end of file",
			[]testTrack{{id: 1, sampleSizes: []uint32{2, 2}, opts: corruptTrackOptions{skipStco: true}}},
			true,
		},
		{
			"missing stsz at end of file",
			[]testTrack{{id: 1, sampleSizes: []uint32{2, 2}, opts: corruptTrackOptions{skipStsz: true}}},
			true,
		},
		{
			"duplicated stsz",
			[]testTrack{{id: 1, sampleSizes: []uint32{2, 2}, opts: corruptTrackOptions{dupStsz: true}}},
			true,
		},
		{
			"duplicated stsz followed by a valid track",
			[]testTrack{
				{id: 1, sampleSizes: []uint32{2, 2}, opts: corruptTrackOptions{dupStsz: true}},
				{id: 2, sampleSizes: []uint32{2}},
			},
			true,
		},
		{
			"duplicated stco",
			[]testTrack{{id: 1, sampleSizes: []uint32{2, 2}, opts: corruptTrackOptions{dupStco: true}}},
			true,
		},
		{
			"duplicated stco followed by a valid track",
			[]testTrack{
				{id: 1, sampleSizes: []uint32{2, 2}, opts: corruptTrackOptions{dupStco: true}},
				{id: 2, sampleSizes: []uint32{2}},
			},
			true,
		},
		{
			"stco before stsz",
			[]testTrack{{id: 1, sampleSizes: []uint32{2, 2}, opts: corruptTrackOptions{stcoBeforeStsz: true}}},
			true,
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			data := buildPresentation(ca.tracks)

			var p Presentation
			err := p.Unmarshal(bytes.NewReader(data))

			if ca.error {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, p.Tracks, len(ca.tracks))

			for i, expectedTrack := range ca.tracks {
				actualTrack := p.Tracks[i]
				require.Equal(t, expectedTrack.id, actualTrack.ID)
				require.Len(t, actualTrack.Samples, len(expectedTrack.sampleSizes))

				for j, sample := range actualTrack.Samples {
					require.Equal(t, expectedTrack.sampleSizes[j], sample.PayloadSize)

					// each track payload is filled with its track ID, hence
					// a cross-track address would return different bytes.
					pl, err2 := sample.GetPayload()
					require.NoError(t, err2)
					require.Equal(t, bytes.Repeat([]byte{byte(expectedTrack.id)}, int(expectedTrack.sampleSizes[j])), pl)
				}
			}
		})
	}
}

func TestPresentationUnmarshalTruncatedSampleTables(t *testing.T) {
	for _, ca := range []struct {
		name  string
		track testTrack
		mark  string
	}{
		{
			"inside stsz",
			testTrack{id: 1, sampleSizes: []uint32{2, 2}},
			"stsz",
		},
		{
			"inside stco",
			testTrack{id: 1, sampleSizes: []uint32{2, 2}},
			"stco",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			data := buildPresentation([]testTrack{ca.track})

			idx := bytes.Index(data, []byte(ca.mark))
			require.NotEqual(t, -1, idx)
			data = data[:idx+2]

			var p Presentation
			err := p.Unmarshal(bytes.NewReader(data))
			require.Error(t, err)

			for _, track := range p.Tracks {
				for _, sample := range track.Samples {
					require.Nil(t, sample.GetPayload)
				}
			}
		})
	}
}

func TestPresentationUnmarshalRoundTrip(t *testing.T) {
	p := Presentation{
		Tracks: []*Track{
			{
				ID:        1,
				TimeScale: 90000,
				Codec:     &codecs.Opus{ChannelCount: 2},
				Samples: []*Sample{
					{
						Duration:    90000,
						PayloadSize: 2,
						GetPayload:  func() ([]byte, error) { return []byte{1, 2}, nil },
					},
					{
						Duration:    90000,
						PayloadSize: 3,
						GetPayload:  func() ([]byte, error) { return []byte{3, 4, 5}, nil },
					},
				},
			},
			{
				ID:        2,
				TimeScale: 90000,
				Codec:     &codecs.Opus{ChannelCount: 1},
				Samples: []*Sample{
					{
						Duration:    90000,
						PayloadSize: 1,
						GetPayload:  func() ([]byte, error) { return []byte{6}, nil },
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := p.Marshal(&buf)
	require.NoError(t, err)

	var p2 Presentation
	err = p2.Unmarshal(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Len(t, p2.Tracks, 2)

	expected := getSampleData(t, &p)
	actual := getSampleData(t, &p2)
	require.Equal(t, expected, actual)
}
