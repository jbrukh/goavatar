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
	if len(ValidSubdirs) != 2 {
		t.Errorf("should be local and cache subdirs")
	}

	// check that we can retrieve them
	searchPath := r.SearchPath()
	if len(searchPath) != 2 {
		t.Errorf("should be able to get subdirs")
	}

	// check that they were created
	for _, subdir := range searchPath {
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
	var (
		SubdirLocal = "local"
		SubdirCache = "cache"
	)
	id, fp := r.NewResourceIdWithSubdir(SubdirLocal)
	if id == "" || filepath.Dir(fp) != filepath.Join(r.basedir, SubdirLocal) || filepath.Base(fp) != id {
		t.Errorf("something went wrong with id generation")
	}

	// test cached id
	id, fp = r.NewResourceIdWithSubdir(SubdirCache)
	if id == "" || filepath.Dir(fp) != filepath.Join(r.basedir, SubdirCache) || filepath.Base(fp) != id {
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
	searchPath := r.SearchPath()
	testPaths := []string{
		filepath.Join(searchPath[0], ids[0]),
		filepath.Join(searchPath[1], ids[1]),
		filepath.Join(searchPath[0], ids[2]),
		filepath.Join(searchPath[1], ids[2]),
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
	searchPath := r.SearchPath()
	testPaths := []string{
		filepath.Join(searchPath[0], ids[0]),
		filepath.Join(searchPath[1], ids[1]),
		filepath.Join(searchPath[0], ids[2]),
		filepath.Join(searchPath[1], ids[2]),
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

	var SubdirCache = "cache"

	// move from local to cloud
	if err := r.move(ids[0], SubdirCache); err != nil || exists(testPaths[0]) {
		t.Errorf("could not move; err: %v; testPath: %s", err, testPaths[0])
	}

	// move from cloud to cloud
	if err := r.move(ids[1], SubdirCache); err != nil || !exists(testPaths[1]) {
		t.Errorf("could not move to the same dir")
	}
}

func TestListing(t *testing.T) {
	r, err := NewRepository(testBaseDir)

	// something wrong with constructor
	if err != nil {
		t.Errorf("could not create the directories")
	}

	ids := []string{
		"30dbbd1d-6426-baaf-9eab-29ad6e6740fc",
		"37b221be-753b-9bfc-52aa-8c1aade21399",
		"3ef4019c-b6f8-e44d-25f3-907240f52978",
	}
	subdir := "local"
	fakeDir := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	// create the test files
	for _, id := range ids {
		if err := touchFile(r.resourcePath(subdir, id)); err != nil {
			t.Errorf("could not touch test path")
		}
	}

	var checkListing = func() {
		infos, err := r.list(subdir)
		if err != nil {
			t.Errorf("could not get listing")
		}

		if len(infos) != len(ids) {
			t.Errorf("wrong number of files in listing: %v", len(infos))
		}
	}
	checkListing()

	// now add an invalid id, which will be ignored
	if err := touchFile(r.resourcePath(subdir, "not-valid-id")); err != nil {
		t.Errorf("could not touch test path")
	}
	checkListing()

	// now add a directory in there
	if err := os.MkdirAll(r.resourcePath(subdir, fakeDir), 0755); err != nil {
		t.Errorf("could not create directory: %v", err)
	}
	checkListing()
}

func touchFile(file string) error {
	_, err := os.OpenFile(file, os.O_CREATE, 0644)
	return err
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}
