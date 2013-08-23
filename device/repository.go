//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

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
	baseDir string
}

// Create a new Repository.
func NewRepository(baseDir string) *Repository {
	return &Repository{baseDir}
}

// subdirectories
const (
	SubdirLocal = "local"
	SubdirCloud = "cloud"
)

// subdirectory search path
var subdirSearchPath = []string{SubdirLocal, SubdirCloud}

// Return the base directory of the repository.
func (r *Repository) BaseDir() string {
	return r.baseDir
}

func (r *Repository) NewResourceId() (resourceId, resourcePath string) {
	return
}
