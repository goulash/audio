// Copyright 2016 Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package flac

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testWAV       = "test.wav"
	testFile      = "test.flac"
	testCover     = "cover.jpg"
	testPerformer = "performer.jpg"
)

var testFileStreamInfoHeader = 0x00000022
var testFileStreamInfo = &StreamInfo{
	MinBlockSize:  4096,
	MaxBlockSize:  4096,
	MinFrameSize:  339,
	MaxFrameSize:  9008,
	SampleRate:    44100,
	NumChannels:   2,
	BitsPerSample: 16,
	TotalSamples:  16536,
	MD5Sum:        []byte("ce88fffba66d962c99bdd809c73d4d18"),
}

func TestReadStreamMarker(z *testing.T) {
	tests := map[string]error{
		"fLaC":    nil,
		"FLAC":    ErrInvalidStream,
		"fLaC...": nil,
		"fLa":     ErrUnexpectedEOF,
		"":        ErrUnexpectedEOF,
	}

	for k, v := range tests {
		buf := bytes.NewBufferString(k)
		err := readStreamMarker(buf)
		if err != v {
			z.Errorf("readStreamMarker(%q) = %v, expecting %v", k, err, v)
		}
	}
}

func TestReadBlockHeader(z *testing.T) {
	assert := assert.New(z)
	tests := []struct {
		In   []byte
		Out  uint32
		Last bool
		Type blockType
		Size int64
		Err  error
	}{
		{[]byte{0x0, 0x0, 0x0, 0x22}, 34, false, streamInfoBlock, 34, nil},
		{[]byte{0x80, 0x0, 0x0, 0x0}, 2147483648, true, streamInfoBlock, 0, nil},
		{[]byte{0x80, 0x0, 0x0}, 0, false, streamInfoBlock, 0, ErrUnexpectedEOF},
	}

	for _, t := range tests {
		buf := bytes.NewBuffer(t.In)
		h, err := readBlockHeader(buf)
		assert.Equal(t.Out, uint32(h))
		assert.Equal(t.Err, err)
		assert.Equal(t.Last, h.IsLast())
		assert.Equal(t.Type, h.Type())
		assert.Equal(t.Size, h.Length())
	}
}

func TestFile(z *testing.T) {
	assert := assert.New(z)
	f, err := os.Open(testFile)
	if !assert.Nil(err) {
		return
	}
	m, err := ReadMetadata(f)
	if !assert.Nil(err) {
		return
	}
	si, ti := testFileStreamInfo, m.StreamInfo()
	assert.Equal(si.MinBlockSize, ti.MinBlockSize)
	assert.Equal(si.MaxBlockSize, ti.MaxBlockSize)
	assert.Equal(si.MinFrameSize, ti.MinFrameSize)
	assert.Equal(si.MaxFrameSize, ti.MaxFrameSize)
	assert.Equal(si.SampleRate, ti.SampleRate)
	assert.Equal(si.NumChannels, ti.NumChannels)
	assert.Equal(si.BitsPerSample, ti.BitsPerSample)
	assert.Equal(si.TotalSamples, ti.TotalSamples)
	assert.Equal(string(si.MD5Sum), hex.EncodeToString(ti.MD5Sum))
}
