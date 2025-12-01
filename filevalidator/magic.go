package filevalidator

import (
	"bytes"
	"io"
	"net/http"
	"strings"
)

// MagicSignature defines a file type signature
type MagicSignature struct {
	MIME   string
	Offset int    // Offset from start of file
	Magic  []byte // Magic bytes to match
}

// magicSignatures contains file signatures for MIME detection
// Ordered by specificity (most specific first)
var magicSignatures = []MagicSignature{
	// Images
	{MIME: "image/jpeg", Offset: 0, Magic: []byte{0xFF, 0xD8, 0xFF}},
	{MIME: "image/png", Offset: 0, Magic: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}},
	{MIME: "image/gif", Offset: 0, Magic: []byte("GIF87a")},
	{MIME: "image/gif", Offset: 0, Magic: []byte("GIF89a")},
	{MIME: "image/webp", Offset: 8, Magic: []byte("WEBP")}, // After RIFF header
	{MIME: "image/bmp", Offset: 0, Magic: []byte("BM")},
	{MIME: "image/tiff", Offset: 0, Magic: []byte{0x49, 0x49, 0x2A, 0x00}}, // Little endian
	{MIME: "image/tiff", Offset: 0, Magic: []byte{0x4D, 0x4D, 0x00, 0x2A}}, // Big endian
	{MIME: "image/x-icon", Offset: 0, Magic: []byte{0x00, 0x00, 0x01, 0x00}},
	{MIME: "image/heic", Offset: 4, Magic: []byte("ftypheic")},
	{MIME: "image/heic", Offset: 4, Magic: []byte("ftypmif1")},
	{MIME: "image/avif", Offset: 4, Magic: []byte("ftypavif")},

	// Documents
	{MIME: "application/pdf", Offset: 0, Magic: []byte("%PDF-")},

	// Archives - ZIP-based
	// Note: Office docs (DOCX, XLSX, PPTX) and JAR also use ZIP format
	// We detect as generic ZIP first, then refine based on content in refineDetection()
	{MIME: "application/zip", Offset: 0, Magic: []byte{0x50, 0x4B, 0x03, 0x04}},
	{MIME: "application/zip", Offset: 0, Magic: []byte{0x50, 0x4B, 0x05, 0x06}}, // Empty ZIP
	{MIME: "application/zip", Offset: 0, Magic: []byte{0x50, 0x4B, 0x07, 0x08}}, // Spanned ZIP

	// Archives - Other
	{MIME: "application/gzip", Offset: 0, Magic: []byte{0x1F, 0x8B}},
	{MIME: "application/x-tar", Offset: 257, Magic: []byte("ustar")}, // POSIX tar
	{MIME: "application/x-rar-compressed", Offset: 0, Magic: []byte("Rar!\x1a\x07\x00")},
	{MIME: "application/x-rar-compressed", Offset: 0, Magic: []byte("Rar!\x1a\x07\x01\x00")}, // RAR5
	{MIME: "application/x-7z-compressed", Offset: 0, Magic: []byte{'7', 'z', 0xBC, 0xAF, 0x27, 0x1C}},
	{MIME: "application/x-bzip2", Offset: 0, Magic: []byte("BZh")},
	{MIME: "application/x-xz", Offset: 0, Magic: []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}},

	// Audio
	{MIME: "audio/mpeg", Offset: 0, Magic: []byte("ID3")},      // MP3 with ID3
	{MIME: "audio/mpeg", Offset: 0, Magic: []byte{0xFF, 0xFB}}, // MP3 frame sync
	{MIME: "audio/mpeg", Offset: 0, Magic: []byte{0xFF, 0xFA}}, // MP3 frame sync
	{MIME: "audio/mpeg", Offset: 0, Magic: []byte{0xFF, 0xF3}}, // MP3 frame sync
	{MIME: "audio/mpeg", Offset: 0, Magic: []byte{0xFF, 0xF2}}, // MP3 frame sync
	{MIME: "audio/flac", Offset: 0, Magic: []byte("fLaC")},
	{MIME: "audio/ogg", Offset: 0, Magic: []byte("OggS")},
	{MIME: "audio/wav", Offset: 0, Magic: []byte("RIFF")},     // Check WAVE at offset 8
	{MIME: "audio/aac", Offset: 0, Magic: []byte{0xFF, 0xF1}}, // ADTS
	{MIME: "audio/aac", Offset: 0, Magic: []byte{0xFF, 0xF9}}, // ADTS
	{MIME: "audio/aac", Offset: 0, Magic: []byte("ADIF")},
	{MIME: "audio/midi", Offset: 0, Magic: []byte("MThd")},

	// Video
	{MIME: "video/webm", Offset: 0, Magic: []byte{0x1A, 0x45, 0xDF, 0xA3}},       // EBML (WebM/MKV)
	{MIME: "video/x-matroska", Offset: 0, Magic: []byte{0x1A, 0x45, 0xDF, 0xA3}}, // MKV uses same header
	{MIME: "video/mp4", Offset: 4, Magic: []byte("ftyp")},                        // MP4/M4V/M4A
	{MIME: "video/quicktime", Offset: 4, Magic: []byte("moov")},
	{MIME: "video/quicktime", Offset: 4, Magic: []byte("free")},
	{MIME: "video/x-msvideo", Offset: 0, Magic: []byte("RIFF")}, // Check AVI at offset 8
	{MIME: "video/x-flv", Offset: 0, Magic: []byte("FLV")},
	{MIME: "video/3gpp", Offset: 4, Magic: []byte("ftyp3g")},

	// Text/Data (these are harder to detect, use extension fallback)
	{MIME: "application/json", Offset: 0, Magic: []byte("{")},
	{MIME: "application/json", Offset: 0, Magic: []byte("[")},
	{MIME: "application/xml", Offset: 0, Magic: []byte("<?xml")},
	{MIME: "text/html", Offset: 0, Magic: []byte("<!DOCTYPE html")},
	{MIME: "text/html", Offset: 0, Magic: []byte("<!doctype html")},
	{MIME: "text/html", Offset: 0, Magic: []byte("<html")},
	{MIME: "text/html", Offset: 0, Magic: []byte("<HTML")},

	// Executables (for blocking)
	{MIME: "application/x-msdownload", Offset: 0, Magic: []byte("MZ")},                    // EXE/DLL
	{MIME: "application/x-mach-binary", Offset: 0, Magic: []byte{0xCF, 0xFA, 0xED, 0xFE}}, // Mach-O 64-bit
	{MIME: "application/x-mach-binary", Offset: 0, Magic: []byte{0xCE, 0xFA, 0xED, 0xFE}}, // Mach-O 32-bit
	{MIME: "application/x-executable", Offset: 0, Magic: []byte{0x7F, 'E', 'L', 'F'}},     // ELF

	// Fonts
	{MIME: "font/woff", Offset: 0, Magic: []byte("wOFF")},
	{MIME: "font/woff2", Offset: 0, Magic: []byte("wOF2")},
	{MIME: "font/otf", Offset: 0, Magic: []byte("OTTO")},
	{MIME: "font/ttf", Offset: 0, Magic: []byte{0x00, 0x01, 0x00, 0x00}},
}

