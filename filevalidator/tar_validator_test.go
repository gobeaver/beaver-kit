package filevalidator

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestTarValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name      string
		validator *TarValidator
		makeTar   func() []byte
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid tar with files",
			validator: DefaultTarValidator(),
			makeTar: func() []byte {
				return createTar([]tarEntry{
					{name: "file1.txt", content: "hello"},
					{name: "file2.txt", content: "world"},
				})
			},
			wantErr: false,
		},
		{
			name:      "path traversal blocked",
			validator: DefaultTarValidator(),
			makeTar: func() []byte {
				return createTar([]tarEntry{
					{name: "../etc/passwd", content: "bad"},
				})
			},
			wantErr: true,
			errMsg:  "dangerous path",
		},
		{
			name:      "absolute path blocked",
			validator: DefaultTarValidator(),
			makeTar: func() []byte {
				return createTar([]tarEntry{
					{name: "/etc/passwd", content: "bad"},
				})
			},
			wantErr: true,
			errMsg:  "dangerous path",
		},
		{
			name:      "symlinks blocked by default",
			validator: DefaultTarValidator(),
			makeTar: func() []byte {
				return createTarWithSymlink("link", "/etc/passwd")
			},
			wantErr: true,
			errMsg:  "symlinks not allowed",
		},
		{
			name: "symlinks allowed when enabled",
			validator: func() *TarValidator {
				v := DefaultTarValidator()
				v.AllowSymlinks = true
				return v
			}(),
			makeTar: func() []byte {
				return createTarWithSymlink("link", "target.txt")
			},
			wantErr: false,
		},
		{
			name: "too many files",
			validator: func() *TarValidator {
				v := DefaultTarValidator()
				v.MaxFiles = 2
				return v
			}(),
			makeTar: func() []byte {
				return createTar([]tarEntry{
					{name: "file1.txt", content: "a"},
					{name: "file2.txt", content: "b"},
					{name: "file3.txt", content: "c"},
				})
			},
			wantErr: true,
			errMsg:  "too many files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.makeTar()
			reader := bytes.NewReader(data)

			err := tt.validator.ValidateContent(reader, int64(len(data)))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestTarGzValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name      string
		validator *TarGzValidator
		makeTarGz func() []byte
		wantErr   bool
	}{
		{
			name:      "valid tar.gz",
			validator: DefaultTarGzValidator(),
			makeTarGz: func() []byte {
				return createTarGz([]tarEntry{
					{name: "file.txt", content: "hello world"},
				})
			},
			wantErr: false,
		},
		{
			name:      "invalid gzip",
			validator: DefaultTarGzValidator(),
			makeTarGz: func() []byte {
				return []byte("not gzip data")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.makeTarGz()
			reader := bytes.NewReader(data)

			err := tt.validator.ValidateContent(reader, int64(len(data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGzipValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name      string
		validator *GzipValidator
		makeGzip  func() []byte
		wantErr   bool
	}{
		{
			name:      "valid gzip",
			validator: DefaultGzipValidator(),
			makeGzip: func() []byte {
				var buf bytes.Buffer
				w := gzip.NewWriter(&buf)
				_, _ = w.Write([]byte("hello world"))
				w.Close()
				return buf.Bytes()
			},
			wantErr: false,
		},
		{
			name:      "invalid gzip",
			validator: DefaultGzipValidator(),
			makeGzip: func() []byte {
				return []byte("not gzip")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.makeGzip()
			reader := bytes.NewReader(data)

			err := tt.validator.ValidateContent(reader, int64(len(data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

type tarEntry struct {
	name    string
	content string
}

func createTar(entries []tarEntry) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for _, e := range entries {
		hdr := &tar.Header{
			Name: e.name,
			Mode: 0644,
			Size: int64(len(e.content)),
		}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write([]byte(e.content))
	}

	tw.Close()
	return buf.Bytes()
}

func createTarWithSymlink(name, target string) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	hdr := &tar.Header{
		Name:     name,
		Linkname: target,
		Typeflag: tar.TypeSymlink,
		Mode:     0777,
	}
	_ = tw.WriteHeader(hdr)

	tw.Close()
	return buf.Bytes()
}

func createTarGz(entries []tarEntry) []byte {
	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		hdr := &tar.Header{
			Name: e.name,
			Mode: 0644,
			Size: int64(len(e.content)),
		}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write([]byte(e.content))
	}

	tw.Close()
	gw.Close()
	return buf.Bytes()
}
