package filevalidator

import "sync"

// DefaultRegistry returns a registry with all built-in validators registered
// This is efficient - validators are just struct pointers in a map
// Lookup is O(1), unused validators have zero CPU cost
func DefaultRegistry() *ContentValidatorRegistry {
	registry := NewContentValidatorRegistry()

	// Register all validators
	// Cost: ~20 map insertions = ~2µs total (negligible)

	// Archive validators
	archiveValidator := DefaultArchiveValidator()
	for _, mime := range archiveValidator.SupportedMIMETypes() {
		registry.Register(mime, archiveValidator)
	}

	// Image validator
	imageValidator := DefaultImageValidator()
	for _, mime := range imageValidator.SupportedMIMETypes() {
		registry.Register(mime, imageValidator)
	}

	// PDF validator
	pdfValidator := DefaultPDFValidator()
	for _, mime := range pdfValidator.SupportedMIMETypes() {
		registry.Register(mime, pdfValidator)
	}

	// Office validator (DOCX, XLSX, PPTX)
	officeValidator := DefaultOfficeValidator()
	for _, mime := range officeValidator.SupportedMIMETypes() {
		registry.Register(mime, officeValidator)
	}

	// TAR validator
	tarValidator := DefaultTarValidator()
	for _, mime := range tarValidator.SupportedMIMETypes() {
		registry.Register(mime, tarValidator)
	}

	// GZIP validator
	gzipValidator := DefaultGzipValidator()
	for _, mime := range gzipValidator.SupportedMIMETypes() {
		registry.Register(mime, gzipValidator)
	}

	// TAR.GZ validator
	tarGzValidator := DefaultTarGzValidator()
	for _, mime := range tarGzValidator.SupportedMIMETypes() {
		registry.Register(mime, tarGzValidator)
	}

	// Media validators
	mp4Validator := DefaultMP4Validator()
	for _, mime := range mp4Validator.SupportedMIMETypes() {
		registry.Register(mime, mp4Validator)
	}

	mp3Validator := DefaultMP3Validator()
	for _, mime := range mp3Validator.SupportedMIMETypes() {
		registry.Register(mime, mp3Validator)
	}

	webmValidator := DefaultWebMValidator()
	for _, mime := range webmValidator.SupportedMIMETypes() {
		registry.Register(mime, webmValidator)
	}

	wavValidator := DefaultWAVValidator()
	for _, mime := range wavValidator.SupportedMIMETypes() {
		registry.Register(mime, wavValidator)
	}

	oggValidator := DefaultOggValidator()
	for _, mime := range oggValidator.SupportedMIMETypes() {
		registry.Register(mime, oggValidator)
	}

	flacValidator := DefaultFLACValidator()
	for _, mime := range flacValidator.SupportedMIMETypes() {
		registry.Register(mime, flacValidator)
	}

	aviValidator := DefaultAVIValidator()
	for _, mime := range aviValidator.SupportedMIMETypes() {
		registry.Register(mime, aviValidator)
	}

	mkvValidator := DefaultMKVValidator()
	for _, mime := range mkvValidator.SupportedMIMETypes() {
		registry.Register(mime, mkvValidator)
	}

	movValidator := DefaultMOVValidator()
	for _, mime := range movValidator.SupportedMIMETypes() {
		registry.Register(mime, movValidator)
	}

	aacValidator := DefaultAACValidator()
	for _, mime := range aacValidator.SupportedMIMETypes() {
		registry.Register(mime, aacValidator)
	}

	// Text validators
	jsonValidator := DefaultJSONValidator()
	for _, mime := range jsonValidator.SupportedMIMETypes() {
		registry.Register(mime, jsonValidator)
	}

	xmlValidator := DefaultXMLValidator()
	for _, mime := range xmlValidator.SupportedMIMETypes() {
		registry.Register(mime, xmlValidator)
	}

	csvValidator := DefaultCSVValidator()
	for _, mime := range csvValidator.SupportedMIMETypes() {
		registry.Register(mime, csvValidator)
	}

	textValidator := DefaultPlainTextValidator()
	for _, mime := range textValidator.SupportedMIMETypes() {
		registry.Register(mime, textValidator)
	}

	return registry
}

