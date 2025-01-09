package filevalidator

// MediaTypeGroup defines a categorization of MIME types
type MediaTypeGroup string

const (
	AllowAllImages    MediaTypeGroup = "image/*"
	AllowAllDocuments MediaTypeGroup = "document/*"
	AllowAllAudio     MediaTypeGroup = "audio/*"
	AllowAllVideo     MediaTypeGroup = "video/*"
	AllowAllText      MediaTypeGroup = "text/*"
	AllowAll          MediaTypeGroup = "*/*"
)

// Common MIME types mapping for each group
var mediaTypeGroups = map[MediaTypeGroup][]string{
	AllowAllImages: {
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
		"image/svg+xml",
		"image/tiff",
		"image/bmp",
		"image/heic",
		"image/heif",
	},
	AllowAllDocuments: {
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain",
		"text/csv",
		"text/rtf",
		"application/rtf",
	},
	AllowAllAudio: {
		"audio/mpeg",
		"audio/wav",
		"audio/ogg",
		"audio/midi",
		"audio/x-midi",
		"audio/aac",
		"audio/flac",
		"audio/mp4",
		"audio/webm",
		"audio/x-ms-wma",
	},
	AllowAllVideo: {
		"video/mp4",
		"video/mpeg",
		"video/webm",
		"video/quicktime",
		"video/x-msvideo",
		"video/x-ms-wmv",
		"video/3gpp",
		"video/x-flv",
	},
	AllowAllText: {
		"text/plain",
		"text/html",
		"text/css",
		"text/csv",
		"text/javascript",
		"text/xml",
		"text/markdown",
	},
}

// Common extension to MIME type mapping
var extensionToMimeType = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".bmp":  "image/bmp",
	".heic": "image/heic",
	".heif": "image/heif",

	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".txt":  "text/plain",
	".csv":  "text/csv",
	".rtf":  "text/rtf",

	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".mid":  "audio/midi",
	".midi": "audio/x-midi",
	".aac":  "audio/aac",
	".flac": "audio/flac",
	".m4a":  "audio/mp4",
	".wma":  "audio/x-ms-wma",

	".mp4":  "video/mp4",
	".mpeg": "video/mpeg",
	".mpg":  "video/mpeg",
	".webm": "video/webm",
	".mov":  "video/quicktime",
	".avi":  "video/x-msvideo",
	".wmv":  "video/x-ms-wmv",
	".3gp":  "video/3gpp",
	".flv":  "video/x-flv",

	".html":     "text/html",
	".htm":      "text/html",
	".css":      "text/css",
	".js":       "text/javascript",
	".xml":      "text/xml",
	".md":       "text/markdown",
	".markdown": "text/markdown",
}

// MIMETypeForExtension returns the MIME type for a given file extension
// Returns empty string if the extension is not recognized
func MIMETypeForExtension(ext string) string {
	return extensionToMimeType[ext]
}

// ExpandAcceptedTypes takes a slice of accepted types (which can include MediaTypeGroups)
// and returns a slice with all specific MIME types
func ExpandAcceptedTypes(acceptedTypes []string) []string {
	expanded := make([]string, 0)

	for _, acceptType := range acceptedTypes {
		// Check if the type is a MediaTypeGroup
		if groupTypes, exists := mediaTypeGroups[MediaTypeGroup(acceptType)]; exists {
			// Add all MIME types from the group
			expanded = append(expanded, groupTypes...)
		} else if acceptType == string(AllowAll) {
			// Special case: match all types
			expanded = append(expanded, "*/*")
		} else {
			// Add the specific MIME type as is
			expanded = append(expanded, acceptType)
		}
	}

	return expanded
}

// AddCustomMediaTypeMapping adds a custom file extension to MIME type mapping
func AddCustomMediaTypeMapping(ext string, mimeType string) {
	extensionToMimeType[ext] = mimeType
}

// AddCustomMediaTypeGroupMapping adds custom MIME types to a media type group
func AddCustomMediaTypeGroupMapping(group MediaTypeGroup, mimeTypes []string) {
	if existing, ok := mediaTypeGroups[group]; ok {
		mediaTypeGroups[group] = append(existing, mimeTypes...)
	} else {
		mediaTypeGroups[group] = mimeTypes
	}
}
