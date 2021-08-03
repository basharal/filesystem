package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/basharal/filesystem/client"
	"github.com/fatih/color"
	"github.com/golang/glog"
)

var (
	flagConf = flag.String("config", "config.json", "path to json file with config")
	flagHelp = flag.Bool("help", false, "print usage")
)

func processCommands(ctx context.Context, cmd commands) {
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
			if err := cmd.Handle(ctx, line); err != nil {
				color.Red(err.Error())
			}
		}
	}
}

func main() {
	flag.Parse()
	conf, err := Parse(*flagConf)
	if err != nil {
		glog.Fatal(err)
	}

	c, err := client.New(client.Opts{Servers: conf.Servers})
	if err != nil {
		glog.Fatal(err)
	}
	cmds := newCommands(c)
	if *flagHelp {
		supported := cmds.Supported()
		for k, v := range supported {
			fmt.Printf("%s - %s\n", k, v)
		}
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := c.Dial(ctx); err != nil {
		glog.Fatal(err)
	}

	processCommands(ctx, cmds)
}
