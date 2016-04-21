// Copyright 2016 Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package flac

import (
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

func TestFile(z *testing.T) {
	assert := assert.New(z)
	f, err := os.Open(testFile)
	assert.Nil(err)
	m, err := ReadMetadata(f)
	assert.Nil(err)
	si, ti := testFileStreamInfo, m.StreamInfo()
	assert.Equal(si.MinBlockSize, ti.MinBlockSize)
	assert.Equal(si.MaxBlockSize, ti.MaxBlockSize)
	assert.Equal(si.MinFrameSize, ti.MinFrameSize)
	assert.Equal(si.MaxFrameSize, ti.MaxFrameSize)
	assert.Equal(si.SampleRate, ti.SampleRate)
	assert.Equal(si.NumChannels, ti.NumChannels)
	assert.Equal(si.BitsPerSample, ti.BitsPerSample)
	assert.Equal(si.TotalSamples, ti.TotalSamples)
	assert.Equal(si.MD5Sum, ti.MD5Sum)
}
