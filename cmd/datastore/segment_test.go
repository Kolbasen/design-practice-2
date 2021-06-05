package datastore

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

const KB = 1024
const numOfSegments = 3

func createUniqueString(i int) string {
	return fmt.Sprintf("%010d", i)
}

const dataIteration = KB * numOfSegments / 32

func Test_Segment(t *testing.T) {
	dir, err := ioutil.TempDir(".", "test-segment-*")
	if err != nil {
		t.Fatal("Creating db store error")
	}
	defer os.RemoveAll(dir)

	db, err := NewDb(dir, KB)
	if err != nil {
		t.Fatal("Creating db error", err)
	}

	for i := 0; i < dataIteration; i++ {
		key := createUniqueString(i)
		err := db.Put(key, key)
		if err != nil {
			t.Errorf("Put db error: %s", err)
		}
	}

	time.Sleep(time.Duration(100 * time.Millisecond))

	if len(db.segments) != 2 {
		t.Error("Segments are not merged")
	}

	for i := 0; i < dataIteration; i++ {
		key := createUniqueString(i)
		val, err := db.Get(key)
		if err != nil {
			t.Errorf("Get db error: %s", err)
		}
		if val != key {
			t.Errorf("Compare val error: %d", err)
		}
	}

	files, err := ioutil.ReadDir(dir)

	if len(files) > 2 {
		t.Error("Too many file in dir")
	}
}
