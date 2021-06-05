package datastore

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const KB = 1024
const numOfSegments = 2

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

	if len(db.segments) != numOfSegments {
		t.Error("Enought segmens are not created")
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

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Errorf("Walking error: %s", err)
		}

		if info.IsDir() {
			return nil
		}
		t.Log(info.Size())
		if info.Size() > KB {
			t.Errorf("segment %s: size is %d, but expected to be less than %d", path, info.Size(), KB)
		}
		return nil
	})

}
