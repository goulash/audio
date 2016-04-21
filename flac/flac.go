// Copyright 2016 Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// Package flac implements FLAC decoding.
//
// Reference
//
// https://xiph.org/flac/format.html
package flac

import (
	"errors"
	"io"
	"time"
)

var (
	ErrUnexpectedEOF = errors.New("unexpected EOF")
	ErrInvalidStream = errors.New("stream is invalid")
)

func ReadMetadata(r io.Reader) (*Metadata, error) {
	err := readStreamMarker(r)
	if err != nil {
		return nil, err
	}
	return readMetadata(r)
}

// Stream Marker {{{

func readStreamMarker(r io.Reader) error {
	s, err := readString(r, 4)
	if err != nil {
		return err
	}
	if s != "fLaC" {
		return ErrInvalidStream
	}
	return nil
}

// }}}

// Metadata {{{

func readMetadata(r io.Reader) (*Metadata, error) {
	m := Metadata{
		bytes: 4,
	}

	for {
		h, err := readBlockHeader(r)
		if err != nil {
			return nil, err
		}
		m.bytes += h.Length() + 4

		switch h.Type() {
		case streamInfoBlock:
			si, err := readStreamInfoBlock(r, h)
			if err != nil {
				return nil, err
			}
			m.info = si
		case paddingBlock:
			if err := readPaddingBlock(r, h); err != nil {
				return nil, err
			}
		case applicationBlock:
			if err := readApplicationBlock(r, h); err != nil {
				return nil, err
			}
		case seektableBlock:
			if err := readSeekTableBlock(r, h); err != nil {
				return nil, err
			}
		case vorbisCommentBlock:
			raw, err := readVorbisCommentBlock(r, h)
			if err != nil {
				return nil, err
			}
			m.raw = raw
		case cuesheetBlock:
			if err := readCuesheetBlock(r, h); err != nil {
				return nil, err
			}
		case pictureBlock:
			if err := readPictureBlock(r, h); err != nil {
				return nil, err
			}
		case invalidBlock:
			return nil, ErrInvalidStream
		default:
			// The standard allows for new block types to be defined.
			// We can either die or ignore them. For our purpose, it
			// is better to ignore them, which as far as the implementation
			// goes, is basically the same as padding.
			readPaddingBlock(r, h)
		}

		if h.IsLast() {
			break
		}
	}

	return &m, nil
}

type Metadata struct {
	bytes int64
	info  *StreamInfo
	raw   map[string][]string
}

func (m *Metadata) StreamInfo() *StreamInfo { return m.info }

func (m *Metadata) Length() time.Duration { return m.info.Duration() }

func (m *Metadata) Bitrate(filesize int64) int {
	z := filesize - m.bytes
	d := m.Length()
	kbps := z / int64(d*1000/time.Second)
	if kbps <= 0 {
		return -1
	}
	return int(kbps)
}

// Metadata Block Header {{{

// readBlockHeader reads 4 bytes.
func readBlockHeader(r io.Reader) (blockHeader, error) {
	v, err := readUint32(r)
	return blockHeader(v), err
}

type blockHeader uint32
type blockType int16

const (
	streamInfoBlock = iota
	paddingBlock
	applicationBlock
	seektableBlock
	vorbisCommentBlock
	cuesheetBlock
	pictureBlock

	invalidBlock blockType = 127
)

func (h blockHeader) IsLast() bool    { return h&0x80000000 != 0 }           // true only when bit 0 is set
func (h blockHeader) Type() blockType { return blockType((h >> 24) & 0x7F) } // the type is in bit 1:8
func (h blockHeader) Length() int64   { return int64(h & 0x00FFFFFF) }       // the last 24 bits
func (h blockHeader) IsValid() bool   { return h.Type() != invalidBlock }    // this is the only thing that can be invalid

// }}}

// Metadata Block: STREAMINFO {{{