// Global default registry (lazy initialized)
var (
	globalRegistry     *ContentValidatorRegistry
	globalRegistryOnce sync.Once
)

// GetDefaultRegistry returns the global default registry
// Thread-safe, lazy initialization
func GetDefaultRegistry() *ContentValidatorRegistry {
	globalRegistryOnce.Do(func() {
		globalRegistry = DefaultRegistry()
	})
	return globalRegistry
}

// RegisteredMIMETypes returns all MIME types that have validators registered
func (r *ContentValidatorRegistry) RegisteredMIMETypes() []string {
	types := make([]string, 0, len(r.validators))
	for mime := range r.validators {
		types = append(types, mime)
	}
	return types
}

// HasValidator returns true if a validator is registered for the given MIME type
func (r *ContentValidatorRegistry) HasValidator(mimeType string) bool {
	return r.validators[mimeType] != nil
}

// Count returns the number of registered validators
func (r *ContentValidatorRegistry) Count() int {
	return len(r.validators)
}

// Unregister removes a validator for a MIME type
func (r *ContentValidatorRegistry) Unregister(mimeType string) {
	delete(r.validators, mimeType)
}

// Clear removes all registered validators
func (r *ContentValidatorRegistry) Clear() {
	r.validators = make(map[string]ContentValidator)
}

// Clone creates a copy of the registry
func (r *ContentValidatorRegistry) Clone() *ContentValidatorRegistry {
	clone := NewContentValidatorRegistry()
	for mime, validator := range r.validators {
		clone.validators[mime] = validator
	}
	return clone
}

// MinimalRegistry returns a registry with only essential validators
// Use this if you want minimal memory footprint
func MinimalRegistry() *ContentValidatorRegistry {
	registry := NewContentValidatorRegistry()

	// Only register validators for high-risk formats
	archiveValidator := DefaultArchiveValidator()
	for _, mime := range archiveValidator.SupportedMIMETypes() {
		registry.Register(mime, archiveValidator)
	}

	imageValidator := DefaultImageValidator()
	for _, mime := range imageValidator.SupportedMIMETypes() {
		registry.Register(mime, imageValidator)
	}

	pdfValidator := DefaultPDFValidator()
	for _, mime := range pdfValidator.SupportedMIMETypes() {
		registry.Register(mime, pdfValidator)
	}

	return registry
}

// ImageOnlyRegistry returns a registry with only image validators
func ImageOnlyRegistry() *ContentValidatorRegistry {
	registry := NewContentValidatorRegistry()
	imageValidator := DefaultImageValidator()
	for _, mime := range imageValidator.SupportedMIMETypes() {
		registry.Register(mime, imageValidator)
	}
	return registry
}

// DocumentOnlyRegistry returns a registry with only document validators
func DocumentOnlyRegistry() *ContentValidatorRegistry {
	registry := NewContentValidatorRegistry()

	pdfValidator := DefaultPDFValidator()
	for _, mime := range pdfValidator.SupportedMIMETypes() {
		registry.Register(mime, pdfValidator)
	}

	officeValidator := DefaultOfficeValidator()
	for _, mime := range officeValidator.SupportedMIMETypes() {
		registry.Register(mime, officeValidator)
	}

	return registry
}

// MediaOnlyRegistry returns a registry with only media validators
func MediaOnlyRegistry() *ContentValidatorRegistry {
	registry := NewContentValidatorRegistry()

	validators := []ContentValidator{
		DefaultMP4Validator(),
		DefaultMP3Validator(),
		DefaultWebMValidator(),
		DefaultWAVValidator(),
		DefaultOggValidator(),
		DefaultFLACValidator(),
		DefaultAVIValidator(),
		DefaultMKVValidator(),
		DefaultMOVValidator(),
		DefaultAACValidator(),
	}

	for _, v := range validators {
		for _, mime := range v.SupportedMIMETypes() {
			registry.Register(mime, v)
		}
	}

	return registry
}
