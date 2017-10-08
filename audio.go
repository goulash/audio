// Copyright (c) 2016, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package audio

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"time"

	"github.com/dhowden/tag"
	"github.com/goulash/stat"
)

var Stats struct {
	Identify     stat.Run
	ReadMetadata stat.Run
	Transcode    stat.Run
}

type Codec int

const (
	Unknown Codec = iota

	WAV // Wave­form Audio File For­mat

	ALAC // Apple Lossless Audio Codec
	FLAC // Free Lossless Audio Codec
	APE  // Monkey's Audio
	OFR  // OptimFROG
	TAK  // Tom's verlustfreier Audiokompressor
	WV   // WavPack
	TTA  // True Audio
	WMAL // Windows Media Audio Lossless

	MP3  // MPEG-Layer 3 Audio
	M4A  // MPEG4 Audio
	M4B  // MPEG4 Audio Book
	M4P  // MPEG4 Protected Audio
	AAC  // Advanced Audio Coding
	OGG  // Vorbis
	OPUS // OPUS
	WMA  // Windows Media Audio
)

func (c Codec) String() string {
	switch c {
	case WAV:
		return "WAV"
	case ALAC:
		return "ALAC"
	case FLAC:
		return "FLAC"
	case APE:
		return "APE"
	case OFR:
		return "OFR"
	case TAK:
		return "TAK"
	case WV:
		return "WV"
	case TTA:
		return "TTA"
	case WMAL:
		return "WMAL"
	case MP3:
		return "MP3"
	case M4A:
		return "M4A"
	case M4B:
		return "M4B"
	case M4P:
		return "M4P"
	case AAC:
		return "AAC"
	case OGG:
		return "OGG"
	case OPUS:
		return "OPUS"
	case WMA:
		return "WMA"
	default:
		return "?"
	}
}

type Metadata interface {
	Title() string         // The primary song title
	Album() string         // The album the song belongs to
	Artist() string        // The primary performer/artist of the song
	AlbumArtist() string   // The album artist
	Composer() string      // The composer of the song
	Year() int             // The release year of the song (not necessarily recording year)
	Genre() string         // The genre of the song
	Track() (int, int)     // The track number of the song, 0 for unknown
	Disc() (int, int)      // The disc number of the song, 0 for unknown
	Length() time.Duration // The length of the song in milliseconds
	Comment() string       // A comment
	Copyright() string     // Copyright notice, usually in the form YYYY Name
	Website() string       // Website of the performer

	EncodedBy() string       // Who encoded the audio
	EncoderSettings() string // Any specific encoder settings
	Encoding() Codec         // Codec returns the file codec, or Unknown.
	EncodingBitrate() int    // Bitrate returns the file bitrate in Kbps, or -1 if unknown.

	OriginalFilename() string // The original filename of the song
}

func GuessIdentity(file string) Codec {
	return Unknown
}

func Identify(file string) (Codec, error) {
	start := time.Now()
	defer func() { Stats.Identify.Add(float64(time.Since(start))) }()

	f, err := os.Open(file)
	if err != nil {
		return Unknown, err
	}
	defer f.Close()

	_, ft, err := tag.Identify(f)
	if err != nil {
		return Unknown, err
	}

	switch ft {
	case tag.FLAC:
		return FLAC, nil
	case tag.OGG:
		return OGG, nil
	case tag.MP3:
		return MP3, nil
	case tag.M4A:
		return M4A, nil
	case tag.M4B:
		return M4B, nil
	case tag.M4P:
		return M4P, nil
	case tag.ALAC:
		return ALAC, nil
	default: // Unknown
		return Unknown, nil
	}
}

var MetadataReaders = make(map[Codec]func(string) (Metadata, error))

func ReadMetadata(file string) (Metadata, error) {
	start := time.Now()
	defer func() { Stats.ReadMetadata.Add(float64(time.Since(start))) }()

	c, err := Identify(file)
	if err != nil {
		return nil, err
	}
	f, ok := MetadataReaders[c]
	if !ok {
		return nil, errors.New("reading metadata for this codec unsupported")
	}
	return f(file)
}

type SystemDecoder interface {
	DecodeFileToStdout(input string) *exec.Cmd
}

var SystemDecoders = make(map[Codec]func() SystemDecoder)

type SystemEncoder interface {
	EncodeStdinToFile(output string, md Metadata) *exec.Cmd
}

var SystemEncoders = make(map[Codec]func() SystemEncoder)

// Transcode uses registered decoders and encoders to create decoder and encoder commands,
// which it then passes to TranscodeWith.
func Transcode(input, output string) ([]byte, error) {
	start := time.Now()
	defer func() { Stats.Transcode.Add(float64(time.Since(start))) }()

	md, err := ReadMetadata(input)
	if err != nil {
		return nil, err
	}
	ic := md.Encoding()
	decf, ok := SystemDecoders[ic]
	if !ok {
		return nil, errors.New("decoding for this codec unsupported")
	}
	oc := GuessIdentity(output)
	encf, ok := SystemEncoders[oc]
	if !ok {
		return nil, errors.New("encoding for this codec unsupported")
	}

	dec := decf().DecodeFileToStdout(input)
	enc := encf().EncodeStdinToFile(output, md)
	return TranscodeWith(dec, enc)
}

// TranscodeWith takes two commands, pipes the output from dec to the input of enc.
//
// Ideally, we would like to be able to run both of the commands like:
//
//	dec := flac.NewDecoder()
//  enc := opus.NewEncoder()
//	audio.TranscodeWith(dec.NewStdoutDecoder(input), enc.NewStdinEncoder(output, audio.ReadMetadata(input)))
func TranscodeWith(dec, enc *exec.Cmd) ([]byte, error) {
	// Set up the pipe
	var err error
	enc.Stdin, err = dec.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// Set up the combined output
	var b bytes.Buffer
	dec.Stderr = &b
	enc.Stdout = &b
	enc.Stderr = &b

	// Run both commands
	if err := enc.Start(); err != nil {
		return b.Bytes(), err
	}
	if err := dec.Run(); err != nil {
		return b.Bytes(), err
	}
	return b.Bytes(), enc.Wait()
}
