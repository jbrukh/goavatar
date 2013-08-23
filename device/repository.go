//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	. "github.com/jbrukh/goavatar/util"
	"os"
	"path/filepath"
)

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

// subdirectories
const (
	SubdirLocal = "local"
	SubdirCloud = "cloud"
)

// max number of times to try to
// generate a resource id on clash
const maxGenerateRetries = 10

// subdirectory search path
var subdirSearchPath = []string{SubdirLocal, SubdirCloud}

// ----------------------------------------------------------------- //
// Repository
// ----------------------------------------------------------------- //

// A Repository encasulates an Octopus data repository. It is
// local to a user's computer and performs all the operations
// around storing the files.
//
// A Repository in the base directory may have a small number
// of subdirectories for logical grouping. By default, all files
// are stored in the 'local' subdirectory. A secondary directory
// for caching cloud data is called 'cloud'. Other subdirectories
// may come into play in the future.
//
// In a Repository, all OBF files are addressed by a unique resource
// id.  The id generation process is encapsulated inside the
// repository and you should use the NewResourceId() and
// NewResourceIdWithSubdir() methods to do so.

// An Octopus file repository.
type Repository struct {
	basedir string
}

// Create a new Repository. Will return an error
// if all the requisite directories could not be
// created.
func NewRepository(basedir string) (r *Repository, err error) {
	r = &Repository{basedir}

	// create the base directory
	if err = os.MkdirAll(basedir, 0755); err != nil {
		return nil, err
	}

	// create all the subdirs
	for _, subdir := range subdirSearchPath {
		fullPath := filepath.Join(basedir, subdir)
		if err = os.MkdirAll(fullPath, 0755); err != nil {
			return nil, err
		}
	}

	// ok!
	return
}

// Return the base directory of the repository.
func (r *Repository) Basedir() string {
	return r.basedir
}

// Return all known subdirs of this repository.
func (r *Repository) Subdirs() (subdirs []string) {
	for _, subdir := range subdirSearchPath {
		fullPath := filepath.Join(r.basedir, subdir)
		subdirs = append(subdirs, fullPath)
	}
	return
}

// Generate a new default id.
func (r *Repository) NewResourceId() (resourceId, resourcePath string) {
	return r.NewResourceIdWithSubdir(SubdirLocal)
}

// Generate a new id within a specified subdir.
func (r *Repository) NewResourceIdWithSubdir(subdir string) (resourceId, resourcePath string) {
	var fp, id string

	// check validity
	if subdir != SubdirLocal && subdir != SubdirCloud {
		panic("bad subdir")
	}

	// create a resource id
	for i := 1; i <= maxGenerateRetries; i++ {
		// try to generate the id
		id, _ = Uuid()
		fp = filepath.Join(r.basedir, subdir, id)

		// check for clash
		_, err := os.Stat(fp)
		if err == nil && i == maxGenerateRetries {
			// could not resolve clash
			panic("could not generate a unique resourceId, nothing to be done")
		}
	}

	// id is aquired, generate the path
	return id, fp
}
