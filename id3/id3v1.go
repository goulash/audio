// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package id3

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
)

// ErrInvalidID3v1 is an error which is returned when no ID3v1 header is found.
var ErrInvalidID3v1 = errors.New("invalid ID3v1 header")

// id3v1Genres is a list of genres as given in the ID3v1 specification.
var id3v1Genres = [...]string{
	"Blues", "Classic Rock", "Country", "Dance", "Disco", "Funk", "Grunge",
	"Hip-Hop", "Jazz", "Metal", "New Age", "Oldies", "Other", "Pop", "R&B",
	"Rap", "Reggae", "Rock", "Techno", "Industrial", "Alternative", "Ska",
	"Death Metal", "Pranks", "Soundtrack", "Euro-Techno", "Ambient",
	"Trip-Hop", "Vocal", "Jazz+Funk", "Fusion", "Trance", "Classical",
	"Instrumental", "Acid", "House", "Game", "Sound Clip", "Gospel",
	"Noise", "AlternRock", "Bass", "Soul", "Punk", "Space", "Meditative",
	"Instrumental Pop", "Instrumental Rock", "Ethnic", "Gothic",
	"Darkwave", "Techno-Industrial", "Electronic", "Pop-Folk",
	"Eurodance", "Dream", "Southern Rock", "Comedy", "Cult", "Gangsta",
	"Top 40", "Christian Rap", "Pop/Funk", "Jungle", "Native American",
	"Cabaret", "New Wave", "Psychadelic", "Rave", "Showtunes", "Trailer",
	"Lo-Fi", "Tribal", "Acid Punk", "Acid Jazz", "Polka", "Retro",
	"Musical", "Rock & Roll", "Hard Rock", "Folk", "Folk-Rock",
	"National Folk", "Swing", "Fast Fusion", "Bebob", "Latin", "Revival",
	"Celtic", "Bluegrass", "Avantgarde", "Gothic Rock", "Progressive Rock",
	"Psychedelic Rock", "Symphonic Rock", "Slow Rock", "Big Band",
	"Chorus", "Easy Listening", "Acoustic", "Humour", "Speech", "Chanson",
	"Opera", "Chamber Music", "Sonata", "Symphony", "Booty Bass", "Primus",
	"Porn Groove", "Satire", "Slow Jam", "Club", "Tango", "Samba",
	"Folklore", "Ballad", "Power Ballad", "Rhythmic Soul", "Freestyle",
	"Duet", "Punk Rock", "Drum Solo", "Acapella", "Euro-House", "Dance Hall",
}

// ReadID3v1Tags reads ID3v1 tags from the io.ReadSeeker.
// Returns ErrInvalidID3v1 if there are no ID3v1 tags,
// otherwise non-nil error if there was a problem.
func ReadID3v1(r io.ReadSeeker) (*MetadataID3v1, error) {
	_, err := r.Seek(-128, os.SEEK_END)
	if err != nil {
		return nil, err
	}

	if tag, err := readString(r, 3); err != nil {
		return nil, err
	} else if tag != "TAG" {
		return nil, ErrInvalidID3v1
	}

	title, err := readString(r, 30)
	if err != nil {
		return nil, err
	}

	artist, err := readString(r, 30)
	if err != nil {
		return nil, err
	}

	album, err := readString(r, 30)
	if err != nil {
		return nil, err
	}

	y, err := readString(r, 4)
	if err != nil {
		return nil, err
	}
	year, _ := strconv.Atoi(y)

	commentBytes, err := readBytes(r, 29)
	if err != nil {
		return nil, err
	}

	var comment string
	var track int
	if commentBytes[27] == 0 {
		comment = strings.TrimSpace(string(commentBytes[:28]))
		track = int(commentBytes[28])
	}

	var genre string
	genreID, err := readBytes(r, 1)
	if err != nil {
		return nil, err
	}
	if int(genreID[0]) < len(id3v1Genres) {
		genre = id3v1Genres[int(genreID[0])]
	}

	return &MetadataID3v1{
		title:   strings.TrimSpace(title),
		artist:  strings.TrimSpace(artist),
		album:   strings.TrimSpace(album),
		genre:   genre,
		year:    year,
		track:   track,
		comment: comment,
	}, nil
}

//var _ = audio.Metadata(new(MetadataID3v1))

type MetadataID3v1 struct {
	title   string
	artist  string
	album   string
	genre   string
	year    int
	track   int
	comment string
}
