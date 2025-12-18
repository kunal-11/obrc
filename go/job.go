package main

import (
	"fmt"
	"io"
	"os"
)

type job struct {
	f          *os.File
	start, end int64
	result     map[string]*jobResult
}

type jobResult struct {
	Sum   int64
	Count int
	Min   int
	Max   int
}

func (j *job) run() {
	itr := iterator{
		f:     j.f,
		start: j.start,
		end:   j.end,
		buf:   make([]byte, 1024*1024),
	}
	bufName := [512]byte{}
	for itr.HasNext() {
		cityLen, err := parseName(&itr, &bufName)
		if err != nil {
			fmt.Println("Error parsing name: ", err)
			os.Exit(1)
		}
		city := bufName[:cityLen]

		temp, err := parseTemp(&itr)
		if err != nil {
			fmt.Println("Error parsing temp: ", err)
			os.Exit(1)
		}

		cur, ok := j.result[string(city)]
		if !ok {
			cur = &jobResult{}
			j.result[string(city)] = cur
		}
		cur.Sum += int64(temp)
		cur.Count += 1
		cur.Min = min(cur.Min, temp)
		cur.Max = max(cur.Max, temp)
	}
}

func parseName(itr *iterator, name *[512]byte) (int, error) {
	for i := range 512 {
		next, err := itr.Next()
		if err != nil {
			return 0, err
		}
		if next == ';' {
			return i, nil
		}
		if next == '\n' {
			fmt.Println("unexpected newline at: ", itr.start+itr.read+int64(itr.i))
		}
		name[i] = next
	}
	return 0, fmt.Errorf("name not ended")
}

func parseTemp(itr *iterator) (int, error) {
	result := 0
	sign := 1
	for {
		num, err := itr.Next()
		if err != nil {
			return 0, err
		}

		if num == byte('\n') {
			return sign * result, nil
		}
		if num == '.' {
			continue
		}
		if num == '-' {
			sign = -1
			continue
		}
		if num >= '0' && num <= '9' {
			result *= 10
			result += int(num - byte('0'))
		}
	}
}

type iterator struct {
	f                *os.File
	start, end, read int64
	i, cur           int
	buf              []byte
}

func (itr *iterator) HasNext() bool {
	return int64(itr.i)+itr.read+itr.start <= itr.end
}

func (itr *iterator) Next() (byte, error) {
	if itr.i == itr.cur {
		itr.read += int64(itr.cur)

		read, err := itr.f.ReadAt(itr.buf, itr.start+itr.read)
		if err != nil && err != io.EOF {
			return 0, err
		}

		itr.cur = read
		itr.i = 0
	}
	itr.i++
	return itr.buf[itr.i-1], nil
}
