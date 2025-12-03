package zip

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gobeaver/beaver-kit/filekit"
)

func TestCreate(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	t.Run("creates new zip file", func(t *testing.T) {
		fs, err := Create(zipPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer fs.Close()

		// Verify file was created
		if _, err := os.Stat(zipPath); os.IsNotExist(err) {
			t.Error("expected zip file to be created")
		}
	})
}

func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	// Create a valid ZIP file first
	createTestZip(t, zipPath, map[string]string{
		"file1.txt":     "content1",
		"dir/file2.txt": "content2",
	})

	t.Run("opens existing zip file", func(t *testing.T) {
		fs, err := Open(zipPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer fs.Close()

		// Verify files are indexed
		exists, _ := fs.FileExists(context.Background(), "file1.txt")
		if !exists {
			t.Error("expected file1.txt to exist")
		}
	})

	t.Run("fails for non-existent file", func(t *testing.T) {
		_, err := Open(filepath.Join(tmpDir, "nonexistent.zip"))
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}

func TestWrite(t *testing.T) {
	ctx := context.Background()

	t.Run("writes file to new zip", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "test.zip")

		fs, _ := Create(zipPath)
		defer fs.Close()

		err := fs.Write(ctx, "test.txt", strings.NewReader("hello world"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file exists
		exists, _ := fs.FileExists(ctx, "test.txt")
		if !exists {
			t.Error("expected file to exist after write")
		}
	})

	t.Run("fails on read-only zip", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "test.zip")
		createTestZip(t, zipPath, map[string]string{"existing.txt": "content"})

		fs, _ := Open(zipPath)
		defer fs.Close()

		err := fs.Write(ctx, "new.txt", strings.NewReader("content"))
		if err == nil {
			t.Error("expected error for write on read-only zip")
		}
	})

	t.Run("prevents overwrite by default", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "test.zip")

		fs, _ := Create(zipPath)
		defer fs.Close()

		fs.Write(ctx, "test.txt", strings.NewReader("first"))
		err := fs.Write(ctx, "test.txt", strings.NewReader("second"))
		if err == nil {
			t.Error("expected error for overwrite")
		}
	})

	t.Run("allows overwrite with option", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "test.zip")

		fs, _ := OpenOrCreate(zipPath)
		defer fs.Close()

		fs.Write(ctx, "test.txt", strings.NewReader("first"))
		err := fs.Write(ctx, "test.txt", strings.NewReader("second"), filekit.WithOverwrite(true))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "test.zip")

		fs, _ := Create(zipPath)
		defer fs.Close()

		err := fs.Write(ctx, "a/b/c/test.txt", strings.NewReader("nested"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify parent directories exist
		exists, _ := fs.DirExists(ctx, "a")
		if !exists {
			t.Error("expected directory 'a' to exist")
		}
	})

	t.Run("fails on path traversal", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "test.zip")

		fs, _ := Create(zipPath)
		defer fs.Close()

		err := fs.Write(ctx, "../etc/passwd", strings.NewReader("malicious"))
		if err == nil {
			t.Error("expected error for path traversal")
		}
	})
}

