package datastore

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

const outFileName = "current-data"

const marker = "marker"

const mergingSegmentsNum = 2

var ErrNotFound = fmt.Errorf("record does not exist")

type hashIndex map[string]int64

type Db struct {
	segments    []*Segment
	segmentSize int64
	dir         string

	mutex           sync.Mutex
	mergeInProgress bool
}

func NewDb(dir string, segmentSize int64) (*Db, error) {
	db := &Db{
		segments:    []*Segment{},
		segmentSize: segmentSize,
		dir:         dir,
	}
	err := db.recover()
	if err != nil && err != io.EOF {
		return nil, err
	}
	_, err = db.createDbSegment()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (db *Db) createDbSegment() (*Segment, error) {
	name := time.Now().UnixNano()
	segmentPath := filepath.Join(db.dir, strconv.FormatInt(name, 10))

	sgm, err := NewSegment(true, segmentPath, db.segmentSize)

	if err != nil {
		return nil, err
	}

	db.segments = append(db.segments, sgm)

	if len(db.segments) > mergingSegmentsNum {
		go db.mergeDbSegments()
	}
	return sgm, err
}

func (db *Db) mergeDbSegments() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if db.mergeInProgress == true {
		return nil
	}

	db.mergeInProgress = true

	mergeList := db.segments[0:mergingSegmentsNum]

	data := make(map[string]string)

	for _, sgm := range mergeList {
		allData, err := sgm.GetAllData()

		if err != nil {
			return err
		}

		for key, val := range allData {

			if val == marker {
				delete(data, key)
			} else {
				data[key] = val
			}
		}

		err1 := os.Remove(sgm.outPath)

		if err1 != nil {
			return err1
		}

	}

	sgm, err := NewSegment(true, mergeList[0].outPath+mergeList[1].outPath, db.segmentSize)

	if err != nil {
		return err
	}

	for key, val := range data {
		errorChannel := make(chan error)
		sgm.Put(ChannelData{
			data: entry{
				key:   key,
				value: val,
			},
			errorChannel: errorChannel,
		})
		err := <-errorChannel
		if err != nil {
			return err
		}
	}

	sgm.removeWritingLoop()

	segments := append([]*Segment{sgm}, db.segments[mergingSegmentsNum:]...)
	db.segments = segments

	db.mergeInProgress = false
	return nil
}

func (db *Db) recover() error {
	files, err := ioutil.ReadDir(db.dir)
	if err != nil {
		return err
	}
	var segments []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		segments = append(segments, file.Name())
	}
	sort.SliceStable(segments, func(i, j int) bool {
		iTime, err := strconv.Atoi(segments[i])
		if err != nil {
			log.Fatal(err)
		}
		jTime, err := strconv.Atoi(segments[j])
		if err != nil {
			log.Fatal(err)
		}
		return time.Unix(0, int64(iTime)).Before(time.Unix(0, int64(jTime)))
	})
	for _, name := range segments {
		path := filepath.Join(db.dir, name)
		sgm, err := NewSegment(false, path, db.segmentSize)
		if err != nil {
			return err
		}
		err = sgm.recover()
		if err != nil && err != io.EOF {
			return err
		}
		db.segments = append(db.segments, sgm)
	}
	return err
}

func (db *Db) Close() error {
	for _, sgm := range db.segments {
		err := sgm.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *Db) Get(key string) (string, error) {
	sgms := db.segments

	for _, sgm := range sgms {
		val, err := sgm.Get(key)

		if err == nil {
			return val, nil
		}
		if err != ErrNotFound {
			return "", err
		}
	}

	return "", fmt.Errorf("Some end error")
}

func (db *Db) Put(key, value string) error {
	e := entry{
		key:   key,
		value: value,
	}

	currentSegment := db.segments[len(db.segments)-1]

	currentOffset := currentSegment.outOffset

	if currentOffset+int64(len(value)) > db.segmentSize {
		currentSegment.removeWritingLoop()
		currentSegment.out.Close()
		sgm, err := db.createDbSegment()

		if err != nil {
			return err
		}

		currentSegment = sgm
	}
	errorChannel := make(chan error)
	err := currentSegment.Put(ChannelData{
		data:         e,
		errorChannel: errorChannel,
	})
	if err != nil {
		return err
	}
	return <-errorChannel
}

func (db *Db) Delete(key string) error {
	err := db.Put(key, marker)

	return err
}
