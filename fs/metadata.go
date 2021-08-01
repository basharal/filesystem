package fs

import (
	"strings"

	"github.com/basharal/trie"
	"github.com/golang/glog"
)

// Type of the filesystem node
type NodeType int

var (
	fileType NodeType = 1
	dirType  NodeType = 2
)

// Metadata provides common metadata for files and directories.
type Metadata struct {
	fs *FileSystem
	nt NodeType

	// node is set later due to a chicken and egg problem with the trie node. node is immutable.
	node *trie.Node
}

func newMetadata(fs *FileSystem, nt NodeType) *Metadata {
	return &Metadata{
		nt: nt,
		fs: fs,
	}
}

// setNode must be called to set the node on the metadata. It's a chicken and egg problem as to
// why we can't do it as part of newMetadata.
func (md *Metadata) setNode(n *trie.Node) error {
	if md.node != nil {
		return ErrAlreadyExist
	}
	md.node = n
	return nil
}

func (md *Metadata) Node() *trie.Node {
	return md.node
}

// AbsolutePath return the absolute path of the dir/file. For dirs, we remove '/' except for the
// root.
func (md *Metadata) AbsolutePath() string {
	if md.node == nil {
		glog.Fatalln("Impossible. node is set at creation time.")
	}
	path := md.node.Path()
	if len(path) > 1 && strings.HasSuffix(path, SeperatorStr) {
		return strings.TrimSuffix(path, SeperatorStr)
	}
	return path
}

// Returns the name of the node. For dirs, we trim suffix '/' for dirs)
func (md *Metadata) Name() string {
	if md.node == nil {
		glog.Fatalln("Impossible. node is set at creation time.")
	}
	return md.node.Name()
}
