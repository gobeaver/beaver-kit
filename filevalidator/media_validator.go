package filevalidator

import (
	"fmt"
	"io"
)

// MediaValidator validates video and audio files by checking headers/magic bytes.
// Only reads the first few KB - does NOT load entire file.
type MediaValidator struct {
	MaxSize int64
}

// DefaultMediaValidator creates a media validator with sensible defaults
func DefaultMediaValidator() *MediaValidator {
	return &MediaValidator{
		MaxSize: 5 * GB,
	}
}

// MP4Validator validates MP4/M4A/M4V files
type MP4Validator struct {
	MaxSize int64
}

// DefaultMP4Validator creates an MP4 validator
func DefaultMP4Validator() *MP4Validator {
	return &MP4Validator{
		MaxSize: 5 * GB,
	}
}

// ValidateContent validates MP4 by checking for ftyp box
func (v *MP4Validator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read first 32 bytes for ftyp box
	header := make([]byte, 32)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return NewValidationError(ErrorTypeContent, "failed to read MP4 header")
	}
	header = header[:n]

	if !v.isValidMP4(header) {
		return NewValidationError(ErrorTypeContent, "invalid MP4 format - missing ftyp box")
	}

	return nil
}

func (v *MP4Validator) isValidMP4(header []byte) bool {
	if len(header) < 12 {
		return false
	}

	// MP4 starts with a box: [4 bytes size][4 bytes type]
	// ftyp box should be at offset 4
	if string(header[4:8]) == "ftyp" {
		return true
	}

	// Some files have ftyp at different offset, check for common brands
	mp4Brands := []string{"ftyp", "moov", "mdat", "free", "skip", "wide"}
	boxType := string(header[4:8])
	for _, brand := range mp4Brands {
		if boxType == brand {
			return true
		}
	}

	return false
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *MP4Validator) SupportedMIMETypes() []string {
	return []string{
		"video/mp4",
		"video/x-m4v",
		"audio/mp4",
		"audio/x-m4a",
	}
}

// MP3Validator validates MP3 audio files
type MP3Validator struct {
	MaxSize int64
}

// DefaultMP3Validator creates an MP3 validator
func DefaultMP3Validator() *MP3Validator {
	return &MP3Validator{
		MaxSize: 500 * MB,
	}
}

// ValidateContent validates MP3 by checking for ID3 tag or frame sync
func (v *MP3Validator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read first 10 bytes (ID3 header size)
	header := make([]byte, 10)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return NewValidationError(ErrorTypeContent, "failed to read MP3 header")
	}
	header = header[:n]

	if !v.isValidMP3(header) {
		return NewValidationError(ErrorTypeContent, "invalid MP3 format")
	}

	return nil
}

func (v *MP3Validator) isValidMP3(header []byte) bool {
	if len(header) < 3 {
		return false
	}

	// Check for ID3v2 tag
	if string(header[:3]) == "ID3" {
		return true
	}

	// Check for ID3v1 tag (at end of file, but we check beginning)
	// Check for MP3 frame sync (0xFF followed by 0xE* or 0xF*)
	if len(header) >= 2 {
		if header[0] == 0xFF && (header[1]&0xE0) == 0xE0 {
			return true
		}
	}

	return false
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *MP3Validator) SupportedMIMETypes() []string {
	return []string{
		"audio/mpeg",
		"audio/mp3",
	}
}

// WebMValidator validates WebM video files
type WebMValidator struct {
	MaxSize int64
}

// DefaultWebMValidator creates a WebM validator
func DefaultWebMValidator() *WebMValidator {
	return &WebMValidator{
		MaxSize: 5 * GB,
	}
}

// ValidateContent validates WebM by checking EBML header
func (v *WebMValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read first 32 bytes for EBML header
	header := make([]byte, 32)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return NewValidationError(ErrorTypeContent, "failed to read WebM header")
	}
	header = header[:n]

	if !v.isValidWebM(header) {
		return NewValidationError(ErrorTypeContent, "invalid WebM format - missing EBML header")
	}

	return nil
}

