package fs

import (
	"bytes"
	"sort"
	"testing"
)

func TestFileSystem_Move(t *testing.T) {
	// Setup
	fs, err := createTestFS()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		src string
		dst string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Relative", args{"bar/file1", "file3"}, false},
		{"Absolute", args{"/bar/file2", "file4"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := fs.Move(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("FileSystem.Move() error = %v, wantErr %v", err, tt.wantErr)
			}
			if _, err := fs.Read(tt.args.dst, bytes.NewBuffer(nil)); err != nil {
				t.Errorf("FileSystem.Read() error = %v, wantErr %v", err, nil)
			}
		})
	}
}

func createTestFS() (*FileSystem, error) {
	// Create a known file-system with a certain structure
	fs := New()
	dirs := []string{"foo/", "/bar/"}
	for _, dir := range dirs {
		if err := fs.MakeDir(dir); err != nil {
			return nil, err
		}
	}

	if err := fs.ChangeDir("bar/"); err != nil {
		return nil, err
	}

	subDirs := []string{"foo/", "foo2/"}
	for _, dir := range subDirs {
		if err := fs.MakeDir(dir); err != nil {
			return nil, err
		}
	}

	// Let's create some files.
	files := []string{"file1", "file2"}
	for _, file := range files {
		if err := fs.NewFile(file); err != nil {
			return nil, err
		}
	}

	// Let's write some content
	if _, err := fs.Write(files[0], bytes.NewBufferString("foobar")); err != nil {
		return nil, err
	}

	if err := fs.ChangeDir("/"); err != nil {
		return nil, err
	}

	return fs, nil

}

func TestFileSystem_ListDir(t *testing.T) {
	// Setup
	fs, err := createTestFS()
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		src string
	}
	tests := []struct {
		name          string
		args          args
		expectedDirs  []string
		expectedFiles []string
	}{
		{"Relative", args{"bar"}, []string{"foo", "foo2"}, []string{"file1", "file2"}},
		{"Abs1", args{"/foo/"}, []string{}, []string{}},
		{"Abs2", args{"/"}, []string{"bar", "foo"}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, dirs, err := fs.ListDir(tt.args.src)
			if err != nil {
				t.Error(err)
			}
			fileNames := make([]string, 0)
			for _, file := range files {
				fileNames = append(fileNames, file.md.Name())
			}
			dirNames := make([]string, 0)
			for _, dir := range dirs {
				dirNames = append(dirNames, dir.md.Name())
			}
			sort.Strings(fileNames)
			sort.Strings(dirNames)
			if len(fileNames) != len(tt.expectedFiles) {
				t.Errorf("Expected %d filenames, got %v", len(fileNames), len(tt.expectedFiles))
			}
			if len(dirNames) != len(tt.expectedDirs) {
				t.Errorf("Expected %d dirs, got %v", len(dirNames), len(tt.expectedDirs))
			}
			for i, file := range tt.expectedFiles {
				if file != fileNames[i] {
					t.Errorf("Expected filenames to match: %v vs %v", file, fileNames[i])
				}
			}
			for i, dir := range tt.expectedDirs {
				if dir != dirNames[i] {
					t.Errorf("Expected dirs to match: %v vs %v", dir, dirNames[i])
				}
			}
		})
	}
}

func TestFileSystem_Find(t *testing.T) {
	// Setup
	fs, err := createTestFS()
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		src string
	}
	tests := []struct {
		name          string
		args          args
		expectedDirs  []string
		expectedFiles []string
	}{
		{"Relative", args{"bar"}, []string{"foo", "foo2"}, []string{"file1", "file2"}},
		{"Abs1", args{"/foo/"}, []string{}, []string{}},
		{"Abs2", args{"/"}, []string{"bar", "foo"}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, dirs, err := fs.ListDir(tt.args.src)
			if err != nil {
				t.Error(err)
			}
			fileNames := make([]string, 0)
			for _, file := range files {
				fileNames = append(fileNames, file.md.Name())
			}
			dirNames := make([]string, 0)
			for _, dir := range dirs {
				dirNames = append(dirNames, dir.md.Name())
			}
			sort.Strings(fileNames)
			sort.Strings(dirNames)
			if len(fileNames) != len(tt.expectedFiles) {
				t.Errorf("Expected %d filenames, got %v", len(fileNames), len(tt.expectedFiles))
			}
			if len(dirNames) != len(tt.expectedDirs) {
				t.Errorf("Expected %d dirs, got %v", len(dirNames), len(tt.expectedDirs))
			}
			for i, file := range tt.expectedFiles {
				if file != fileNames[i] {
					t.Errorf("Expected filenames to match: %v vs %v", file, fileNames[i])
				}
			}
			for i, dir := range tt.expectedDirs {
				if dir != dirNames[i] {
					t.Errorf("Expected dirs to match: %v vs %v", dir, dirNames[i])
				}
			}
		})
	}
}
