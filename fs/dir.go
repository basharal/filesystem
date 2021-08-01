package fs

// Dir is an abstraction of a directory
type Dir struct {
	// md is immutable.
	md *Metadata
}

func newDir(fs *FileSystem) *Dir {
	return &Dir{
		md: newMetadata(fs, dirType),
	}
}
