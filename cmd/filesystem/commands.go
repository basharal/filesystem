package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/basharal/filesystem/fs"
	"github.com/fatih/color"
)

type handlerFunc func(args []string) error

type cmdHandler struct {
	usage   string
	handler handlerFunc
}

type commands struct {
	fs        *fs.FileSystem
	supported map[string]cmdHandler
}

func newCommands(fs *fs.FileSystem) commands {
	c := commands{
		fs: fs,
	}
	supported := map[string]cmdHandler{
		"add":   {"add creates an empty file (i.e., add /foo)", c.add},
		"cd":    {"changes current directory (i.e., cd /foo)", c.chDir},
		"find":  {"finds all files/dirs matching string at path (i.e., find /foo hello)", c.find},
		"ls":    {"lists directory content at path (or current dir)", c.ls},
		"mkdir": {"creates a new directory (i.e., mkdir foo)", c.mkDir},
		"mv":    {"mv moves a file from a to b (i.e., mv foo.txt /bar.txt", c.mv},
		"pwd":   {"prints current path", c.pwd},
		"read": {"reads from in-memory filesystem into local filesystem. " +
			"will truncate the local file (i.e., read /bar /tmp/bar", c.read},
		"regex": {"returns path to first regex match at path (i.e., regex /bar .*foo", c.regex},
		"rm":    {"removes a file/directory(if empty) (i.e., rm foo)", c.rm},
		"write": {"reads from local filesystem and writes into in-memory filesystem. " +
			"will append (i.e., write /tmp/bar /bar", c.write},
	}
	c.supported = supported
	return c
}

func (c commands) Supported() map[string]string {
	s := make(map[string]string, len(c.supported))
	for k, v := range c.supported {
		s[k] = v.usage
	}

	return s
}

func (c commands) mkDir(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("wrong arguments")
	}
	return c.fs.MakeDir(args[0])
}

func (c commands) chDir(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("wrong arguments")
	}
	return c.fs.ChangeDir(args[0])
}

func (c commands) rm(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("wrong arguments")
	}
	return c.fs.Remove(args[0])
}

func (c commands) mv(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong arguments")
	}
	return c.fs.Move(args[0], args[1])
}

func (c commands) add(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("wrong arguments")
	}
	return c.fs.NewFile(args[0])
}

func (c commands) find(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong arguments")
	}
	files, dirs, err := c.fs.Find(args[0], args[1])
	if err != nil {
		return err
	}

	c.printFilesAndDirs(files, dirs, true)
	return nil
}

func (c commands) regex(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong arguments")
	}
	found, err := c.fs.FindFirstRegex(args[0], args[1])
	if err != nil {
		return err
	}

	fmt.Println(found)
	return nil
}

func (c commands) pwd(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("wrong arguments")
	}
	dir := c.fs.CurrentDir()
	fmt.Println(dir)
	return nil
}

func (c commands) printFilesAndDirs(files []*fs.File, dirs []*fs.Dir, fullPath bool) {
	// TODO: Sort by name.
	for _, f := range files {
		s := f.String()
		if fullPath {
			s = f.Path()
		}
		fmt.Printf("%d\t%s\n", f.Size(), s)
	}
	for _, d := range dirs {
		s := d.String()
		if fullPath {
			s = d.Path()
		}
		color.Cyan("\t%s\n", s)
	}
}

func (c commands) ls(args []string) error {
	if len(args) != 1 && len(args) != 0 {
		return fmt.Errorf("wrong arguments")
	}
	if len(args) == 0 {
		args = []string{""}
	}
	files, dirs, err := c.fs.ListDir(args[0])
	if err != nil {
		return err
	}

	c.printFilesAndDirs(files, dirs, false)
	return nil
}

func (c commands) read(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong arguments")
	}

	f, err := os.Create(args[1])
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := c.fs.Read(args[0], f); err != nil {
		return err
	}

	return nil
}

func (c commands) write(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong arguments")
	}

	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := c.fs.Write(args[1], f); err != nil {
		return err
	}

	return nil
}

func (c commands) Handle(line string) error {
	cmd, args, err := c.parse(line)
	if err != nil {
		return err
	}
	found, ok := c.supported[cmd]
	if !ok {
		return fmt.Errorf("unknown command %s", cmd)
	}
	return found.handler(args)
}

func (c commands) parse(line string) (string, []string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", nil, fmt.Errorf("empty command")
	}
	// Space must be escaped for arguments. The filesystem supports space, but parsing doesn't.
	splitted := strings.Split(line, " ")
	if len(splitted) > 1 {
		return splitted[0], splitted[1:], nil
	}
	return splitted[0], []string{}, nil
}

type item struct {
	name  string
	isDir bool
}

type itemSlice []item

func (x itemSlice) Len() int           { return len(x) }
func (x itemSlice) Less(i, j int) bool { return x[i].name < x[j].name }
func (x itemSlice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
