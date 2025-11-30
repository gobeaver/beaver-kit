package filevalidator

import (
	"testing"
)

func TestDefaultRegistry(t *testing.T) {
	registry := DefaultRegistry()

	// Should have validators registered
	if registry.Count() == 0 {
		t.Error("DefaultRegistry() should have validators registered")
	}

	// Check some expected MIME types
	expectedTypes := []string{
		"application/zip",
		"image/png",
		"image/jpeg",
		"application/pdf",
		"video/mp4",
		"audio/mpeg",
		"application/json",
	}

	for _, mime := range expectedTypes {
		if !registry.HasValidator(mime) {
			t.Errorf("DefaultRegistry() missing validator for %s", mime)
		}
	}
}

func TestGetDefaultRegistry_Singleton(t *testing.T) {
	r1 := GetDefaultRegistry()
	r2 := GetDefaultRegistry()

	if r1 != r2 {
		t.Error("GetDefaultRegistry() should return the same instance")
	}
}

func TestMinimalRegistry(t *testing.T) {
	registry := MinimalRegistry()

	// Should have fewer validators than default
	defaultRegistry := DefaultRegistry()
	if registry.Count() >= defaultRegistry.Count() {
		t.Error("MinimalRegistry() should have fewer validators than DefaultRegistry()")
	}

	// Should still have essential validators
	essentialTypes := []string{
		"application/zip",
		"image/png",
		"application/pdf",
	}

	for _, mime := range essentialTypes {
		if !registry.HasValidator(mime) {
			t.Errorf("MinimalRegistry() missing essential validator for %s", mime)
		}
	}
}

func TestImageOnlyRegistry(t *testing.T) {
	registry := ImageOnlyRegistry()

	// Should have image validators
	if !registry.HasValidator("image/png") {
		t.Error("ImageOnlyRegistry() should have image/png validator")
	}

	// Should NOT have non-image validators
	if registry.HasValidator("application/pdf") {
		t.Error("ImageOnlyRegistry() should not have application/pdf validator")
	}
	if registry.HasValidator("video/mp4") {
		t.Error("ImageOnlyRegistry() should not have video/mp4 validator")
	}
}

func TestDocumentOnlyRegistry(t *testing.T) {
	registry := DocumentOnlyRegistry()

	// Should have document validators
	if !registry.HasValidator("application/pdf") {
		t.Error("DocumentOnlyRegistry() should have application/pdf validator")
	}

	// Should NOT have non-document validators
	if registry.HasValidator("image/png") {
		t.Error("DocumentOnlyRegistry() should not have image/png validator")
	}
	if registry.HasValidator("video/mp4") {
		t.Error("DocumentOnlyRegistry() should not have video/mp4 validator")
	}
}

func TestMediaOnlyRegistry(t *testing.T) {
	registry := MediaOnlyRegistry()

	// Should have media validators
	mediaTypes := []string{
		"video/mp4",
		"audio/mpeg",
		"video/webm",
		"audio/wav",
	}

	for _, mime := range mediaTypes {
		if !registry.HasValidator(mime) {
			t.Errorf("MediaOnlyRegistry() missing validator for %s", mime)
		}
	}

	// Should NOT have non-media validators
	if registry.HasValidator("image/png") {
		t.Error("MediaOnlyRegistry() should not have image/png validator")
	}
	if registry.HasValidator("application/pdf") {
		t.Error("MediaOnlyRegistry() should not have application/pdf validator")
	}
}

func TestContentValidatorRegistry_Operations(t *testing.T) {
	registry := NewContentValidatorRegistry()

	// Test Register and HasValidator
	validator := DefaultImageValidator()
	registry.Register("image/test", validator)

	if !registry.HasValidator("image/test") {
		t.Error("HasValidator() should return true after Register()")
	}

	// Test GetValidator
	got := registry.GetValidator("image/test")
	if got != validator {
		t.Error("GetValidator() should return the registered validator")
	}

	// Test Unregister
	registry.Unregister("image/test")
	if registry.HasValidator("image/test") {
		t.Error("HasValidator() should return false after Unregister()")
	}

	// Test Clear
	registry.Register("image/a", validator)
	registry.Register("image/b", validator)
	registry.Clear()
	if registry.Count() != 0 {
		t.Error("Count() should be 0 after Clear()")
	}
}

func TestContentValidatorRegistry_Clone(t *testing.T) {
	original := DefaultRegistry()
	clone := original.Clone()

	// Should have same count
	if original.Count() != clone.Count() {
		t.Errorf("Clone() count = %d, want %d", clone.Count(), original.Count())
	}

	// Modifying clone should not affect original
	clone.Clear()
	if original.Count() == 0 {
		t.Error("Modifying clone should not affect original")
	}
}

func TestContentValidatorRegistry_RegisteredMIMETypes(t *testing.T) {
	registry := NewContentValidatorRegistry()
	registry.Register("image/png", DefaultImageValidator())
	registry.Register("image/jpeg", DefaultImageValidator())

	types := registry.RegisteredMIMETypes()

	if len(types) != 2 {
		t.Errorf("RegisteredMIMETypes() returned %d types, want 2", len(types))
	}

	// Check both types are present (order not guaranteed)
	found := map[string]bool{}
	for _, mime := range types {
		found[mime] = true
	}

	if !found["image/png"] {
		t.Error("RegisteredMIMETypes() missing image/png")
	}
	if !found["image/jpeg"] {
		t.Error("RegisteredMIMETypes() missing image/jpeg")
	}
}

func BenchmarkDefaultRegistry(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultRegistry()
	}
}

func BenchmarkRegistryLookup(b *testing.B) {
	registry := DefaultRegistry()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = registry.GetValidator("image/png")
		_ = registry.GetValidator("video/mp4")
		_ = registry.GetValidator("application/pdf")
	}
}