func (v *WebMValidator) isValidWebM(header []byte) bool {
	if len(header) < 4 {
		return false
	}

	// WebM/Matroska starts with EBML header: 0x1A 0x45 0xDF 0xA3
	return header[0] == 0x1A && header[1] == 0x45 && header[2] == 0xDF && header[3] == 0xA3
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *WebMValidator) SupportedMIMETypes() []string {
	return []string{
		"video/webm",
		"audio/webm",
	}
}

// WAVValidator validates WAV audio files
type WAVValidator struct {
	MaxSize int64
}

// DefaultWAVValidator creates a WAV validator
func DefaultWAVValidator() *WAVValidator {
	return &WAVValidator{
		MaxSize: 1 * GB,
	}
}

// ValidateContent validates WAV by checking RIFF/WAVE header
func (v *WAVValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read first 12 bytes for RIFF header
	header := make([]byte, 12)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return NewValidationError(ErrorTypeContent, "failed to read WAV header")
	}
	header = header[:n]

	if !v.isValidWAV(header) {
		return NewValidationError(ErrorTypeContent, "invalid WAV format - missing RIFF/WAVE header")
	}

	return nil
}

func (v *WAVValidator) isValidWAV(header []byte) bool {
	if len(header) < 12 {
		return false
	}

	// WAV format: "RIFF" + [4 bytes size] + "WAVE"
	return string(header[0:4]) == "RIFF" && string(header[8:12]) == "WAVE"
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *WAVValidator) SupportedMIMETypes() []string {
	return []string{
		"audio/wav",
		"audio/x-wav",
		"audio/wave",
	}
}

// OggValidator validates Ogg container files (Vorbis, Opus, Theora)
type OggValidator struct {
	MaxSize int64
}

// DefaultOggValidator creates an Ogg validator
func DefaultOggValidator() *OggValidator {
	return &OggValidator{
		MaxSize: 1 * GB,
	}
}

// ValidateContent validates Ogg by checking OggS magic
func (v *OggValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read first 4 bytes for OggS magic
	header := make([]byte, 4)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return NewValidationError(ErrorTypeContent, "failed to read Ogg header")
	}
	header = header[:n]

	if string(header) != "OggS" {
		return NewValidationError(ErrorTypeContent, "invalid Ogg format - missing OggS magic")
	}

	return nil
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *OggValidator) SupportedMIMETypes() []string {
	return []string{
		"audio/ogg",
		"video/ogg",
		"application/ogg",
	}
}

// FLACValidator validates FLAC audio files
type FLACValidator struct {
	MaxSize int64
}

// DefaultFLACValidator creates a FLAC validator
func DefaultFLACValidator() *FLACValidator {
	return &FLACValidator{
		MaxSize: 1 * GB,
	}
}

// ValidateContent validates FLAC by checking fLaC magic
func (v *FLACValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read first 4 bytes for fLaC magic
	header := make([]byte, 4)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return NewValidationError(ErrorTypeContent, "failed to read FLAC header")
	}
	header = header[:n]

	if string(header) != "fLaC" {
		return NewValidationError(ErrorTypeContent, "invalid FLAC format - missing fLaC magic")
	}

	return nil
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *FLACValidator) SupportedMIMETypes() []string {
	return []string{
		"audio/flac",
		"audio/x-flac",
	}
}

// AVIValidator validates AVI video files
type AVIValidator struct {
	MaxSize int64
}

// DefaultAVIValidator creates an AVI validator
func DefaultAVIValidator() *AVIValidator {
	return &AVIValidator{
		MaxSize: 5 * GB,
	}
}

// ValidateContent validates AVI by checking RIFF/AVI header
func (v *AVIValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read first 12 bytes for RIFF header
	header := make([]byte, 12)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return NewValidationError(ErrorTypeContent, "failed to read AVI header")
	}
	header = header[:n]

	if !v.isValidAVI(header) {
		return NewValidationError(ErrorTypeContent, "invalid AVI format - missing RIFF/AVI header")
	}

	return nil
}

