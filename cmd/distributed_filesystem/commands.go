package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/basharal/filesystem/client"
	"github.com/basharal/filesystem/proto/pb_filesystem"
	"github.com/fatih/color"
)

type handlerFunc func(ctx context.Context, args []string) error

type cmdHandler struct {
	usage   string
	handler handlerFunc
}

type commands struct {
	fs        *client.Client
	supported map[string]cmdHandler
}

func newCommands(client *client.Client) commands {
	c := commands{
		fs: client,
	}
	supported := map[string]cmdHandler{
		"add":   {"add creates an empty file (i.e., add /foo)", c.add},
		"ls":    {"lists directory content at path (or current dir)", c.ls},
		"mkdir": {"creates a new directory (i.e., mkdir foo)", c.mkDir},
		"read": {"reads from in-memory filesystem into local filesystem. " +
			"will truncate the local file (i.e., read /bar /tmp/bar", c.read},
		"rm": {"removes a file/directory(if empty) (i.e., rm foo)", c.rm},
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

func (c commands) mkDir(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("wrong arguments")
	}
	return c.fs.MakeDir(ctx, args[0])
}

func (c commands) rm(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("wrong arguments")
	}
	return c.fs.Remove(ctx, args[0])
}

func (c commands) add(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("wrong arguments")
	}
	return c.fs.CreateFile(ctx, args[0])
}

func (c commands) printFilesAndDirs(files []*pb_filesystem.File, dirs []*pb_filesystem.Dir, fullPath bool) {
	// TODO: Sort by name.
	for _, f := range files {
		fmt.Printf("%d\t%s\n", f.Size, f.Name)
	}
	for _, d := range dirs {
		color.Cyan("\t%s\n", d.Name)
	}
}

func (c commands) ls(ctx context.Context, args []string) error {
	if len(args) != 1 && len(args) != 0 {
		return fmt.Errorf("wrong arguments")
	}
	if len(args) == 0 {
		args = []string{""}
	}
	files, dirs, err := c.fs.ListDir(ctx, args[0])
	if err != nil {
		return err
	}

	c.printFilesAndDirs(files, dirs, false)
	return nil
}

func (c commands) read(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong arguments")
	}

	if err := c.fs.ReadFile(ctx, args[1], args[0]); err != nil {
		return err
	}

	return nil
}

func (c commands) write(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong arguments")
	}

	if err := c.fs.WriteFile(ctx, args[0], args[1]); err != nil {
		return err
	}

	return nil
}

func (c commands) Handle(ctx context.Context, line string) error {
	cmd, args, err := c.parse(line)
	if err != nil {
		return err
	}
	found, ok := c.supported[cmd]
	if !ok {
		return fmt.Errorf("unknown command %s", cmd)
	}
	return found.handler(ctx, args)
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
