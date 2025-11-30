package filevalidator

import (
	"bytes"
	"testing"
)

func TestDetectMIMEFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		// Images
		{
			name:     "JPEG",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'},
			expected: "image/jpeg",
		},
		{
			name:     "PNG",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expected: "image/png",
		},
		{
			name:     "GIF87a",
			data:     []byte("GIF87a"),
			expected: "image/gif",
		},
		{
			name:     "GIF89a",
			data:     []byte("GIF89a"),
			expected: "image/gif",
		},
		{
			name:     "BMP",
			data:     []byte("BM"),
			expected: "image/bmp",
		},
		{
			name:     "WebP",
			data:     []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P'},
			expected: "image/webp",
		},

		// Documents
		{
			name:     "PDF",
			data:     []byte("%PDF-1.4"),
			expected: "application/pdf",
		},

		// Archives
		{
			name:     "ZIP",
			data:     []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00},
			expected: "application/zip",
		},
		{
			name:     "GZIP",
			data:     []byte{0x1F, 0x8B, 0x08, 0x00},
			expected: "application/gzip",
		},
		{
			name:     "7z",
			data:     []byte{'7', 'z', 0xBC, 0xAF, 0x27, 0x1C},
			expected: "application/x-7z-compressed",
		},
		{
			name:     "RAR",
			data:     []byte("Rar!\x1a\x07\x00"),
			expected: "application/x-rar-compressed",
		},

		// Audio
		{
			name:     "MP3 with ID3",
			data:     []byte("ID3"),
			expected: "audio/mpeg",
		},
		{
			name:     "MP3 frame sync",
			data:     []byte{0xFF, 0xFB, 0x90},
			expected: "audio/mpeg",
		},
		{
			name:     "FLAC",
			data:     []byte("fLaC"),
			expected: "audio/flac",
		},
		{
			name:     "OGG",
			data:     []byte("OggS"),
			expected: "audio/ogg",
		},
		{
			name:     "WAV",
			data:     []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'A', 'V', 'E'},
			expected: "audio/wav",
		},

		// Video
		{
			name:     "WebM/MKV EBML",
			data:     []byte{0x1A, 0x45, 0xDF, 0xA3},
			expected: "video/webm",
		},
		{
			name:     "MP4 ftyp",
			data:     []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'},
			expected: "video/mp4",
		},
		{
			name:     "AVI",
			data:     []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'A', 'V', 'I', ' '},
			expected: "video/x-msvideo",
		},
		{
			name:     "FLV",
			data:     []byte("FLV"),
			expected: "video/x-flv",
		},

		// Text/Data
		{
			name:     "JSON object",
			data:     []byte(`{"key": "value"}`),
			expected: "application/json",
		},
		{
			name:     "JSON array",
			data:     []byte(`[1, 2, 3]`),
			expected: "application/json",
		},
		{
			name:     "XML",
			data:     []byte(`<?xml version="1.0"?>`),
			expected: "application/xml",
		},
		{
			name:     "HTML doctype",
			data:     []byte("<!DOCTYPE html>"),
			expected: "text/html",
		},

		// Executables
		{
			name:     "EXE/DLL (MZ)",
			data:     []byte("MZ"),
			expected: "application/x-msdownload",
		},
		{
			name:     "ELF",
			data:     []byte{0x7F, 'E', 'L', 'F'},
			expected: "application/x-executable",
		},

		// Fonts
		{
			name:     "WOFF",
			data:     []byte("wOFF"),
			expected: "font/woff",
		},
		{
			name:     "WOFF2",
			data:     []byte("wOF2"),
			expected: "font/woff2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectMIMEFromBytes(tt.data)
			if result != tt.expected {
				t.Errorf("DetectMIMEFromBytes() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectMIME(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	reader := bytes.NewReader(data)

	mime, err := DetectMIME(reader)
	if err != nil {
		t.Fatalf("DetectMIME() error = %v", err)
	}
	if mime != "image/png" {
		t.Errorf("DetectMIME() = %q, want %q", mime, "image/png")
	}
}

func TestRefineDetection_RIFF(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "WAV",
			data:     []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'A', 'V', 'E'},
			expected: "audio/wav",
		},
		{
			name:     "AVI",
			data:     []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'A', 'V', 'I', ' '},
			expected: "video/x-msvideo",
		},
		{
			name:     "WebP",
			data:     []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P'},
			expected: "image/webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectMIMEFromBytes(tt.data)
			if result != tt.expected {
				t.Errorf("DetectMIMEFromBytes() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRefineDetection_MP4Brands(t *testing.T) {
	tests := []struct {
		name     string
		brand    string
		expected string
	}{
		{"M4A audio", "M4A ", "audio/mp4"},
		{"M4V video", "M4V ", "video/x-m4v"},
		{"QuickTime", "qt  ", "video/quicktime"},
		{"3GP", "3gp4", "video/3gpp"},
		{"isom (default)", "isom", "video/mp4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build ftyp box: size(4) + "ftyp"(4) + brand(4)
			data := []byte{0x00, 0x00, 0x00, 0x14, 'f', 't', 'y', 'p'}
			data = append(data, []byte(tt.brand)...)

			result := DetectMIMEFromBytes(data)
			if result != tt.expected {
				t.Errorf("DetectMIMEFromBytes() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsBinaryMIME(t *testing.T) {
	tests := []struct {
		mime     string
		expected bool
	}{
		{"image/png", true},
		{"application/pdf", true},
		{"video/mp4", true},
		{"text/plain", false},
		{"text/html", false},
		{"application/json", false},
		{"application/xml", false},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			result := IsBinaryMIME(tt.mime)
			if result != tt.expected {
				t.Errorf("IsBinaryMIME(%q) = %v, want %v", tt.mime, result, tt.expected)
			}
		})
	}
}

func TestIsExecutableMIME(t *testing.T) {
	tests := []struct {
		mime     string
		expected bool
	}{
		{"application/x-msdownload", true},
		{"application/x-executable", true},
		{"application/x-mach-binary", true},
		{"application/pdf", false},
		{"image/png", false},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			result := IsExecutableMIME(tt.mime)
			if result != tt.expected {
				t.Errorf("IsExecutableMIME(%q) = %v, want %v", tt.mime, result, tt.expected)
			}
		})
	}
}

func TestGetMIMECategory(t *testing.T) {
	tests := []struct {
		mime     string
		expected string
	}{
		{"image/png", "image"},
		{"image/jpeg", "image"},
		{"video/mp4", "video"},
		{"audio/mpeg", "audio"},
		{"text/plain", "text"},
		{"font/woff", "font"},
		{"application/zip", "archive"},
		{"application/gzip", "archive"},
		{"application/pdf", "document"},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "document"},
		{"application/x-msdownload", "executable"},
		{"application/octet-stream", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			result := GetMIMECategory(tt.mime)
			if result != tt.expected {
				t.Errorf("GetMIMECategory(%q) = %q, want %q", tt.mime, result, tt.expected)
			}
		})
	}
}

func TestDetectMIMEFromBytes_Empty(t *testing.T) {
	result := DetectMIMEFromBytes([]byte{})
	if result != "application/octet-stream" {
		t.Errorf("DetectMIMEFromBytes(empty) = %q, want %q", result, "application/octet-stream")
	}
}
