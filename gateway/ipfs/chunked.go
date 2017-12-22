/*

  Copyright 2017 Loopring Project Ltd (Loopring Foundation).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

*/

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
