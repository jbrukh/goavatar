//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package repo

import (
	. "github.com/jbrukh/goavatar/util"
	"os"
	"path/filepath"
	"testing"
)

const testBaseDir = "../var/unit-tests/test-repo"

// Test repository, basedir, and subdir
// creation.
func TestNewRepository(t *testing.T) {
	r, err := NewRepository(testBaseDir)

	// something wrong with constructor
	if err != nil {
		t.Errorf("could not create the directories")
	}

	// check basedir is set
	if r.Basedir() != testBaseDir {
		t.Errorf("basedir not set")
	}

	// check basedir created
	if _, err := os.Stat(testBaseDir); os.IsNotExist(err) {
		t.Errorf("basedir wasn't created")
	}

	// check that there are two subdirs
	if len(subdirSearchPath) != 2 {
		t.Errorf("should be local and cloud subdirs")
	}

	// check that we can retrieve them
	subdirs := r.Subdirs()
	if len(subdirs) != 2 {
		t.Errorf("should be able to get subdirs")
	}

	// check that they were created
	for _, subdir := range subdirs {
		if _, err := os.Stat(subdir); os.IsNotExist(err) {
			t.Errorf("did not create subdir")
		}
	}
}

func TestNewResourceId(t *testing.T) {
	r, err := NewRepository(testBaseDir)

	// something wrong with constructor
	if err != nil {
		t.Errorf("could not create the directories")
	}

	// test a local id
	id, fp := r.NewResourceIdWithSubdir(SubdirLocal)
	if id == "" || filepath.Dir(fp) != filepath.Join(r.basedir, SubdirLocal) || filepath.Base(fp) != id {
		t.Errorf("something went wrong with id generation")
	}

	// test cloud id
	id, fp = r.NewResourceIdWithSubdir(SubdirCloud)
	if id == "" || filepath.Dir(fp) != filepath.Join(r.basedir, SubdirCloud) || filepath.Base(fp) != id {
		t.Errorf("something went wrong with id generation")
	}

	// test a local id
	id, fp = r.NewResourceId()
	if id == "" || filepath.Dir(fp) != filepath.Join(r.basedir, SubdirLocal) || filepath.Base(fp) != id {
		t.Errorf("something went wrong with id generation")
	}

	// test invalid dir
	TestPanic(t, func() {
		r.NewResourceIdWithSubdir("nonsense")
	})
}

// Test file lookup.
func TestLookup(t *testing.T) {
	r, err := NewRepository(testBaseDir)

	// something wrong with constructor
	if err != nil {
		t.Errorf("could not create the directories")
	}

	ids := []string{"silly-id", "sillier-id", "silliest-id"}
	subdirs := r.Subdirs()
	testPaths := []string{
		filepath.Join(subdirs[0], ids[0]),
		filepath.Join(subdirs[1], ids[1]),
		filepath.Join(subdirs[0], ids[2]),
		filepath.Join(subdirs[1], ids[2]),
	}

	// create test resources
	if err := touchFile(testPaths[0]); err != nil {
		t.Errorf("could not touch test path")
	}

	if err := touchFile(testPaths[1]); err != nil {
		t.Errorf("could not touch test path")
	}

	if err := touchFile(testPaths[2]); err != nil {
		t.Errorf("could not touch test path")
	}

	if err := touchFile(testPaths[3]); err != nil {
		t.Errorf("could not touch test path")
	}

	// test the lookup
	if fp, err := r.Lookup(ids[0]); err != nil || fp != testPaths[0] {
		t.Errorf("bad lookup")
	}

	if fp, err := r.Lookup(ids[1]); err != nil || fp != testPaths[1] {
		t.Errorf("bad lookup")
	}

	if fp, err := r.Lookup(ids[2]); err != nil || fp != testPaths[2] {
		t.Errorf("bad lookup")
	}

}

func TestMove(t *testing.T) {
	r, err := NewRepository(testBaseDir)

	// something wrong with constructor
	if err != nil {
		t.Errorf("could not create the directories")
	}

	ids := []string{"silly-id", "sillier-id", "silliest-id"}
	subdirs := r.Subdirs()
	testPaths := []string{
		filepath.Join(subdirs[0], ids[0]),
		filepath.Join(subdirs[1], ids[1]),
		filepath.Join(subdirs[0], ids[2]),
		filepath.Join(subdirs[1], ids[2]),
	}

	// create test resources
	if err := touchFile(testPaths[0]); err != nil {
		t.Errorf("could not touch test path")
	}

	if err := touchFile(testPaths[1]); err != nil {
		t.Errorf("could not touch test path")
	}

	if err := touchFile(testPaths[2]); err != nil {
		t.Errorf("could not touch test path")
	}

	if err := touchFile(testPaths[3]); err != nil {
		t.Errorf("could not touch test path")
	}

	// move from local to cloud
	if err := r.Move(ids[0], SubdirCloud); err != nil || exists(testPaths[0]) {
		t.Errorf("could not move; err: %v; testPath: %s", err, testPaths[0])
	}

	// move from cloud to cloud
	if err := r.Move(ids[1], SubdirCloud); err != nil || !exists(testPaths[1]) {
		t.Errorf("could not move to the same dir")
	}
}

func touchFile(file string) error {
	_, err := os.OpenFile(file, os.O_CREATE, 0644)
	return err
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}
