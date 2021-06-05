package datastore

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

type ChannelData struct {
	data         entry
	errorChannel chan error
}

type Segment struct {
	out       *os.File
	outPath   string
	outOffset int64
	index     hashIndex

	size int64

	mutex          sync.Mutex
	writingChannel chan ChannelData
}

func NewSegment(isActive bool, outPath string, size int64) (*Segment, error) {
	f, err := os.OpenFile(outPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)

	out := f

	if err != nil {
		return nil, err
	}

	if !isActive {
		out = nil
		f.Close()
		out.Close()
	}

	smg := &Segment{
		outPath:   outPath,
		outOffset: 0,
		size:      size,
		index:     map[string]int64{},
		out:       out,
	}

	if isActive {
		smg.writingChannel = make(chan ChannelData)
		go smg.writingLoop()
	}

	return smg, nil
}

const bufSize = 8192

func (sgm *Segment) recover() error {
	input, err := os.Open(sgm.outPath)
	if err != nil {
		return err
	}
	defer input.Close()

	var buf [bufSize]byte
	in := bufio.NewReaderSize(input, bufSize)
	for err == nil {
		var (
			header, data []byte
			n            int
		)
		header, err = in.Peek(bufSize)
		if err == io.EOF {
			if len(header) == 0 {
				return err
			}
		} else if err != nil {
			return err
		}
		size := binary.LittleEndian.Uint32(header)

		if size < bufSize {
			data = buf[:size]
		} else {
			data = make([]byte, size)
		}
		n, err = in.Read(data)

		if err == nil {
			if n != int(size) {
				return fmt.Errorf("corrupted file")
			}

			var e entry
			e.Decode(data)
			sgm.index[e.key] = sgm.outOffset
			sgm.outOffset += int64(n)
		}
	}
	return err
}

func (sgm *Segment) Close() error {
	return sgm.out.Close()
}

func (sgm *Segment) GetAllData() (map[string]string, error) {
	all := make(map[string]string)

	for key := range sgm.index {
		val, err := sgm.Get(key)

		if err != nil {
			return nil, err
		}
		all[key] = val
	}
	return all, nil
}

func (sgm *Segment) Get(key string) (string, error) {
	position, ok := sgm.index[key]
	if !ok {
		return "", ErrNotFound
	}

	file, err := os.Open(sgm.outPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Seek(position, 0)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(file)
	value, err := readValue(reader)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (sgm *Segment) Put(data ChannelData) error {
	if sgm.writingChannel == nil {
		return fmt.Errorf("No writing channel")
	}

	sgm.writingChannel <- data

	return nil
}

func (sgm *Segment) writingLoop() error {
	if sgm.writingChannel == nil {
		return fmt.Errorf("No writing channel")
	}

	for channelData := range sgm.writingChannel {
		sgm.mutex.Lock()

		data := channelData.data

		n, err := sgm.out.Write(data.Encode())

		if err != nil {
			channelData.errorChannel <- err
		}

		if data.value != marker {
			sgm.index[data.key] = sgm.outOffset
		}
		sgm.outOffset += int64(n)

		if sgm.outOffset > sgm.size {
			sgm.out.Close()
			sgm.out = nil
		}

		channelData.errorChannel <- err

		sgm.mutex.Unlock()
	}

	return nil
}

func (sgm *Segment) removeWritingLoop() {
	close(sgm.writingChannel)
	sgm.writingChannel = nil
}