func TestRead(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	createTestZip(t, zipPath, map[string]string{
		"file.txt": "hello world",
	})

	t.Run("reads file from zip", func(t *testing.T) {
		fs, _ := Open(zipPath)
		defer fs.Close()

		reader, err := fs.Read(ctx, "file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer reader.Close()

		data, _ := io.ReadAll(reader)
		if string(data) != "hello world" {
			t.Errorf("expected 'hello world', got '%s'", string(data))
		}
	})

	t.Run("fails for non-existent file", func(t *testing.T) {
		fs, _ := Open(zipPath)
		defer fs.Close()

		_, err := fs.Read(ctx, "nonexistent.txt")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("reads newly written file", func(t *testing.T) {
		newZipPath := filepath.Join(tmpDir, "new.zip")
		fs, _ := Create(newZipPath)
		defer fs.Close()

		fs.Write(ctx, "test.txt", strings.NewReader("new content"))

		reader, err := fs.Read(ctx, "test.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer reader.Close()

		data, _ := io.ReadAll(reader)
		if string(data) != "new content" {
			t.Errorf("expected 'new content', got '%s'", string(data))
		}
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("deletes file in read-write mode", func(t *testing.T) {
		zipPath := filepath.Join(tmpDir, "delete-test.zip")
		createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

		fs, _ := OpenOrCreate(zipPath)
		defer fs.Close()

		err := fs.Delete(ctx, "file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		exists, _ := fs.FileExists(ctx, "file.txt")
		if exists {
			t.Error("expected file to be deleted")
		}
	})

	t.Run("fails on read-only zip", func(t *testing.T) {
		zipPath := filepath.Join(tmpDir, "readonly-test.zip")
		createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

		fs, _ := Open(zipPath)
		defer fs.Close()

		err := fs.Delete(ctx, "file.txt")
		if err == nil {
			t.Error("expected error for delete on read-only zip")
		}
	})
}

func TestFileExists(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	createTestZip(t, zipPath, map[string]string{
		"file.txt":     "content",
		"dir/file.txt": "nested",
	})

	t.Run("returns true for existing file", func(t *testing.T) {
		fs, _ := Open(zipPath)
		defer fs.Close()

		exists, err := fs.FileExists(ctx, "file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected file to exist")
		}
	})

	t.Run("returns false for directory", func(t *testing.T) {
		fs, _ := Open(zipPath)
		defer fs.Close()

		// FileExists should return false for directories (use DirExists for directories)
		exists, err := fs.FileExists(ctx, "dir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Error("FileExists should return false for directories")
		}

		// DirExists should return true for the directory
		dirExists, err := fs.DirExists(ctx, "dir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !dirExists {
			t.Error("expected directory to exist via DirExists")
		}
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		fs, _ := Open(zipPath)
		defer fs.Close()

		exists, err := fs.FileExists(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Error("expected path to not exist")
		}
	})
}

func TestStat(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	createTestZip(t, zipPath, map[string]string{
		"file.txt": "hello world",
	})

	t.Run("returns file info", func(t *testing.T) {
		fs, _ := Open(zipPath)
		defer fs.Close()

		info, err := fs.Stat(ctx, "file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if info.Name != "file.txt" {
			t.Errorf("expected name='file.txt', got '%s'", info.Name)
		}
		if info.Size != 11 { // "hello world" = 11 bytes
			t.Errorf("expected size=11, got %d", info.Size)
		}
		if info.IsDir {
			t.Error("expected IsDir=false")
		}
	})

	t.Run("fails for non-existent file", func(t *testing.T) {
		fs, _ := Open(zipPath)
		defer fs.Close()

		_, err := fs.Stat(ctx, "nonexistent.txt")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}

func TestListContents(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	createTestZip(t, zipPath, map[string]string{
		"file1.txt":      "content1",
		"file2.txt":      "content2",
		"dir/file3.txt":  "content3",
		"dir/nested.txt": "nested",
	})

	t.Run("lists root directory", func(t *testing.T) {
		fs, _ := Open(zipPath)
		defer fs.Close()

		files, err := fs.ListContents(ctx, "", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have file1.txt, file2.txt, dir
		if len(files) != 3 {
			t.Errorf("expected 3 items, got %d", len(files))
		}
	})

	t.Run("lists subdirectory", func(t *testing.T) {
		fs, _ := Open(zipPath)
		defer fs.Close()

		files, err := fs.ListContents(ctx, "dir", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have file3.txt and nested.txt
		if len(files) != 2 {
			t.Errorf("expected 2 items, got %d", len(files))
		}
	})

	t.Run("fails for non-existent directory", func(t *testing.T) {
		fs, _ := Open(zipPath)
		defer fs.Close()

		_, err := fs.ListContents(ctx, "nonexistent", false)
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})
}

func TestCreateDir(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("creates directory", func(t *testing.T) {
		zipPath := filepath.Join(tmpDir, "createdir.zip")
		fs, _ := Create(zipPath)
		defer fs.Close()

		err := fs.CreateDir(ctx, "mydir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		exists, _ := fs.DirExists(ctx, "mydir")
		if !exists {
			t.Error("expected directory to exist")
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		zipPath := filepath.Join(tmpDir, "nested.zip")
		fs, _ := Create(zipPath)
		defer fs.Close()

		err := fs.CreateDir(ctx, "a/b/c")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, dir := range []string{"a", "a/b", "a/b/c"} {
			exists, _ := fs.DirExists(ctx, dir)
			if !exists {
				t.Errorf("expected directory '%s' to exist", dir)
			}
		}
	})
}

func TestDeleteDir(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("deletes directory and contents", func(t *testing.T) {
		zipPath := filepath.Join(tmpDir, "deletedir.zip")
		createTestZip(t, zipPath, map[string]string{
			"dir/file1.txt": "content1",
			"dir/file2.txt": "content2",
			"other.txt":     "other",
		})

		fs, _ := OpenOrCreate(zipPath)
		defer fs.Close()

		err := fs.DeleteDir(ctx, "dir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// dir should be deleted
		exists, _ := fs.FileExists(ctx, "dir")
		if exists {
			t.Error("expected directory to be deleted")
		}

		// other.txt should still exist
		exists, _ = fs.FileExists(ctx, "other.txt")
		if !exists {
			t.Error("expected other.txt to still exist")
		}
	})
}

func TestOpenOrCreate(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("creates new file if not exists", func(t *testing.T) {
		zipPath := filepath.Join(tmpDir, "new.zip")

		fs, err := OpenOrCreate(zipPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		fs.Close()

		// File should exist after close
		if _, err := os.Stat(zipPath); os.IsNotExist(err) {
			t.Error("expected zip file to be created")
		}
	})

	t.Run("opens existing file", func(t *testing.T) {
		zipPath := filepath.Join(tmpDir, "existing.zip")
		createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

		fs, err := OpenOrCreate(zipPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer fs.Close()

		exists, _ := fs.FileExists(context.Background(), "file.txt")
		if !exists {
			t.Error("expected existing file to be accessible")
		}
	})
}

func TestZipPersistence(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "persist.zip")

	t.Run("persists data after close", func(t *testing.T) {
		// Create and write
		fs, _ := Create(zipPath)
		fs.Write(ctx, "file.txt", strings.NewReader("persisted content"))
		fs.Close()

		// Reopen and verify
		fs2, _ := Open(zipPath)
		defer fs2.Close()

		reader, err := fs2.Read(ctx, "file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer reader.Close()

		data, _ := io.ReadAll(reader)
		if string(data) != "persisted content" {
			t.Errorf("expected 'persisted content', got '%s'", string(data))
		}
	})

	t.Run("persists modifications in read-write mode", func(t *testing.T) {
		modZipPath := filepath.Join(tmpDir, "modify.zip")
		createTestZip(t, modZipPath, map[string]string{"original.txt": "original"})

		// Modify
		fs, _ := OpenOrCreate(modZipPath)
		fs.Write(ctx, "new.txt", strings.NewReader("new content"))
		fs.Delete(ctx, "original.txt")
		fs.Close()

		// Verify
		fs2, _ := Open(modZipPath)
		defer fs2.Close()

		exists, _ := fs2.FileExists(ctx, "new.txt")
		if !exists {
			t.Error("expected new.txt to exist")
		}

		exists, _ = fs2.FileExists(ctx, "original.txt")
		if exists {
			t.Error("expected original.txt to be deleted")
		}
	})
}

func TestImplementsInterface(t *testing.T) {
	var _ filekit.FileSystem = (*Adapter)(nil)
}

// Helper function to create a test ZIP file
func createTestZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create test zip: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)

	// Create directories first
	dirs := make(map[string]bool)
	for name := range files {
		dir := filepath.Dir(name)
		for dir != "." && dir != "/" && dir != "" {
			dirs[dir] = true
			dir = filepath.Dir(dir)
		}
	}

	for dir := range dirs {
		header := &zip.FileHeader{
			Name:   dir + "/",
			Method: zip.Store,
		}
		header.SetMode(os.ModeDir | 0755)
		_, err := w.CreateHeader(header)
		if err != nil {
			t.Fatalf("failed to create directory in test zip: %v", err)
		}
	}

	// Create files
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("failed to create file in test zip: %v", err)
		}
		_, err = io.Copy(fw, bytes.NewReader([]byte(content)))
		if err != nil {
			t.Fatalf("failed to write file in test zip: %v", err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close test zip: %v", err)
	}
}
