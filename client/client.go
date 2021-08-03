package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/basharal/filesystem/fs"
	"github.com/basharal/filesystem/proto/pb_filesystem"
	"google.golang.org/grpc"
)

// Server represents a file-server
type Server struct {
	// StartPrefix is the prefix for first possible path on the server (inclusive)
	StartPrefix string `json:"start_prefix"`

	// EndPrefix is the prefix for last possible path on the server (exclusive)
	EndPrefix string `json:"end_prefix"`

	// Addr is the ip:port (or host:port) for the server to accept gRPC requests.
	Addr string `json:"addr_prefix"`
}

type Opts struct {
	Servers []Server
}

type Client struct {
	servers []Server

	mu      sync.RWMutex
	clients map[string]pb_filesystem.FileSeverClient
	conns   map[string]*grpc.ClientConn
}

func New(opts Opts) (*Client, error) {
	// TODO: validate prefixes and stuff
	return &Client{servers: opts.Servers}, nil
}

// Dial connects to all server. TODO: Make this lazy and also have it dial
// upon disconnects.
func (c *Client) Dial(ctx context.Context) error {
	// Dial all servers. TODO: Not do that and do it lazily.
	conns := make(map[string]*grpc.ClientConn)
	clients := make(map[string]pb_filesystem.FileSeverClient)
	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	for _, server := range c.servers {
		conn, err := grpc.DialContext(ctx, server.Addr, grpc.WithInsecure())
		if err != nil {
			return err
		}
		conns[server.Addr] = conn
		clients[server.Addr] = pb_filesystem.NewFileSeverClient(conn)

	}

	// Don't cleanup
	c.mu.Lock()
	c.conns = conns
	c.clients = clients
	c.mu.Unlock()
	conns = nil
	return nil
}

func (c *Client) clientsForPath(path string) ([]pb_filesystem.FileSeverClient, error) {
	// TODO: optimize this. We should do some sort of binary search/b-tree
	servers := make([]string, 0)
	for _, server := range c.servers {
		if !fs.IsAbs(path) {
			return nil, fmt.Errorf("path must be absolute")
		}
		// TODO: support longer prefixes
		if path == fs.SeperatorStr || path[1] >= server.StartPrefix[0] && path[1] < server.EndPrefix[0] {
			servers = append(servers, server.Addr)
		}
	}
	clients := make([]pb_filesystem.FileSeverClient, 0, len(servers))
	c.mu.RLock()
	for _, addr := range servers {
		clients = append(clients, c.clients[addr])
	}
	c.mu.RUnlock()
	return clients, nil
}

func (c *Client) ListDir(ctx context.Context, path string) ([]*pb_filesystem.File, []*pb_filesystem.Dir, error) {
	clients, err := c.clientsForPath(path)
	if err != nil {
		return nil, nil, err
	}

	// guarantee that the channels won't block.
	// TODO: optimize this and support cancelation upon first error.
	filesCh := make(chan []*pb_filesystem.File, len(clients))
	dirsCh := make(chan []*pb_filesystem.Dir, len(clients))
	errCh := make(chan error, len(clients))
	combinedFiles := make([]*pb_filesystem.File, 0)
	combinedDirs := make([]*pb_filesystem.Dir, 0)
	var wg sync.WaitGroup
	for _, client := range clients {
		client := client
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := client.ListDir(ctx, &pb_filesystem.Path{Path: path})
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			filesCh <- out.Files
			dirsCh <- out.Dirs
		}()
	}

	wg.Wait()
	// We guaranteed that channels won't block
	close(errCh)
	close(filesCh)
	close(dirsCh)
	for err := range errCh {
		if err != nil {
			return nil, nil, err
		}
	}
	for files := range filesCh {
		combinedFiles = append(combinedFiles, files...)
	}
	for dirs := range dirsCh {
		combinedDirs = append(combinedDirs, dirs...)
	}

	return combinedFiles, combinedDirs, nil
}
func (c *Client) MakeDir(ctx context.Context, path string) error {
	clients, err := c.clientsForPath(path)
	if err != nil {
		return err
	}

	// We must have a single server.
	if len(clients) != 1 {
		return fmt.Errorf("must have a single server per path")
	}

	if _, err := clients[0].MakeDir(ctx, &pb_filesystem.Path{Path: path}); err != nil {
		return err
	}
	return nil
}
func (c *Client) Remove(ctx context.Context, path string) error {
	clients, err := c.clientsForPath(path)
	if err != nil {
		return err
	}

	// We must have a single server.
	if len(clients) != 1 {
		return fmt.Errorf("must have a single server per path")
	}

	if _, err := clients[0].Remove(ctx, &pb_filesystem.Path{Path: path}); err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateFile(ctx context.Context, path string) error {
	clients, err := c.clientsForPath(path)
	if err != nil {
		return err
	}

	// We must have a single server.
	if len(clients) != 1 {
		return fmt.Errorf("must have a single server per path")
	}

	if _, err := clients[0].CreateFile(ctx, &pb_filesystem.Path{Path: path}); err != nil {
		return err
	}
	return nil
}

func (c *Client) ReadFile(ctx context.Context, local, remote string) error {
	clients, err := c.clientsForPath(remote)
	if err != nil {
		return err
	}

	f, err := os.Create(local)
	if err != nil {
		return err
	}
	defer f.Close()

	// We must have a single server.
	if len(clients) != 1 {
		return fmt.Errorf("must have a single server per path")
	}

	client, err := clients[0].ReadFile(ctx, &pb_filesystem.Path{Path: remote})
	if err != nil {
		return err
	}

	reader := streamReader{stream: client}
	if _, err := io.Copy(f, reader); err != nil {
		return err
	}
	return nil
}
func (c *Client) WriteFile(ctx context.Context, local, remote string) error {
	clients, err := c.clientsForPath(remote)
	if err != nil {
		return err
	}

	// We must have a single server.
	if len(clients) != 1 {
		return fmt.Errorf("must have a single server per path")
	}

	f, err := os.Open(local)
	if err != nil {
		return err
	}
	defer f.Close()

	client, err := clients[0].WriteFile(ctx)
	if err != nil {
		return err
	}

	// Send the first message with the path
	req := &pb_filesystem.FilePayload{Input: &pb_filesystem.FilePayload_Path{Path: remote}}
	if err := client.Send(req); err != nil {
		client.CloseSend()
		return err
	}

	writer := streamWriter{stream: client}
	if _, err := io.Copy(writer, f); err != nil {
		return err
	}

	// Done.
	if _, err := client.CloseAndRecv(); err != nil {
		return err
	}

	return nil
}

type streamWriter struct {
	stream pb_filesystem.FileSever_WriteFileClient
}

func (sw streamWriter) Write(p []byte) (int, error) {
	payload := &pb_filesystem.FilePayload{Input: &pb_filesystem.FilePayload_Data{Data: p}}
	if err := sw.stream.Send(payload); err != nil {
		return 0, err
	}
	return len(p), nil
}

type streamReader struct {
	stream pb_filesystem.FileSever_ReadFileClient

	buf []byte
}

func (sw streamReader) Read(p []byte) (int, error) {
	if len(sw.buf) > 0 {
		return sw.read(p), nil
	}
	pb, err := sw.stream.Recv()
	if err != nil {
		return 0, err
	}
	sw.buf = pb.GetData()
	return sw.read(p), nil
}

func (sw streamReader) read(p []byte) int {
	n := copy(p, sw.buf)
	sw.buf = sw.buf[n:]
	return n
}
