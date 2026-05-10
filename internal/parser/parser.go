package parser

import (
	"fmt"
	"io"
	"os"
	"bytes"
)

const alignBufSize = 4096

type ErrStatFailed struct {
	Path string
	Err  error
}

func (e *ErrStatFailed) Error() string {
	return fmt.Sprintf("parser: stat %q: %v", e.Path, e.Err)
}

func (e *ErrStatFailed) Unwrap() error { return e.Err }

type ErrOpenFailed struct {
	Path string
	Err  error
}

func (e *ErrOpenFailed) Error() string {
	return fmt.Sprintf("parser: open %q: %v", e.Path, e.Err)
}

func (e *ErrOpenFailed) Unwrap() error { return e.Err }

type ErrAlignFailed struct {
	Boundary string
	Err      error
}

func (e *ErrAlignFailed) Error() string {
	return fmt.Sprintf("parser: align %s: %v", e.Boundary, e.Err)
}

func (e *ErrAlignFailed) Unwrap() error { return e.Err }

type ErrReadChunkFailed struct {
	Start int64
	End   int64
	Err   error
}

func (e *ErrReadChunkFailed) Error() string {
	return fmt.Sprintf("parser: read chunk [%d, %d): %v", e.Start, e.End, e.Err)
}

func (e *ErrReadChunkFailed) Unwrap() error { return e.Err }

type Chunk struct {
	Start int64
	End   int64
}

type Parser struct {
	path string
}

type SplitResponse struct {
	Chunk []Chunk
	File  *os.File
	Err   error
}

func NewParser(path string) *Parser {
	return &Parser{path: path}
}

func alignToNewline(f *os.File, pos, size int64) (int64, error) {
	if pos >= size {
		return size, nil
	}
	buf := make([]byte, alignBufSize)
	for pos < size {
		toRead := int64(alignBufSize)
		if pos+toRead > size {
			toRead = size - pos
		}
		n, err := f.ReadAt(buf[:toRead], pos)
		if n > 0 {
			if idx := bytes.IndexByte(buf[:n], '\n'); idx >= 0 {
				return pos + int64(idx) + 1, nil
			}
			pos += int64(n)
		}
		if err == io.EOF {
			return size, nil
		}
		if err != nil {
			return pos, fmt.Errorf("parser: read bytes at %d: %w", pos, err)
		}
	}
	return size, nil
}

func (p *Parser) Split(n int) SplitResponse {
	info, err := os.Stat(p.path)
	if err != nil {
		return SplitResponse{Err: &ErrStatFailed{Path: p.path, Err: err}}
	}

	size := info.Size()
	if size == 0 {
		return SplitResponse{Chunk: []Chunk{}}
	}

	f, err := os.Open(p.path)
	if err != nil {
		return SplitResponse{Err: &ErrOpenFailed{Path: p.path, Err: err}}
	}

	if n <= 0 {
		n = 1
	}

	chunkSize := size / int64(n)
	if chunkSize == 0 {
		chunkSize = 1
	}

	var chunks []Chunk
	prevEnd := int64(0)

	for i := 0; i < n; i++ {
		start := int64(i) * chunkSize
		var end int64
		if i == n-1 {
			end = size
		} else {
			end = int64(i+1) * chunkSize
		}

		if i > 0 {
			aligned, err := alignToNewline(f, start-1, size)
			if err != nil {
				f.Close()
				return SplitResponse{Err: &ErrAlignFailed{Boundary: "start", Err: err}}
			}
			start = aligned
		}

		if i < n-1 {
			aligned, err := alignToNewline(f, end-1, size)
			if err != nil {
				f.Close()
				return SplitResponse{Err: &ErrAlignFailed{Boundary: "end", Err: err}}
			}
			end = aligned
		}

		if start < prevEnd {
			start = prevEnd
		}

		if start >= end {
			continue
		}

		chunks = append(chunks, Chunk{Start: start, End: end})
		prevEnd = end
	}

	return SplitResponse{Chunk: chunks, File: f}
}

func ReadChunk(f *os.File, c Chunk) ([]byte, error) {
	length := c.End - c.Start
	if length <= 0 {
		return []byte{}, nil
	}
	buf := make([]byte, length)
	_, err := f.ReadAt(buf, c.Start)
	if err != nil && err != io.EOF {
		return nil, &ErrReadChunkFailed{Start: c.Start, End: c.End, Err: err}
	}
	return buf, nil
}
