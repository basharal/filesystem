package main

import (
	"context"
	"flag"

	"github.com/basharal/filesystem/server"
	"github.com/golang/glog"
)

var (
	port  = flag.Int("port", 0, "port to listen on")
	start = flag.String("start_prefix", "", "start prefix for file-paths for server (inclusive)")
	end   = flag.String("end_prefix", "", "end prefix for file-paths for server (exclusive")
)

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := server.New(server.Opts{
		StartPrefix: *start,
		EndPrefix:   *end,
		Port:        *port,
	})
	if err != nil {
		glog.Fatal(err)
	}
	s.ListenAndServe(ctx)
}
