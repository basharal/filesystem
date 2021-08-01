package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/basharal/filesystem/fs"
	"github.com/fatih/color"
)

var (
	flagHelp = flag.Bool("help", false, "print usage")
)

func processCommands(ctx context.Context, fs *fs.FileSystem, cmd commands) {
	fmt.Println("Please enter filesystem command.")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			reader := bufio.NewReader(os.Stdin)
			line, err := reader.ReadString('\n')
			if err != nil {
				color.Red(err.Error())
				continue
			}
			if err := cmd.Handle(line); err != nil {
				color.Red(err.Error())
			}
		}
	}
}

func main() {
	flag.Parse()
	fs := fs.New()
	cmds := newCommands(fs)

	if *flagHelp {
		supported := cmds.Supported()
		for k, v := range supported {
			fmt.Printf("%s - %s\n", k, v)
		}
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processCommands(ctx, fs, cmds)
}
