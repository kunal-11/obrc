package main

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type part struct {
	start, end int64
}

var pool *sync.Pool

func calcParts(file *os.File, parts int) ([]part, error) {
	stats, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error reading file stats: %w", err)
	}
	fileSize := stats.Size()
	partLen := fileSize / int64(parts)

	offsets := make([]part, 0, parts)
	offset := int64(0)
	for range parts {
		end, err := findNewLine(file, offset+partLen)
		if err == io.EOF {
			end = min(end, fileSize-1)
		} else if err != nil {
			return nil, err
		}
		offsets = append(offsets, part{start: offset, end: end})
		offset = end + 1
	}
	return offsets, nil
}

func findNewLine(file *os.File, offset int64) (int64, error) {
	buf := make([]byte, 2048)
	read, err := file.ReadAt(buf, offset)
	if err != io.EOF && err != nil {
		return 0, err
	}
	for i := range read {
		if buf[i] == '\n' {
			return offset + int64(i), nil
		}
	}
	if err != io.EOF {
		return 0, fmt.Errorf("newline not found after %v bytes", read)
	}
	return offset + int64(read), io.EOF
}

func ReadFile(filepath string, chunks, bufLen int) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}

	parts, err := calcParts(file, chunks)
	if err != nil {
		return err
	}

	pool = &sync.Pool{
		New: func() any {
			return make([]byte, bufLen)
		},
	}
	ch := make(chan []byte, 1024)

	go worker(ch)

	wg := sync.WaitGroup{}
	for _, chunk := range parts {
		wg.Go(func() {
			if err := readChunks(file, chunk.start, chunk.end, ch); err != nil {
				panic(err)
			}
		})
	}
	wg.Wait()
	close(ch)

	return nil
}

func readChunks(file *os.File, start, end int64, ch chan []byte) error {
	for start <= end {
		buf := pool.Get().([]byte)
		readLen, err := file.ReadAt(buf, start)
		if err == io.EOF {
			ch <- buf[:readLen]
			break
		} else if err != nil {
			return err
		}
		i := readLen - 1
		for i >= 0 && buf[i] != '\n' {
			i--
		}
		ch <- buf[:i+1]
		start += int64(i + 1)
	}
	return nil
}

func worker(ch chan []byte) {
	for buf := range ch {
		pool.Put(buf[:cap(buf)])
	}
}

func main() {
	err := ReadFile("../../data/measurements.txt", 6, 1024*1024*16)
	if err != nil {
		panic(err)
	}
}