// DetectMIME detects the MIME type from file content using magic bytes
// Falls back to http.DetectContentType if no magic match found
func DetectMIME(reader io.Reader) (string, error) {
	// Read enough bytes for detection (512 bytes covers most signatures)
	buf := make([]byte, 512)
	n, err := io.ReadFull(reader, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", NewValidationError(ErrorTypeMIME, "failed to read file for MIME detection")
	}
	buf = buf[:n]

	return DetectMIMEFromBytes(buf), nil
}

// DetectMIMEFromBytes detects MIME type from a byte slice
func DetectMIMEFromBytes(data []byte) string {
	if len(data) == 0 {
		return "application/octet-stream"
	}

	// Try magic signatures first
	mime := detectByMagic(data)
	if mime != "" {
		// Special case: distinguish similar formats
		mime = refineDetection(data, mime)
		return mime
	}

	// Fall back to http.DetectContentType
	contentType := http.DetectContentType(data)
	// Remove charset suffix
	if idx := strings.Index(contentType, ";"); idx > 0 {
		contentType = contentType[:idx]
	}

	return contentType
}

// detectByMagic checks data against known magic signatures
func detectByMagic(data []byte) string {
	for _, sig := range magicSignatures {
		if sig.Offset+len(sig.Magic) > len(data) {
			continue
		}

		if bytes.Equal(data[sig.Offset:sig.Offset+len(sig.Magic)], sig.Magic) {
			return sig.MIME
		}
	}
	return ""
}

