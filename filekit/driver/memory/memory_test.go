package memory

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gobeaver/beaver-kit/filekit"
)

func TestNew(t *testing.T) {
	t.Run("creates adapter with default config", func(t *testing.T) {
		a := New()
		if a == nil {
			t.Fatal("expected adapter to be created")
		}
		if a.maxSize != 0 {
			t.Errorf("expected maxSize=0, got %d", a.maxSize)
		}
	})

	t.Run("creates adapter with max size", func(t *testing.T) {
		a := New(Config{MaxSize: 1024})
		if a.maxSize != 1024 {
			t.Errorf("expected maxSize=1024, got %d", a.maxSize)
		}
	})
}

func TestWrite(t *testing.T) {
	ctx := context.Background()

	t.Run("writes file successfully", func(t *testing.T) {
		a := New()
		content := "hello world"

		err := a.Write(ctx, "test.txt", strings.NewReader(content))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file exists
		exists, err := a.FileExists(ctx, "test.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected file to exist")
		}

		// Verify size tracking
		if a.Size() != int64(len(content)) {
			t.Errorf("expected size=%d, got %d", len(content), a.Size())
		}
	})

	t.Run("fails on path traversal", func(t *testing.T) {
		a := New()

		err := a.Write(ctx, "../etc/passwd", strings.NewReader("malicious"))
		if err == nil {
			t.Fatal("expected error for path traversal")
		}
		if !filekit.IsPermission(err) && !strings.Contains(err.Error(), "not allowed") {
			t.Errorf("expected permission error, got: %v", err)
		}
	})

	t.Run("respects max size limit", func(t *testing.T) {
		a := New(Config{MaxSize: 10})

		err := a.Write(ctx, "large.txt", strings.NewReader("this is too large"))
		if err == nil {
			t.Fatal("expected error for exceeding max size")
		}
	})

	t.Run("prevents overwrite by default", func(t *testing.T) {
		a := New()

		err := a.Write(ctx, "test.txt", strings.NewReader("first"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = a.Write(ctx, "test.txt", strings.NewReader("second"))
		if err == nil {
			t.Fatal("expected error for overwrite")
		}
	})

	t.Run("allows overwrite with option", func(t *testing.T) {
		a := New()

		err := a.Write(ctx, "test.txt", strings.NewReader("first"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = a.Write(ctx, "test.txt", strings.NewReader("second"), filekit.WithOverwrite(true))
		if err != nil {
			t.Fatalf("unexpected error with overwrite: %v", err)
		}

		// Verify content was updated
		reader, err := a.Read(ctx, "test.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer reader.Close()

		data, _ := io.ReadAll(reader)
		if string(data) != "second" {
			t.Errorf("expected content='second', got '%s'", string(data))
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		a := New()

		err := a.Write(ctx, "a/b/c/test.txt", strings.NewReader("nested"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify parent directories exist
		exists, _ := a.DirExists(ctx, "a")
		if !exists {
			t.Error("expected directory 'a' to exist")
		}
		exists, _ = a.DirExists(ctx, "a/b")
		if !exists {
			t.Error("expected directory 'a/b' to exist")
		}
		exists, _ = a.DirExists(ctx, "a/b/c")
		if !exists {
			t.Error("expected directory 'a/b/c' to exist")
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		a := New()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := a.Write(ctx, "test.txt", strings.NewReader("content"))
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	})

	t.Run("sets content type from option", func(t *testing.T) {
		a := New()

		err := a.Write(ctx, "data", strings.NewReader("{}"), filekit.WithContentType("application/json"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, _ := a.Stat(ctx, "data")
		if info.ContentType != "application/json" {
			t.Errorf("expected content-type='application/json', got '%s'", info.ContentType)
		}
	})

	t.Run("detects content type from extension", func(t *testing.T) {
		a := New()

		err := a.Write(ctx, "image.png", strings.NewReader("fake png"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, _ := a.Stat(ctx, "image.png")
		if !strings.Contains(info.ContentType, "png") {
			t.Errorf("expected png content-type, got '%s'", info.ContentType)
		}
	})
}

func TestRead(t *testing.T) {
	ctx := context.Background()

	t.Run("reads file successfully", func(t *testing.T) {
		a := New()
		content := "hello world"
		a.Write(ctx, "test.txt", strings.NewReader(content))

		reader, err := a.Read(ctx, "test.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("unexpected error reading: %v", err)
		}
		if string(data) != content {
			t.Errorf("expected content='%s', got '%s'", content, string(data))
		}
	})

	t.Run("fails for non-existent file", func(t *testing.T) {
		a := New()

		_, err := a.Read(ctx, "nonexistent.txt")
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}
		if !filekit.IsNotExist(err) {
			t.Errorf("expected not exist error, got: %v", err)
		}
	})

	t.Run("returns independent readers", func(t *testing.T) {
		a := New()
		a.Write(ctx, "test.txt", strings.NewReader("hello"))

		reader1, _ := a.Read(ctx, "test.txt")
		reader2, _ := a.Read(ctx, "test.txt")

		// Read from first reader
		data1, _ := io.ReadAll(reader1)
		// Read from second reader - should get same content
		data2, _ := io.ReadAll(reader2)

		if !bytes.Equal(data1, data2) {
			t.Error("expected both readers to return same content")
		}

		reader1.Close()
		reader2.Close()
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes file successfully", func(t *testing.T) {
		a := New()
		a.Write(ctx, "test.txt", strings.NewReader("content"))

		err := a.Delete(ctx, "test.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		exists, _ := a.FileExists(ctx, "test.txt")
		if exists {
			t.Error("expected file to be deleted")
		}
	})

	t.Run("updates size tracking", func(t *testing.T) {
		a := New()
		content := "hello world"
		a.Write(ctx, "test.txt", strings.NewReader(content))

		initialSize := a.Size()
		a.Delete(ctx, "test.txt")

		if a.Size() != initialSize-int64(len(content)) {
			t.Errorf("expected size to decrease by %d", len(content))
		}
	})

	t.Run("fails for non-existent file", func(t *testing.T) {
		a := New()

		err := a.Delete(ctx, "nonexistent.txt")
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}
	})
}

func TestFileExists(t *testing.T) {
	ctx := context.Background()

	t.Run("returns true for existing file", func(t *testing.T) {
		a := New()
		a.Write(ctx, "test.txt", strings.NewReader("content"))

		exists, err := a.FileExists(ctx, "test.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected file to exist")
		}
	})

	t.Run("returns true for existing directory", func(t *testing.T) {
		a := New()
		a.CreateDir(ctx, "mydir")

		exists, err := a.DirExists(ctx, "mydir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected directory to exist")
		}
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		a := New()

		exists, err := a.FileExists(ctx, "nonexistent")
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

	t.Run("returns file info", func(t *testing.T) {
		a := New()
		content := "hello world"
		a.Write(ctx, "test.txt", strings.NewReader(content), filekit.WithMetadata(map[string]string{"key": "value"}))

		info, err := a.Stat(ctx, "test.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if info.Name != "test.txt" {
			t.Errorf("expected name='test.txt', got '%s'", info.Name)
		}
		if info.Size != int64(len(content)) {
			t.Errorf("expected size=%d, got %d", len(content), info.Size)
		}
		if info.IsDir {
			t.Error("expected IsDir=false")
		}
		if info.Metadata["key"] != "value" {
			t.Error("expected metadata to be preserved")
		}
		if info.ModTime.IsZero() {
			t.Error("expected ModTime to be set")
		}
	})

	t.Run("returns directory info", func(t *testing.T) {
		a := New()
		a.CreateDir(ctx, "mydir")

		info, err := a.Stat(ctx, "mydir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if info.Name != "mydir" {
			t.Errorf("expected name='mydir', got '%s'", info.Name)
		}
		if !info.IsDir {
			t.Error("expected IsDir=true")
		}
	})

	t.Run("fails for non-existent path", func(t *testing.T) {
		a := New()

		_, err := a.Stat(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent path")
		}
	})
}

func TestListContents(t *testing.T) {
	ctx := context.Background()

	t.Run("lists files in directory", func(t *testing.T) {
		a := New()
		a.Write(ctx, "dir/file1.txt", strings.NewReader("content1"))
		a.Write(ctx, "dir/file2.txt", strings.NewReader("content2"))
		a.CreateDir(ctx, "dir/subdir")

		files, err := a.ListContents(ctx, "dir", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(files) != 3 {
			t.Errorf("expected 3 items, got %d", len(files))
		}

		// Check names are present (sorted)
		names := make([]string, len(files))
		for i, f := range files {
			names[i] = f.Name
		}

		expected := []string{"file1.txt", "file2.txt", "subdir"}
		for i, name := range expected {
			if names[i] != name {
				t.Errorf("expected name[%d]='%s', got '%s'", i, name, names[i])
			}
		}
	})

	t.Run("lists root directory", func(t *testing.T) {
		a := New()
		a.Write(ctx, "file.txt", strings.NewReader("content"))
		a.CreateDir(ctx, "mydir")

		files, err := a.ListContents(ctx, "", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(files) != 2 {
			t.Errorf("expected 2 items, got %d", len(files))
		}
	})

	t.Run("fails for non-existent directory", func(t *testing.T) {
		a := New()

		_, err := a.ListContents(ctx, "nonexistent", false)
		if err == nil {
			t.Fatal("expected error for non-existent directory")
		}
	})

	t.Run("fails for file path", func(t *testing.T) {
		a := New()
		a.Write(ctx, "file.txt", strings.NewReader("content"))

		_, err := a.ListContents(ctx, "file.txt", false)
		if err == nil {
			t.Fatal("expected error for file path")
		}
	})

	t.Run("does not list nested files directly", func(t *testing.T) {
		a := New()
		a.Write(ctx, "dir/subdir/file.txt", strings.NewReader("content"))

		files, err := a.ListContents(ctx, "dir", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only list "subdir", not "subdir/file.txt"
		if len(files) != 1 {
			t.Errorf("expected 1 item, got %d", len(files))
		}
		if files[0].Name != "subdir" {
			t.Errorf("expected name='subdir', got '%s'", files[0].Name)
		}
	})
}

func TestCreateDir(t *testing.T) {
	ctx := context.Background()

	t.Run("creates directory", func(t *testing.T) {
		a := New()

		err := a.CreateDir(ctx, "mydir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		exists, _ := a.DirExists(ctx, "mydir")
		if !exists {
			t.Error("expected directory to exist")
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		a := New()

		err := a.CreateDir(ctx, "a/b/c")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, path := range []string{"a", "a/b", "a/b/c"} {
			exists, _ := a.DirExists(ctx, path)
			if !exists {
				t.Errorf("expected directory '%s' to exist", path)
			}
		}
	})

	t.Run("fails if file exists at path", func(t *testing.T) {
		a := New()
		a.Write(ctx, "file", strings.NewReader("content"))

		err := a.CreateDir(ctx, "file")
		if err == nil {
			t.Fatal("expected error when file exists at path")
		}
	})
}

func TestDeleteDir(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes empty directory", func(t *testing.T) {
		a := New()
		a.CreateDir(ctx, "mydir")

		err := a.DeleteDir(ctx, "mydir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		exists, _ := a.DirExists(ctx, "mydir")
		if exists {
			t.Error("expected directory to be deleted")
		}
	})

	t.Run("deletes directory with contents", func(t *testing.T) {
		a := New()
		a.Write(ctx, "dir/file1.txt", strings.NewReader("content1"))
		a.Write(ctx, "dir/file2.txt", strings.NewReader("content2"))
		a.Write(ctx, "dir/sub/file3.txt", strings.NewReader("content3"))

		err := a.DeleteDir(ctx, "dir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check directory is deleted
		exists, _ := a.DirExists(ctx, "dir")
		if exists {
			t.Error("expected directory to be deleted")
		}
		// Check files are deleted
		for _, path := range []string{"dir/file1.txt", "dir/file2.txt", "dir/sub/file3.txt"} {
			exists, _ := a.FileExists(ctx, path)
			if exists {
				t.Errorf("expected '%s' to be deleted", path)
			}
		}
	})

	t.Run("updates size tracking", func(t *testing.T) {
		a := New()
		a.Write(ctx, "dir/file.txt", strings.NewReader("hello"))

		a.DeleteDir(ctx, "dir")

		if a.Size() != 0 {
			t.Errorf("expected size=0, got %d", a.Size())
		}
	})

	t.Run("fails for non-existent directory", func(t *testing.T) {
		a := New()

		err := a.DeleteDir(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent directory")
		}
	})

	t.Run("fails for file", func(t *testing.T) {
		a := New()
		a.Write(ctx, "file.txt", strings.NewReader("content"))

		err := a.DeleteDir(ctx, "file.txt")
		if err == nil {
			t.Fatal("expected error when path is a file")
		}
	})
}

func TestClear(t *testing.T) {
	ctx := context.Background()

	t.Run("removes all files and directories", func(t *testing.T) {
		a := New()
		a.Write(ctx, "file1.txt", strings.NewReader("content1"))
		a.Write(ctx, "dir/file2.txt", strings.NewReader("content2"))
		a.CreateDir(ctx, "emptydir")

		a.Clear()

		if a.FileCount() != 0 {
			t.Errorf("expected 0 files, got %d", a.FileCount())
		}
		if a.Size() != 0 {
			t.Errorf("expected size=0, got %d", a.Size())
		}
	})
}

func TestFileCount(t *testing.T) {
	ctx := context.Background()

	t.Run("counts files correctly", func(t *testing.T) {
		a := New()

		if a.FileCount() != 0 {
			t.Errorf("expected 0 files initially")
		}

		a.Write(ctx, "file1.txt", strings.NewReader("content"))
		if a.FileCount() != 1 {
			t.Errorf("expected 1 file")
		}

		a.Write(ctx, "file2.txt", strings.NewReader("content"))
		if a.FileCount() != 2 {
			t.Errorf("expected 2 files")
		}

		a.Delete(ctx, "file1.txt")
		if a.FileCount() != 1 {
			t.Errorf("expected 1 file after delete")
		}
	})
}

func TestConcurrency(t *testing.T) {
	ctx := context.Background()
	a := New()

	t.Run("handles concurrent writes", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 100; i++ {
			go func(n int) {
				path := "file" + string(rune('0'+n%10)) + ".txt"
				a.Write(ctx, path, strings.NewReader("content"), filekit.WithOverwrite(true))
				done <- true
			}(i)
		}

		for i := 0; i < 100; i++ {
			<-done
		}

		// Should not panic or deadlock
	})

	t.Run("handles concurrent reads and writes", func(t *testing.T) {
		a.Write(ctx, "shared.txt", strings.NewReader("initial"))

		done := make(chan bool)
		for i := 0; i < 50; i++ {
			go func() {
				a.Read(ctx, "shared.txt")
				done <- true
			}()
			go func() {
				a.Write(ctx, "shared.txt", strings.NewReader("updated"), filekit.WithOverwrite(true))
				done <- true
			}()
		}

		for i := 0; i < 100; i++ {
			<-done
		}
	})
}

func TestWriteFile(t *testing.T) {
	ctx := context.Background()
	a := New()

	t.Run("returns not supported error", func(t *testing.T) {
		err := a.WriteFile(ctx, "dest.txt", "/local/file.txt")
		if err == nil {
			t.Fatal("expected error")
		}
		// Memory adapter doesn't support writing from local filesystem
	})
}

func TestModTime(t *testing.T) {
	ctx := context.Background()
	a := New()

	t.Run("sets modification time on write", func(t *testing.T) {
		before := time.Now()
		a.Write(ctx, "test.txt", strings.NewReader("content"))
		after := time.Now()

		info, _ := a.Stat(ctx, "test.txt")

		if info.ModTime.Before(before) || info.ModTime.After(after) {
			t.Error("expected ModTime to be between before and after write")
		}
	})
}

func TestImplementsInterface(t *testing.T) {
	var _ filekit.FileSystem = (*Adapter)(nil)
}
