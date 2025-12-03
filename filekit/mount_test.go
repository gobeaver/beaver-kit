package filekit

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// mockFS is a simple mock filesystem for testing
type mockFS struct {
	name    string
	files   map[string][]byte
	dirs    map[string]bool
	copyErr error
	moveErr error
}

func newMockFS(name string) *mockFS {
	return &mockFS{
		name:  name,
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *mockFS) Write(ctx context.Context, path string, content io.Reader, options ...Option) error {
	data, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	m.files[path] = data
	return nil
}

func (m *mockFS) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockFS) ReadAll(ctx context.Context, path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return data, nil
}

func (m *mockFS) Delete(ctx context.Context, path string) error {
	if _, ok := m.files[path]; !ok {
		return errors.New("file not found")
	}
	delete(m.files, path)
	return nil
}

func (m *mockFS) FileExists(ctx context.Context, path string) (bool, error) {
	_, ok := m.files[path]
	return ok, nil
}

func (m *mockFS) DirExists(ctx context.Context, path string) (bool, error) {
	_, ok := m.dirs[path]
	return ok, nil
}

func (m *mockFS) Stat(ctx context.Context, path string) (*FileInfo, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return &FileInfo{
		Name:        path,
		Path:        path,
		Size:        int64(len(data)),
		ModTime:     time.Now(),
		IsDir:       false,
		ContentType: "text/plain",
	}, nil
}

func (m *mockFS) ListContents(ctx context.Context, path string, recursive bool) ([]FileInfo, error) {
	var files []FileInfo
	for filePath, data := range m.files {
		if path == "" || strings.HasPrefix(filePath, path) {
			files = append(files, FileInfo{
				Name:  filePath,
				Path:  filePath,
				Size:  int64(len(data)),
				IsDir: false,
			})
		}
	}
	return files, nil
}

func (m *mockFS) CreateDir(ctx context.Context, path string) error {
	m.dirs[path] = true
	return nil
}

func (m *mockFS) DeleteDir(ctx context.Context, path string) error {
	delete(m.dirs, path)
	return nil
}

// mockCopierFS implements the Copier interface
type mockCopierFS struct {
	*mockFS
	copyCalled bool
}

func newMockCopierFS(name string) *mockCopierFS {
	return &mockCopierFS{mockFS: newMockFS(name)}
}

func (m *mockCopierFS) Copy(ctx context.Context, src, dst string) error {
	m.copyCalled = true
	if m.copyErr != nil {
		return m.copyErr
	}
	data, ok := m.files[src]
	if !ok {
		return errors.New("source not found")
	}
	m.files[dst] = append([]byte{}, data...)
	return nil
}

// mockMoverFS implements the Mover interface
type mockMoverFS struct {
	*mockFS
	moveCalled bool
}

func newMockMoverFS(name string) *mockMoverFS {
	return &mockMoverFS{mockFS: newMockFS(name)}
}

func (m *mockMoverFS) Move(ctx context.Context, src, dst string) error {
	m.moveCalled = true
	if m.moveErr != nil {
		return m.moveErr
	}
	data, ok := m.files[src]
	if !ok {
		return errors.New("source not found")
	}
	m.files[dst] = data
	delete(m.files, src)
	return nil
}

func TestNewMountManager(t *testing.T) {
	mm := NewMountManager()
	if mm == nil {
		t.Fatal("expected non-nil MountManager")
	}
	if mm.mounts == nil {
		t.Fatal("expected non-nil mounts map")
	}
}