// refineDetection handles cases where multiple formats share magic bytes
func refineDetection(data []byte, initialMIME string) string {
	switch initialMIME {
	case "audio/wav", "video/x-msvideo":
		// RIFF container - check specific format at offset 8
		if len(data) >= 12 {
			format := string(data[8:12])
			switch format {
			case "WAVE":
				return "audio/wav"
			case "AVI ":
				return "video/x-msvideo"
			case "WEBP":
				return "image/webp"
			}
		}
		return initialMIME

	case "application/zip":
		// Check if it's an Office document by looking for specific files
		// This is a heuristic - proper detection requires reading ZIP directory
		if len(data) >= 30 {
			// Office docs have [Content_Types].xml or word/, xl/, ppt/ paths early
			content := string(data)
			if strings.Contains(content, "[Content_Types]") ||
				strings.Contains(content, "word/") {
				return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
			}
			if strings.Contains(content, "xl/") {
				return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
			}
			if strings.Contains(content, "ppt/") {
				return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
			}
		}
		return initialMIME

	case "video/webm", "video/x-matroska":
		// Both use EBML header - would need to parse EBML to distinguish
		// Default to WebM as it's more common on web
		return "video/webm"

	case "video/mp4":
		// Check specific brand in ftyp box
		if len(data) >= 12 {
			brand := string(data[8:12])
			switch brand {
			case "M4A ":
				return "audio/mp4"
			case "M4V ":
				return "video/x-m4v"
			case "qt  ":
				return "video/quicktime"
			case "3gp4", "3gp5", "3gp6":
				return "video/3gpp"
			}
		}
		return initialMIME

	default:
		return initialMIME
	}
}

// IsBinaryMIME returns true if the MIME type is typically binary (not text)
func IsBinaryMIME(mime string) bool {
	textPrefixes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
	}

	for _, prefix := range textPrefixes {
		if strings.HasPrefix(mime, prefix) {
			return false
		}
	}

	return true
}

// IsExecutableMIME returns true if the MIME type indicates an executable
func IsExecutableMIME(mime string) bool {
	executableMIMEs := map[string]bool{
		"application/x-msdownload":    true,
		"application/x-msdos-program": true,
		"application/x-executable":    true,
		"application/x-mach-binary":   true,
		"application/x-sharedlib":     true,
		"application/x-dosexec":       true,
	}
	return executableMIMEs[mime]
}

// GetMIMECategory returns a human-readable category for a MIME type
func GetMIMECategory(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "audio/"):
		return "audio"
	case strings.HasPrefix(mime, "text/"):
		return "text"
	case strings.HasPrefix(mime, "font/"):
		return "font"
	case strings.Contains(mime, "zip") || strings.Contains(mime, "tar") ||
		strings.Contains(mime, "rar") || strings.Contains(mime, "7z") ||
		strings.Contains(mime, "gzip") || strings.Contains(mime, "bzip"):
		return "archive"
	case strings.Contains(mime, "document") || mime == "application/pdf" ||
		strings.Contains(mime, "msword") || strings.Contains(mime, "excel") ||
		strings.Contains(mime, "powerpoint"):
		return "document"
	case IsExecutableMIME(mime):
		return "executable"
	default:
		return "other"
	}
}
