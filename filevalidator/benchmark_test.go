package filevalidator

import (
	"bytes"
	"mime/multipart"
	"testing"
)

// BenchmarkNew benchmarks the creation of a new validator
func BenchmarkNew(b *testing.B) {
	constraints := DefaultConstraints()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New(constraints)
	}
}

// BenchmarkNewDefault benchmarks the creation of a default validator
func BenchmarkNewDefault(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewDefault()
	}
}

// BenchmarkValidator_Validate_SmallFile benchmarks validation of a small file
func BenchmarkValidator_Validate_SmallFile(b *testing.B) {
	validator := NewDefault()
	content := []byte("test content")
	_ = &multipart.FileHeader{
		Filename: "test.txt",
		Size:     int64(len(content)),
	}

	// We can't easily mock the file opening part of multipart.FileHeader in a benchmark
	// without a real file or a complex mock.
	// So we'll benchmark ValidateBytes instead which is the core logic
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateBytes(content, "test.txt")
	}
}

// BenchmarkValidator_Validate_Image benchmarks validation of an image (header only)
func BenchmarkValidator_Validate_Image(b *testing.B) {
	// Minimal valid PNG header
	pngHeader := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // Signature
		0x00, 0x00, 0x00, 0x0D, // IHDR length
		0x49, 0x48, 0x44, 0x52, // IHDR chunk type
		0x00, 0x00, 0x00, 0x01, // Width
		0x00, 0x00, 0x00, 0x01, // Height
		0x08, 0x06, 0x00, 0x00, 0x00, // Bit depth, color type, etc.
		0x1F, 0x15, 0xC4, 0x89, // CRC
	}
	// Pad to 1KB to simulate a real file header read
	content := make([]byte, 1024)
	copy(content, pngHeader)

	validator := ForImages().Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateBytes(content, "image.png")
	}
}

// BenchmarkDetectMIME benchmarks the magic bytes detection
func BenchmarkDetectMIME(b *testing.B) {
	// Minimal valid PNG header
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	content := make([]byte, 512)
	copy(content, pngHeader)
	reader := bytes.NewReader(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.Seek(0, 0)
		DetectMIME(reader)
	}
}

// BenchmarkDetectMIMEFromBytes benchmarks the magic bytes detection from bytes
func BenchmarkDetectMIMEFromBytes(b *testing.B) {
	// Minimal valid PNG header
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	content := make([]byte, 512)
	copy(content, pngHeader)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectMIMEFromBytes(content)
	}
}
