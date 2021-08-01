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

func (d *Dir) String() string {
	return d.md.Name()
}

// Path returns absolute path to the dir.
func (d *Dir) Path() string {
	return d.md.AbsolutePath()
}
