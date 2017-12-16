// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The wire protocol for HTTP's "chunked" Transfer-Encoding.

// Package internal contains HTTP internals shared by net/http and
// net/http/httputil.
package ipfs

import (
	"bufio"
	"bytes"
	"io"
	"time"
)

func NewChunkedReader(r io.Reader) *chunkedReader {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &chunkedReader{reader: br, delim: '\n'}
}

type chunkedReader struct {
	reader *bufio.Reader
	delim  byte
}

func (cr *chunkedReader) NextChunk() ([]byte, error) {
	var (
		chunkBytes []byte
		chunkEnd   bool = false
		err        error
	)
	for !chunkEnd {
		chunkBytes, err = cr.reader.ReadSlice(cr.delim)
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
				time.Sleep(1 * time.Second)
			}
			return nil, err
		} else {
			chunkBytes = bytes.TrimSpace(chunkBytes)
			chunkEnd = true
		}
	}
	return chunkBytes, nil
}