func TestMount(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		fs        FileSystem
		wantErr   error
		setupFunc func(*MountManager)
	}{
		{
			name:    "valid mount",
			path:    "/local",
			fs:      newMockFS("local"),
			wantErr: nil,
		},
		{
			name:    "mount without leading slash",
			path:    "cloud",
			fs:      newMockFS("cloud"),
			wantErr: nil,
		},
		{
			name:    "nil driver",
			path:    "/test",
			fs:      nil,
			wantErr: ErrNilDriver,
		},
		{
			name:    "empty path",
			path:    "",
			fs:      newMockFS("test"),
			wantErr: ErrEmptyMountPath,
		},
		{
			name:    "duplicate mount",
			path:    "/local",
			fs:      newMockFS("local2"),
			wantErr: ErrMountExists,
			setupFunc: func(mm *MountManager) {
				mm.Mount("/local", newMockFS("local"))
			},
		},
		{
			name:    "nested mount allowed",
			path:    "/cloud/archive",
			fs:      newMockFS("archive"),
			wantErr: nil,
			setupFunc: func(mm *MountManager) {
				mm.Mount("/cloud", newMockFS("cloud"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mm := NewMountManager()
			if tt.setupFunc != nil {
				tt.setupFunc(mm)
			}

			err := mm.Mount(tt.path, tt.fs)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestUnmount(t *testing.T) {
	mm := NewMountManager()
	mm.Mount("/local", newMockFS("local"))

	// Successful unmount
	err := mm.Unmount("/local")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Unmount non-existent
	err = mm.Unmount("/local")
	if !errors.Is(err, ErrMountNotFound) {
		t.Errorf("expected ErrMountNotFound, got %v", err)
	}
}

func TestMounts(t *testing.T) {
	mm := NewMountManager()
	local := newMockFS("local")
	cloud := newMockFS("cloud")

	mm.Mount("/local", local)
	mm.Mount("/cloud", cloud)

	mounts := mm.Mounts()
	if len(mounts) != 2 {
		t.Errorf("expected 2 mounts, got %d", len(mounts))
	}
	if mounts["/local"] != local {
		t.Error("expected /local mount")
	}
	if mounts["/cloud"] != cloud {
		t.Error("expected /cloud mount")
	}
}

func TestMountPaths(t *testing.T) {
	mm := NewMountManager()
	mm.Mount("/a", newMockFS("a"))
	mm.Mount("/ab", newMockFS("ab"))
	mm.Mount("/abc", newMockFS("abc"))

	paths := mm.MountPaths()
	if len(paths) != 3 {
		t.Errorf("expected 3 paths, got %d", len(paths))
	}
	// Should be sorted longest first
	if paths[0] != "/abc" {
		t.Errorf("expected /abc first, got %s", paths[0])
	}
}

func TestUploadDownload(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	local := newMockFS("local")
	cloud := newMockFS("cloud")

	mm.Mount("/local", local)
	mm.Mount("/cloud", cloud)

	// Write to local
	content := "hello local"
	err := mm.Write(ctx, "/local/test.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Verify in mock
	if string(local.files["test.txt"]) != content {
		t.Error("file not written to correct backend")
	}

	// Read from local
	reader, err := mm.Read(ctx, "/local/test.txt")
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	data, _ := io.ReadAll(reader)
	reader.Close()
	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}

	// Write to cloud
	cloudContent := "hello cloud"
	err = mm.Write(ctx, "/cloud/file.txt", strings.NewReader(cloudContent))
	if err != nil {
		t.Fatalf("write to cloud failed: %v", err)
	}
	if string(cloud.files["file.txt"]) != cloudContent {
		t.Error("file not written to cloud backend")
	}
}

func TestNestedMounts(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	cloud := newMockFS("cloud")
	archive := newMockFS("archive")

	mm.Mount("/cloud", cloud)
	mm.Mount("/cloud/archive", archive)

	// Write to /cloud/archive should go to archive FS
	err := mm.Write(ctx, "/cloud/archive/old.txt", strings.NewReader("archived"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Should be in archive FS, not cloud FS
	if _, ok := archive.files["old.txt"]; !ok {
		t.Error("file should be in archive FS")
	}
	if _, ok := cloud.files["archive/old.txt"]; ok {
		t.Error("file should NOT be in cloud FS")
	}

	// Write to /cloud should go to cloud FS
	err = mm.Write(ctx, "/cloud/new.txt", strings.NewReader("cloud file"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if _, ok := cloud.files["new.txt"]; !ok {
		t.Error("file should be in cloud FS")
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	local := newMockFS("local")
	mm.Mount("/local", local)

	// Write then delete
	mm.Write(ctx, "/local/test.txt", strings.NewReader("content"))
	err := mm.Delete(ctx, "/local/test.txt")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	exists, _ := mm.FileExists(ctx, "/local/test.txt")
	if exists {
		t.Error("file should not exist after delete")
	}
}

func TestExists(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	local := newMockFS("local")
	mm.Mount("/local", local)

	mm.Write(ctx, "/local/test.txt", strings.NewReader("content"))

	exists, err := mm.FileExists(ctx, "/local/test.txt")
	if err != nil {
		t.Fatalf("file exists check failed: %v", err)
	}
	if !exists {
		t.Error("expected file to exist")
	}

	exists, err = mm.FileExists(ctx, "/local/nonexistent.txt")
	if err != nil {
		t.Fatalf("file exists check failed: %v", err)
	}
	if exists {
		t.Error("expected file to not exist")
	}
}

func TestFileInfo(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	local := newMockFS("local")
	mm.Mount("/local", local)

	content := "hello world"
	mm.Write(ctx, "/local/test.txt", strings.NewReader(content))

	info, err := mm.Stat(ctx, "/local/test.txt")
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), info.Size)
	}
}

func TestListRoot(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	mm.Mount("/local", newMockFS("local"))
	mm.Mount("/cloud", newMockFS("cloud"))
	mm.Mount("/cache", newMockFS("cache"))

	files, err := mm.ListContents(ctx, "/", false)
	if err != nil {
		t.Fatalf("list contents failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 mount points, got %d", len(files))
	}

	// All should be directories
	for _, f := range files {
		if !f.IsDir {
			t.Errorf("mount point %s should be a directory", f.Name)
		}
	}
}

func TestListMountContent(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	local := newMockFS("local")
	mm.Mount("/local", local)

	mm.Write(ctx, "/local/file1.txt", strings.NewReader("1"))
	mm.Write(ctx, "/local/file2.txt", strings.NewReader("2"))

	files, err := mm.ListContents(ctx, "/local", false)
	if err != nil {
		t.Fatalf("list contents failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestCreateDeleteDir(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	local := newMockFS("local")
	mm.Mount("/local", local)

	err := mm.CreateDir(ctx, "/local/subdir")
	if err != nil {
		t.Fatalf("create dir failed: %v", err)
	}
	if !local.dirs["subdir"] {
		t.Error("directory not created")
	}

	err = mm.DeleteDir(ctx, "/local/subdir")
	if err != nil {
		t.Fatalf("delete dir failed: %v", err)
	}
	if local.dirs["subdir"] {
		t.Error("directory not deleted")
	}
}

func TestCopySameMount(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	local := newMockCopierFS("local")
	mm.Mount("/local", local)

	local.files["src.txt"] = []byte("content")

	err := mm.Copy(ctx, "/local/src.txt", "/local/dst.txt")
	if err != nil {
		t.Fatalf("copy failed: %v", err)
	}

	// Should use native copy
	if !local.copyCalled {
		t.Error("expected native Copy to be called")
	}

	// Both files should exist
	if _, ok := local.files["src.txt"]; !ok {
		t.Error("source file should still exist")
	}
	if _, ok := local.files["dst.txt"]; !ok {
		t.Error("destination file should exist")
	}
}

func TestCopyCrossMount(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	local := newMockFS("local")
	cloud := newMockFS("cloud")
	mm.Mount("/local", local)
	mm.Mount("/cloud", cloud)

	local.files["file.txt"] = []byte("cross mount content")

	err := mm.Copy(ctx, "/local/file.txt", "/cloud/file.txt")
	if err != nil {
		t.Fatalf("cross-mount copy failed: %v", err)
	}

	// Source should still exist
	if _, ok := local.files["file.txt"]; !ok {
		t.Error("source file should still exist")
	}
	// Destination should exist
	if data, ok := cloud.files["file.txt"]; !ok {
		t.Error("destination file should exist")
	} else if string(data) != "cross mount content" {
		t.Error("content mismatch")
	}
}

func TestMoveSameMount(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	local := newMockMoverFS("local")
	mm.Mount("/local", local)

	local.files["src.txt"] = []byte("content")

	err := mm.Move(ctx, "/local/src.txt", "/local/dst.txt")
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}

	// Should use native move
	if !local.moveCalled {
		t.Error("expected native Move to be called")
	}

	// Source should not exist
	if _, ok := local.files["src.txt"]; ok {
		t.Error("source file should not exist after move")
	}
	// Destination should exist
	if _, ok := local.files["dst.txt"]; !ok {
		t.Error("destination file should exist")
	}
}

func TestMoveCrossMount(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	local := newMockFS("local")
	cloud := newMockFS("cloud")
	mm.Mount("/local", local)
	mm.Mount("/cloud", cloud)

	local.files["file.txt"] = []byte("moving content")

	err := mm.Move(ctx, "/local/file.txt", "/cloud/file.txt")
	if err != nil {
		t.Fatalf("cross-mount move failed: %v", err)
	}

	// Source should not exist
	if _, ok := local.files["file.txt"]; ok {
		t.Error("source file should not exist after move")
	}
	// Destination should exist
	if data, ok := cloud.files["file.txt"]; !ok {
		t.Error("destination file should exist")
	} else if string(data) != "moving content" {
		t.Error("content mismatch")
	}
}

func TestResolveErrors(t *testing.T) {
	ctx := context.Background()
	mm := NewMountManager()
	mm.Mount("/local", newMockFS("local"))

	// Try to access unmounted path
	_, err := mm.Read(ctx, "/unknown/file.txt")
	if !errors.Is(err, ErrMountNotFound) {
		t.Errorf("expected ErrMountNotFound, got %v", err)
	}
}

func TestNormalizeMountPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/", "/"},
		{"/local", "/local"},
		{"local", "/local"},
		{"/local/", "/local"},
		{"/local//subdir", "/local/subdir"},
		{"//local", "/local"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeMountPath(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeMountPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConcurrency(t *testing.T) {
	mm := NewMountManager()
	local := newMockFS("local")
	mm.Mount("/local", local)

	ctx := context.Background()
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(i int) {
			path := "/local/file" + string(rune('0'+i)) + ".txt"
			mm.Write(ctx, path, strings.NewReader("content"))
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			mm.Mounts()
			mm.MountPaths()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}