func (v *AVIValidator) isValidAVI(header []byte) bool {
	if len(header) < 12 {
		return false
	}

	// AVI format: "RIFF" + [4 bytes size] + "AVI "
	return string(header[0:4]) == "RIFF" && string(header[8:12]) == "AVI "
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *AVIValidator) SupportedMIMETypes() []string {
	return []string{
		"video/x-msvideo",
		"video/avi",
	}
}

// MOVValidator validates QuickTime MOV files
type MOVValidator struct {
	MaxSize int64
}

// DefaultMOVValidator creates a MOV validator
func DefaultMOVValidator() *MOVValidator {
	return &MOVValidator{
		MaxSize: 5 * GB,
	}
}

// ValidateContent validates MOV by checking for qt/moov atoms
func (v *MOVValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read first 12 bytes
	header := make([]byte, 12)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return NewValidationError(ErrorTypeContent, "failed to read MOV header")
	}
	header = header[:n]

	if !v.isValidMOV(header) {
		return NewValidationError(ErrorTypeContent, "invalid MOV format")
	}

	return nil
}

func (v *MOVValidator) isValidMOV(header []byte) bool {
	if len(header) < 8 {
		return false
	}

	// MOV uses the same box structure as MP4
	// Common atoms: ftyp, moov, mdat, free, wide
	atomType := string(header[4:8])
	validAtoms := []string{"ftyp", "moov", "mdat", "free", "wide", "skip", "pnot"}

	for _, atom := range validAtoms {
		if atomType == atom {
			// For ftyp, check if it's a QuickTime brand
			if atomType == "ftyp" && len(header) >= 12 {
				brand := string(header[8:12])
				qtBrands := []string{"qt  ", "MSNV", "M4A ", "M4V "}
				for _, qb := range qtBrands {
					if brand == qb {
						return true
					}
				}
			}
			return true
		}
	}

	return false
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *MOVValidator) SupportedMIMETypes() []string {
	return []string{
		"video/quicktime",
	}
}

// MKVValidator validates Matroska video files
type MKVValidator struct {
	MaxSize int64
}

// DefaultMKVValidator creates an MKV validator
func DefaultMKVValidator() *MKVValidator {
	return &MKVValidator{
		MaxSize: 10 * GB,
	}
}

// ValidateContent validates MKV by checking EBML header (same as WebM)
func (v *MKVValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read first 32 bytes for EBML header
	header := make([]byte, 32)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return NewValidationError(ErrorTypeContent, "failed to read MKV header")
	}
	header = header[:n]

	// MKV uses same EBML header as WebM: 0x1A 0x45 0xDF 0xA3
	if len(header) < 4 || header[0] != 0x1A || header[1] != 0x45 || header[2] != 0xDF || header[3] != 0xA3 {
		return NewValidationError(ErrorTypeContent, "invalid MKV format - missing EBML header")
	}

	return nil
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *MKVValidator) SupportedMIMETypes() []string {
	return []string{
		"video/x-matroska",
		"audio/x-matroska",
	}
}

// AACValidator validates AAC audio files
type AACValidator struct {
	MaxSize int64
}

// DefaultAACValidator creates an AAC validator
func DefaultAACValidator() *AACValidator {
	return &AACValidator{
		MaxSize: 500 * MB,
	}
}

// ValidateContent validates AAC by checking ADTS header or ID3 tag
func (v *AACValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read first 10 bytes
	header := make([]byte, 10)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return NewValidationError(ErrorTypeContent, "failed to read AAC header")
	}
	header = header[:n]

	if !v.isValidAAC(header) {
		return NewValidationError(ErrorTypeContent, "invalid AAC format")
	}

	return nil
}

func (v *AACValidator) isValidAAC(header []byte) bool {
	if len(header) < 2 {
		return false
	}

	// Check for ID3 tag
	if len(header) >= 3 && string(header[:3]) == "ID3" {
		return true
	}

	// Check for ADTS sync word (0xFFF)
	if header[0] == 0xFF && (header[1]&0xF0) == 0xF0 {
		return true
	}

	// Check for ADIF header
	if len(header) >= 4 && string(header[:4]) == "ADIF" {
		return true
	}

	return false
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *AACValidator) SupportedMIMETypes() []string {
	return []string{
		"audio/aac",
		"audio/x-aac",
	}
}
