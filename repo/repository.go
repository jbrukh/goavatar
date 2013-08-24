//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package repo

import (
	"fmt"
	. "github.com/jbrukh/goavatar/util"
	"os"
	"path/filepath"
	"regexp"
)

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

// max number of times to try to
// generate a resource id on clash
const maxGenerateRetries = 10

// regex for a resourceId
const resourceRegex = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

// subdirectories
var ValidSubdirs = []string{
	"local",
	"cache",
}

// the default subdir, where all
// data goes first or by default
var DefaultSubdir = ValidSubdirs[0]

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
	basedir    string
	searchPath []string
}

// Create a new Repository. Will return an error
// if all the requisite directories could not be
// created.
func NewRepository(basedir string) (r *Repository, err error) {
	r = &Repository{
		basedir: basedir,
	}

	// create the base directory
	if err = os.MkdirAll(basedir, 0755); err != nil {
		return nil, err
	}

	// create all the subdirs
	for _, subdir := range ValidSubdirs {
		subdirPath := r.subdirPath(subdir)
		if err = os.MkdirAll(subdirPath, 0755); err != nil {
			return nil, err
		}
		r.searchPath = append(r.searchPath, subdirPath)
	}

	// ok!
	return
}

// Create a new repository or panic.
func NewRepositoryOrPanic(basedir string) *Repository {
	r, err := NewRepository(basedir)
	if err != nil {
		panic("could not create a new repository")
	}
	return r
}

// Return the base directory of the repository.
func (r *Repository) Basedir() string {
	return r.basedir
}

// Return all directories on the search path of
// of this repository.
func (r *Repository) SearchPath() (searchPath []string) {
	return r.searchPath
}

// Generate a new default id.
func (r *Repository) NewResourceId() (resourceId, resourcePath string) {
	return r.NewResourceIdWithSubdir(DefaultSubdir)
}

// Generate a new id within a specified subdir.
func (r *Repository) NewResourceIdWithSubdir(subdir string) (resourceId, resourcePath string) {
	var fp, id string

	// check validity
	if !isValidSubdir(subdir) {
		panic(fmt.Sprintf("bad subdir: %s", subdir))
	}

	// create a resource id
	for i := 1; i <= maxGenerateRetries; i++ {
		// try to generate the id
		id, _ = Uuid()
		fp = r.resourcePath(subdir, id)

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

// Look up a resource by its resource id. The search path
// will be checked.
func (r *Repository) Lookup(resourceId string) (resourcePath string, err error) {
	for _, subdir := range ValidSubdirs {
		fp := r.resourcePath(subdir, resourceId)
		if _, err = os.Stat(fp); err == nil {
			return fp, nil
		}
	}
	return "", fmt.Errorf("no such resource in search path: %v", resourceId)
}

// Move a file into the cache subdir for backup.
func (r *Repository) Cache(resourceId string) (err error) {
	return r.move(resourceId, "cache")
}

// List will list all the resources in the default subdir.
func (r *Repository) List() (infos []os.FileInfo, err error) {
	return r.list(DefaultSubdir)
}

// List will list all the resources in the cache.
func (r *Repository) ListCache() (infos []os.FileInfo, err error) {
	return r.list("cache")
}

// ----------------------------------------------------------------- //
// Private Repo Operations
// ----------------------------------------------------------------- //

// List all the resources in a subdirectory.
func (r *Repository) list(subdir string) (infos []os.FileInfo, err error) {
	err = r.forEach(subdir, func(path string, f os.FileInfo) error {
		infos = append(infos, f)
		return nil
	})
	return infos, err
}

// Cache a resource in a particular subdirectory
func (r *Repository) move(resourceId, subdir string) (err error) {
	pth, err := r.Lookup(resourceId)
	if err != nil {
		return err
	}

	newPath := r.resourcePath(subdir, resourceId)
	if newPath == pth {
		return
	}

	// do the renaming
	if err := os.Rename(pth, newPath); err != nil {
		return err
	}

	return nil
}

// Remove all the files in a subdir.
func (r *Repository) clear(subdir string) (err error) {
	return r.forEach(subdir, func(path string, f os.FileInfo) error {
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprint(os.Stderr, "failed to remove the file: %v (err: %v)", path, err)
		}
		return nil
	})
}

// Perform an operation for each resource in a subdir.
func (r *Repository) forEach(subdir string, op func(path string, f os.FileInfo) error) (err error) {
	root := r.subdirPath(subdir)
	err = filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.IsDir() {
			doesMatch, err := regexp.MatchString(resourceRegex, f.Name())
			if err == nil && doesMatch {
				if ferr := op(path, f); ferr != nil {
					return ferr
				}
			}
		}
		return nil
	})
	return err
}

// ----------------------------------------------------------------- //
// Private Methods
// ----------------------------------------------------------------- //

// quick validity test
func isValidSubdir(subdir string) bool {
	for _, validSubdir := range ValidSubdirs {
		if subdir == validSubdir {
			return true
		}
	}
	return false
}

// Returns the full path of a subdir.
func (r *Repository) subdirPath(subdir string) string {
	if !isValidSubdir(subdir) {
		panic(fmt.Sprintf("bad subdir: %s", subdir))
	}
	return filepath.Join(r.basedir, subdir)
}

// Returns the full path of a resource id, given the subdir.
func (r *Repository) resourcePath(subdir, resourceId string) string {
	return filepath.Join(r.subdirPath(subdir), resourceId)
}
