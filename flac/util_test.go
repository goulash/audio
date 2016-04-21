// Copyright 2016 Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package flac

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadUint16(z *testing.T) {
	tests := []struct {
		In  []byte
		Out uint16
		Err error
	}{
		{[]byte{}, 0, ErrUnexpectedEOF},
		{[]byte{0xff}, 0, ErrUnexpectedEOF},
		{[]byte{0xff, 0x0}, 0xff00, nil},
		{[]byte{0x0, 0xff}, 0x00ff, nil},
	}

	assert := assert.New(z)
	for _, t := range tests {
		buf := bytes.NewBuffer(t.In)
		u, err := readUint16(buf)
		assert.Equal(t.Err, err)
		assert.Equal(t.Out, u)
	}
}

func TestReadUint24(z *testing.T) {
	tests := []struct {
		In  []byte
		Out uint32
		Err error
	}{
		{[]byte{}, 0, ErrUnexpectedEOF},
		{[]byte{0x0}, 0, ErrUnexpectedEOF},
		{[]byte{0xff, 0x0, 0x0}, 0xff0000, nil},
		{[]byte{0x0, 0xff, 0x0}, 0x00ff00, nil},
		{[]byte{0x0, 0x0, 0xff}, 0x0000ff, nil},
	}

	assert := assert.New(z)
	for _, t := range tests {
		buf := bytes.NewBuffer(t.In)
		u, err := readUint24(buf)
		assert.Equal(t.Err, err)
		assert.Equal(t.Out, u)
	}
}