func readStreamInfoBlock(r io.Reader, _ blockHeader) (*StreamInfo, error) {
	si := StreamInfo{}

	// Read minimum (16) and maximum (16) block size
	p, err := readUint32(r)
	if err != nil {
		return nil, err
	}
	si.MinBlockSize = uint16(p >> 16)
	si.MaxBlockSize = uint16(p & 0xFFFF)

	// Read minimum (24) and maximum (24) frame size
	x, err := readUint48(r)
	if err != nil {
		return nil, err
	}
	si.MinFrameSize = uint32(x >> 24)
	si.MaxFrameSize = uint32(x & 0xFFFFFF)

	// Read sample rate (20), number of channels (3), bits per sample (5), and total samples (36)
	x, err = readUint64(r)
	if err != nil {
		return nil, err
	}
	si.SampleRate = uint32(x >> 44)
	si.NumChannels = uint8((x >> 41) & 0x07)
	si.BitsPerSample = uint8((x >> 36) & 0x1F)
	si.TotalSamples = uint64(x & 0x0FFFFFFFFF)

	// Read md5 sum (128)
	si.MD5Sum, err = readBytes(r, 16)
	if err != nil {
		return nil, err
	}

	return &si, nil
}

type StreamInfo struct {
	// MinBlockSize is the minimum block size (in samples) used in the stream.
	// The minimum block size is 16.
	MinBlockSize uint16
	// MaxBlockSize is the maximum block size (in samples) used in the stream.
	// A fixed-block-size stream is implied by MinBlockSize == MaxBlockSize.
	// The maximum block size is 65535.
	MaxBlockSize uint16

	// MinFrameSize is the minimum frame size (in bytes) used in the stream.
	// It may be 0 to imply that the value is unknown.
	MinFrameSize uint32 // only 24 bits are used
	// MaxFrameSize is the maximum frame size (in bytes) used in the stream.
	// It may be 0 to imply that the value is unknown.
	MaxFrameSize uint32

	// SampleRate is the sample rate in Hz, and must be greater than 0 and
	// less-or-equal to 655350. This limitation comes from the structure of
	// the frames (it is not a typo).
	SampleRate uint32

	// NumChannels is the number of channels, which range from 1 to 8.
	NumChannels uint8

	// BitsPerSample is the number of bits per sample, which can range from 4 to 32 bits.
	BitsPerSample uint8

	// TotalSamples is the total number of samples in the stream. This is
	// not dependent on the number of channels.
	TotalSamples uint64

	// MD5Sum is an MD5 signature of the unencoded audio data. This allows
	// the decoder to determine if an error exists in the audio data even
	// when the error does not result in an invalid bitstream.
	MD5Sum []byte
}

// Duration returns the total duration of the stream, or zero if it is unknown.
// This is calculated by TotalSamples*time.Second / SampleRate
func (si *StreamInfo) Duration() time.Duration {
	return time.Duration(si.TotalSamples) * time.Second / time.Duration(si.SampleRate)
}

// }}}

// Metadata Block: PADDING {{{

func readPaddingBlock(r io.Reader, h blockHeader) error {
	_, err := readBytes(r, int(h.Length()))
	return err
}

// }}}

// Metadata Block: APPLICATION {{{

func readApplicationBlock(r io.Reader, h blockHeader) error {
	// TODO: not implemented yet
	_, err := readBytes(r, int(h.Length()))
	return err
}

// }}}

// Metadata Block: SEEKTABLE {{{

func readSeekTableBlock(r io.Reader, h blockHeader) error {
	// TODO: not implemented yet
	_, err := readBytes(r, int(h.Length()))
	return err
}

// }}}

// Metadata Block: VORBIS_COMMENT {{{

func readVorbisCommentBlock(r io.Reader, h blockHeader) (map[string][]string, error) {
	// TODO: not implemented yet
	_, err := readBytes(r, int(h.Length()))
	return nil, err
}

// }}}

// Metadata Block: CUESHEET {{{

func readCuesheetBlock(r io.Reader, h blockHeader) error {
	// TODO: not implemented yet
	_, err := readBytes(r, int(h.Length()))
	return err
}

// }}}

// Metadata Block: PICTURE {{{

func readPictureBlock(r io.Reader, h blockHeader) error {
	// TODO: not implemented yet
	_, err := readBytes(r, int(h.Length()))
	return err
}

// }}}
