package server

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/basharal/filesystem/fs"
	"github.com/basharal/filesystem/proto/pb_filesystem"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Opts struct {
	Port        int
	StartPrefix string
	EndPrefix   string
}

type Server struct {
	pb_filesystem.UnimplementedFileSeverServer

	fs    *fs.FileSystem
	start string
	end   string
	port  int
}

func New(opts Opts) (*Server, error) {
	// We only support a single letter prefixes. Longer ones are a bit more complicated
	// since we need to do some prefix matching.
	if len(opts.StartPrefix) != 1 {
		return nil, fmt.Errorf("start prefix must have a single letter")
	}
	if len(opts.EndPrefix) != 1 {
		return nil, fmt.Errorf("end prefix must have a single letter")
	}
	if opts.StartPrefix >= opts.EndPrefix {
		return nil, fmt.Errorf("end prefix must be lexicographically after start prefix")
	}
	return &Server{
		port:  opts.Port,
		start: opts.StartPrefix,
		end:   opts.EndPrefix,
		fs:    fs.New(),
	}, nil
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	pb_filesystem.RegisterFileSeverServer(grpcServer, s)
	go func() {
		<-ctx.Done()
		fmt.Printf("Starting graceful stop for gRPC server.")
		grpcServer.GracefulStop()
		fmt.Printf("Finished graceful stop for gRPC server.")
	}()
	fmt.Printf("Starting gRPC serving at %v\n.", l.Addr())
	grpcServer.Serve(l)
	return nil
}

// validatePath validates that the path belongs to this server.
func (s *Server) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	if !fs.IsAbs(path) {
		return fmt.Errorf("paths must be absolute")
	}

	// Path is absolute.
	if len(path) > 1 {
		// Skip '/'
		if path[1] < s.start[0] || path[1] >= s.end[0] {
			return fmt.Errorf("path isn't intended for server")
		}
	}
	return nil
}

// Returns the list of files/dirs at path.
func (s *Server) ListDir(ctx context.Context, in *pb_filesystem.Path) (*pb_filesystem.ListResponse, error) {
	glog.V(1).Infof("Start ListDir %s\n", in.Path)
	defer glog.V(1).Infof("End ListDir %s\n", in.Path)

	if err := s.validatePath(in.Path); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid path (%s). %s", in.Path, err)
	}
	files, dirs, err := s.fs.ListDir(in.Path)
	if err != nil {
		return nil, err
	}
	res := &pb_filesystem.ListResponse{}
	for _, file := range files {
		res.Files = append(res.Files, &pb_filesystem.File{Name: file.String(), Size: file.Size(), Path: file.Path()})
	}
	for _, dir := range dirs {
		res.Dirs = append(res.Dirs, &pb_filesystem.Dir{Name: dir.String(), Path: dir.Path()})
	}
	return res, nil
}
func (s *Server) MakeDir(ctx context.Context, in *pb_filesystem.Path) (*pb_filesystem.StatusResponse, error) {
	glog.V(1).Infof("Start MakeDir %s\n", in.Path)
	defer glog.V(1).Infof("End MakeDir %s\n", in.Path)
	if err := s.validatePath(in.Path); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid path (%s). %s", in.Path, err)
	}
	if err := s.fs.MakeDir(in.Path); err != nil {
		return nil, err
	}
	return &pb_filesystem.StatusResponse{Status: pb_filesystem.Status_SUCCESS}, nil
}
func (s *Server) Remove(ctx context.Context, in *pb_filesystem.Path) (*pb_filesystem.StatusResponse, error) {
	glog.V(1).Infof("Start Remove %s\n", in.Path)
	defer glog.V(1).Infof("End Remove %s\n", in.Path)
	if err := s.validatePath(in.Path); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid path (%s). %s", in.Path, err)
	}
	if err := s.fs.Remove(in.Path); err != nil {
		return nil, err
	}
	return &pb_filesystem.StatusResponse{Status: pb_filesystem.Status_SUCCESS}, nil
}
func (s *Server) CreateFile(ctx context.Context, in *pb_filesystem.Path) (*pb_filesystem.StatusResponse, error) {
	glog.V(1).Infof("Start CreateFile %s\n", in.Path)
	defer glog.V(1).Infof("End CreateFile %s\n", in.Path)
	if err := s.validatePath(in.Path); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid path (%s). %s", in.Path, err)
	}
	if err := s.fs.NewFile(in.Path); err != nil {
		return nil, err
	}
	return &pb_filesystem.StatusResponse{Status: pb_filesystem.Status_SUCCESS}, nil
}

func (s *Server) ReadFile(in *pb_filesystem.Path, stream pb_filesystem.FileSever_ReadFileServer) error {
	glog.V(1).Infof("Start ReadFile %s\n", in.Path)
	defer glog.V(1).Infof("End ReadFile %s\n", in.Path)
	if err := s.validatePath(in.Path); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid path (%s). %s", in.Path, err)
	}

	writer := streamWriter{stream: stream}
	if _, err := s.fs.Read(in.Path, writer); err != nil {
		return err
	}

	return nil
}
func (s *Server) WriteFile(stream pb_filesystem.FileSever_WriteFileServer) error {
	glog.V(1).Infof("Start WriteFile\n")
	defer glog.V(1).Infof("End WriteFile\n")
	in, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}

	// First message must be the full path. Others are the bytes
	if in.GetPath() == "" {
		return fmt.Errorf("first message must be the path of the file to write to")
	}
	reader := streamReader{stream: stream}
	if _, err := s.fs.Write(in.GetPath(), reader); err != nil {
		return err
	}

	return stream.SendAndClose(&pb_filesystem.StatusResponse{Status: pb_filesystem.Status_SUCCESS})
}

type streamWriter struct {
	stream pb_filesystem.FileSever_ReadFileServer
}

func (sw streamWriter) Write(p []byte) (int, error) {
	if err := sw.stream.Send(&pb_filesystem.Payload{Data: p}); err != nil {
		return 0, err
	}
	return len(p), nil
}

type streamReader struct {
	stream pb_filesystem.FileSever_WriteFileServer

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
