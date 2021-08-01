package fs

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/basharal/trie"
)

const (
	Separator    = '/'
	SeperatorStr = string(Separator)
)

var (
	ErrAlreadyExist = fmt.Errorf("already exists")
	ErrNotFound     = fmt.Errorf("not found")
	ErrInvalidName  = fmt.Errorf("invalid name")
	ErrNotSupported = fmt.Errorf("not supported")
	ErrDirNotEmpty  = fmt.Errorf("directory not empty")
)

// FileSystem is a thread-safe in-memory filesystem that allows basic operations. All public methods
// are thread-safe.
type FileSystem struct {
	// trie is thread-safe and provides internal datastructure for
	// the filesystem metadata.
	trie *trie.Trie

	// mu protects below.
	mu         sync.RWMutex
	currentDir *Dir
	root       *Dir
}

// New returns a new filesystem.
func New() *FileSystem {
	t := trie.New()
	fs := &FileSystem{
		trie: t,
	}

	root := newDir(fs)
	node := t.Add("/", root)
	root.md.setNode(node)

	return &FileSystem{
		trie:       t,
		root:       root,
		currentDir: root,
	}
}

// CurrentDir returns the absolute path of the current directory
func (fs *FileSystem) CurrentDir() string {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.currentDir.md.AbsolutePath()
}

// ChangeDir switches current directory to s (relative/absolute)
func (fs *FileSystem) ChangeDir(s string) error {
	s = fs.normalizeDirPath(s)
	fs.mu.Lock()
	defer fs.mu.Unlock()
	node := fs.findNode(s)
	if node == nil {
		return ErrNotFound
	}
	dir, ok := node.Meta().(*Dir)
	if !ok {
		return fmt.Errorf("directory expected. file given")
	}
	fs.currentDir = dir
	return nil
}

// MakeDir makes a new directory relative or absolute.
func (fs *FileSystem) MakeDir(s string) error {
	s = fs.normalizeDirPath(s)
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.isAbs(s) {
		return fs.mkdirAtNode(s[1:], fs.root.md.node)
	}
	return fs.mkdirAtNode(s, fs.currentDir.md.node)
}

// Remove removes s (relative/absolute) from the filesystem. It could be dir/file.
func (fs *FileSystem) Remove(s string) error {
	// s maybe a dir/file.
	s = fs.normalizePath(s)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Check if it's a file
	node := fs.findNode(s)
	if node == nil {
		// See if appending a '/' helps
		s = fs.normalizeDirPath(s)
		node = fs.findNode(s)
	}
	if node == nil {
		return ErrNotFound
	}

	// Make sure we're not removing the current directory or root. Otherwise, we need to deal
	// with changing directories
	if node == fs.root.md.node || node == fs.currentDir.md.node {
		return ErrNotSupported
	}

	_, ok := node.Meta().(*File)
	if ok {
		// Just a file. We can remove it
		fs.trie.Remove(s)
		return nil
	}

	// We have a directory. We can only remove it after all its content is gone.
	// It's a bit more complicated to do it Because we need to do a reverse topological sort.
	// TODO.
	files, dirs, err := fs.ListDir(s)
	if err != nil {
		return err
	}
	if len(files) != 0 || len(dirs) != 0 {
		return ErrDirNotEmpty
	}

	fs.trie.Remove(s)
	return nil
}

// FindFirstRegex returns the first absolute path matching the regex for the given path (absolute/
// relative)
func (fs *FileSystem) FindFirstRegex(regex, path string) (string, error) {
	// s maybe a dir/file.
	path = fs.normalizePath(path)

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	node := fs.findNode(path)
	if node == nil {
		// See if it's a file
		path = fs.normalizeDirPath(path)
	}
	node = fs.findNode(path)
	if node == nil {
		return "", ErrNotFound
	}

	path, _, err := fs.trie.FirstRegexMatchAtNode(regex, node)
	if err != nil {
		return "", err
	}
	return path, err
}

// ListDir lists all the files/dirs in s (relative/abs)
func (fs *FileSystem) ListDir(s string) ([]*File, []*Dir, error) {
	s = fs.normalizeDirPath(s)
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	node := fs.findNode(s)
	if node == nil {
		return nil, nil, ErrNotFound
	}

	_, nodes, err := fs.trie.ListAtNode(s, node)
	if err != nil {
		return nil, nil, err
	}

	files, dirs := convertNodes(nodes)
	return files, dirs, nil
}

// NewFile creates a new empty file at s (relative/absolute).
func (fs *FileSystem) NewFile(s string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.isAbs(s) {
		return fs.newFileAtNode(s[1:], fs.root.md.node)
	}
	return fs.newFileAtNode(s, fs.currentDir.md.node)
}

// Write writes the what's in reader until EOF to the file s (relative/abs).
func (fs *FileSystem) Write(s string, reader io.Reader) (int64, error) {
	fs.mu.RLock()
	node := fs.findNode(s)
	fs.mu.RUnlock()
	if node == nil {
		return -1, ErrNotFound
	}
	file, ok := node.Meta().(*File)
	if !ok {
		return -1, fmt.Errorf("cannot write content on directories")
	}
	return file.Write(reader)
}

