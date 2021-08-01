package fs

import (
	"bytes"
	"io"
	"sync"
)

// File is an abstraction of a file.
type File struct {
	md *Metadata

	// mu protects below
	mu      sync.RWMutex
	content []byte
}

func newFile(fs *FileSystem) *File {
	return &File{
		md:      newMetadata(fs, fileType),
		content: make([]byte, 0),
	}
}

// Write appends to the file's content as a stream until io.EOF is encountered and returns the
// number of bytes written.
func (f *File) Write(reader io.Reader) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	buf := bytes.NewBuffer(f.content)
	return buf.ReadFrom(reader)
}

// Read reads the file content as a stream and returns the number of bytes read.
func (f *File) Read(writer io.Writer) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	buf := bytes.NewBuffer(f.content)
	return buf.WriteTo(writer)
}

// ReadAt reads at a particular offset of the file. Returns number of bytes read.
func (f *File) ReadAt(writer io.Writer, offset int) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if offset >= len(f.content) {
		return 0, io.EOF
	}
	buf := bytes.NewBuffer(f.content[offset:])
	return buf.WriteTo(writer)
}

func (f *File) Size() int64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return int64(len(f.content))
}
