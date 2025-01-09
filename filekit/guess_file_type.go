package filekit

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// Common MIME types
const (
	MIMETypeTextPlain       = "text/plain"
	MIMETypeTextHTML        = "text/html"
	MIMETypeTextCSS         = "text/css"
	MIMETypeTextJavaScript  = "text/javascript"
	MIMETypeApplicationJSON = "application/json"
	MIMETypeApplicationXML  = "application/xml"
	MIMETypeImageJPEG       = "image/jpeg"
	MIMETypeImagePNG        = "image/png"
	MIMETypeImageGIF        = "image/gif"
	MIMETypeImageSVG        = "image/svg+xml"
	MIMETypeImageWebP       = "image/webp"
	MIMETypeAudioMP3        = "audio/mpeg"
	MIMETypeAudioOGG        = "audio/ogg"
	MIMETypeVideoMP4        = "video/mp4"
	MIMETypeVideoWebM       = "video/webm"
	MIMETypeApplicationPDF  = "application/pdf"
	MIMETypeApplicationZip  = "application/zip"
)

// Common file extensions to MIME types mapping
var extensionToMIME = map[string]string{
	".txt":   MIMETypeTextPlain,
	".html":  MIMETypeTextHTML,
	".htm":   MIMETypeTextHTML,
	".css":   MIMETypeTextCSS,
	".js":    MIMETypeTextJavaScript,
	".json":  MIMETypeApplicationJSON,
	".xml":   MIMETypeApplicationXML,
	".jpg":   MIMETypeImageJPEG,
	".jpeg":  MIMETypeImageJPEG,
	".png":   MIMETypeImagePNG,
	".gif":   MIMETypeImageGIF,
	".svg":   MIMETypeImageSVG,
	".webp":  MIMETypeImageWebP,
	".mp3":   MIMETypeAudioMP3,
	".ogg":   MIMETypeAudioOGG,
	".mp4":   MIMETypeVideoMP4,
	".webm":  MIMETypeVideoWebM,
	".pdf":   MIMETypeApplicationPDF,
	".zip":   MIMETypeApplicationZip,
	".gz":    "application/gzip",
	".tar":   "application/x-tar",
	".csv":   "text/csv",
	".md":    "text/markdown",
	".doc":   "application/msword",
	".docx":  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":   "application/vnd.ms-excel",
	".xlsx":  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":   "application/vnd.ms-powerpoint",
	".pptx":  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".eot":   "application/vnd.ms-fontobject",
	".otf":   "font/otf",
}

// GuessContentType tries to determine the content type of a file from its path and data
func GuessContentType(filePath string, data []byte) string {
	// First try to determine content type from extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if contentType, ok := extensionToMIME[ext]; ok {
		return contentType
	}

	// If we can't determine from extension and we have data, detect from content
	if len(data) > 0 {
		return http.DetectContentType(data)
	}

	// As a last resort, use the standard library's mime package
	contentType := mime.TypeByExtension(ext)
	if contentType != "" {
		return contentType
	}

	// Fall back to octet-stream
	return "application/octet-stream"
}

// IsTextFile returns true if the file is a text file based on its MIME type
func IsTextFile(contentType string) bool {
	return strings.HasPrefix(contentType, "text/") ||
		contentType == MIMETypeApplicationJSON ||
		contentType == MIMETypeApplicationXML ||
		contentType == "application/javascript" ||
		contentType == "application/x-javascript"
}

// IsImageFile returns true if the file is an image file based on its MIME type
func IsImageFile(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// IsAudioFile returns true if the file is an audio file based on its MIME type
func IsAudioFile(contentType string) bool {
	return strings.HasPrefix(contentType, "audio/")
}

// IsVideoFile returns true if the file is a video file based on its MIME type
func IsVideoFile(contentType string) bool {
	return strings.HasPrefix(contentType, "video/")
}

// IsCompressedFile returns true if the file is a compressed file based on its MIME type
func IsCompressedFile(contentType string) bool {
	return contentType == MIMETypeApplicationZip ||
		contentType == "application/gzip" ||
		contentType == "application/x-tar" ||
		contentType == "application/x-7z-compressed" ||
		contentType == "application/x-rar-compressed"
}

// IsPDFFile returns true if the file is a PDF file based on its MIME type
func IsPDFFile(contentType string) bool {
	return contentType == MIMETypeApplicationPDF
}

// GetFileExtensionForMIME returns a suitable file extension for a given MIME type
func GetFileExtensionForMIME(contentType string) string {
	// Remove any parameters from the content type
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)

	// Check for common MIME types
	switch contentType {
	case MIMETypeTextPlain:
		return ".txt"
	case MIMETypeTextHTML:
		return ".html"
	case MIMETypeTextCSS:
		return ".css"
	case MIMETypeTextJavaScript:
		return ".js"
	case MIMETypeApplicationJSON:
		return ".json"
	case MIMETypeApplicationXML:
		return ".xml"
	case MIMETypeImageJPEG:
		return ".jpg"
	case MIMETypeImagePNG:
		return ".png"
	case MIMETypeImageGIF:
		return ".gif"
	case MIMETypeImageSVG:
		return ".svg"
	case MIMETypeImageWebP:
		return ".webp"
	case MIMETypeAudioMP3:
		return ".mp3"
	case MIMETypeAudioOGG:
		return ".ogg"
	case MIMETypeVideoMP4:
		return ".mp4"
	case MIMETypeVideoWebM:
		return ".webm"
	case MIMETypeApplicationPDF:
		return ".pdf"
	case MIMETypeApplicationZip:
		return ".zip"
	}

	// For unknown MIME types, try to get an extension from the mime package
	exts, err := mime.ExtensionsByType(contentType)
	if err == nil && len(exts) > 0 {
		return exts[0]
	}

	// Fall back to .bin for binary data
	return ".bin"
}