// Read reads the file at s (relative/abs) and streams its content to writer.
func (fs *FileSystem) Read(s string, writer io.Writer) (int64, error) {
	fs.mu.RLock()
	node := fs.findNode(s)
	fs.mu.RUnlock()
	if node == nil {
		return -1, ErrNotFound
	}
	file, ok := node.Meta().(*File)
	if !ok {
		return -1, fmt.Errorf("cannot read content on directories")
	}
	return file.Read(writer)
}

// Move moves a file/dir from src to dst. src/dst are relative or absolute.
func (fs *FileSystem) Move(src, dst string) error {
	if err := validateName(src); err != nil {
		return ErrInvalidName
	}

	if err := validateName(dst); err != nil {
		return ErrInvalidName
	}
	absSrc := fs.normalizePath(src)
	absDst := fs.normalizePath(dst)

	fs.mu.Lock()
	defer fs.mu.Unlock()
	srcNode := fs.findNode(src)
	if srcNode == nil {
		return ErrNotFound
	}

	dstNode := fs.findNode(dst)
	if dstNode != nil {
		// Don't support overwrites
		return ErrAlreadyExist
	}

	fs.trie.Remove(absSrc)
	fs.trie.Add(absDst, srcNode.Meta())
	return nil
}

// Find returns the list of files/dirs that match search given the path (relative/abs)
func (fs *FileSystem) Find(path, search string) ([]*File, []*Dir, error) {
	path = fs.normalizeDirPath(path)
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	node := fs.findNode(path)
	if node == nil {
		return nil, nil, ErrNotFound
	}
	_, nodes, err := fs.trie.ExactSearchAtNode(search, node)
	if err != nil {
		return nil, nil, err
	}

	files, dirs := convertNodes(nodes)
	return files, dirs, nil
}

func (fs *FileSystem) isAbs(s string) bool {
	return s != "" && s[0] == Separator
}

// makes a directory relative to n with relative path
func (fs *FileSystem) mkdirAtNode(path string, n *trie.Node) error {
	if path == "" || !strings.HasSuffix(path, SeperatorStr) {
		return ErrInvalidName
	}

	if err := validateName(path); err != nil {
		return err
	}

	// TODO: Support creating subdirectories. For now, we only support one
	splitted := strings.Split(path, SeperatorStr)
	if len(splitted) != 2 {
		return ErrNotSupported
	}

	// Check if we already have a dir with this name
	if _, ok := fs.trie.FindAtNode(path, n); ok {
		return ErrAlreadyExist
	}
	// Try for a file
	if _, ok := fs.trie.FindAtNode(path[:len(path)-1], n); ok {
		return ErrAlreadyExist
	}

	dir := newDir(fs)
	added := fs.trie.AddAtNode(path, n, dir)
	dir.md.setNode(added)
	return nil
}

func (fs *FileSystem) findNode(path string) *trie.Node {
	node := fs.currentDir.md.node
	if fs.isAbs(path) {
		node = fs.trie.Root()
	}
	node, _ = fs.trie.FindAtNode(path, node)
	return node
}

func (fs *FileSystem) normalizeDirPath(path string) string {
	// Dirs always end with a '/'
	if strings.HasSuffix(path, SeperatorStr) {
		return path
	}
	return path + SeperatorStr
}

func (fs *FileSystem) normalizePath(path string) string {
	// TODO: support . and ..
	if fs.isAbs(path) {
		return path
	}
	return fs.currentDir.md.AbsolutePath() + path
}

// creates a new file at n with relative path
func (fs *FileSystem) newFileAtNode(path string, n *trie.Node) error {
	if path == "" || strings.HasSuffix(path, SeperatorStr) {
		return ErrInvalidName
	}

	if err := validateName(path); err != nil {
		return err
	}

	// TODO: Support creating subdirectories. For now, files must be created
	// in the same directory as n
	splitted := strings.Split(path, SeperatorStr)
	if len(splitted) != 1 {
		return ErrNotSupported
	}

	// Check if we already have a file with this name
	if _, ok := fs.trie.FindAtNode(path, n); ok {
		return ErrAlreadyExist
	}
	// Try for a directory
	if _, ok := fs.trie.FindAtNode(path+SeperatorStr, n); ok {
		return ErrAlreadyExist
	}

	file := newFile(fs)
	added := fs.trie.AddAtNode(path, n, file)
	file.md.setNode(added)
	return nil
}

func convertNodes(nodes []*trie.Node) ([]*File, []*Dir) {
	dirs := make([]*Dir, 0)
	files := make([]*File, 0)
	for _, n := range nodes {
		meta := n.Meta()
		if dir, ok := meta.(*Dir); ok {
			dirs = append(dirs, dir)
			continue
		}
		file := meta.(*File)
		files = append(files, file)
	}

	return files, dirs
}
