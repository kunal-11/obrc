package main

import (
	"io"
	"os"
)

type Iterator struct {
	f          *os.File
	buf        []byte
	i, readLen int
	eof        bool
	read       int64
}

func (itr *Iterator) HasNext() error {
	if itr.i == itr.readLen {
		if itr.eof {
			return io.EOF
		} else {
			readLen, err := itr.f.Read(itr.buf)
			if err == io.EOF {
				itr.eof = true
				if readLen == 0 {
					return err
				}
			} else if err != nil {
				return err
			}
			itr.read += int64(itr.i)
			itr.readLen = readLen
			itr.i = 0
		}
	}
	return nil
}

func (itr *Iterator) Next() byte {
	itr.i += 1
	return itr.buf[itr.i-1]
}

func (itr *Iterator) BytesRead() int64 {
	return itr.read + int64(itr.i)
}
