package filevalidator

import (
	"regexp"
	"testing"
)

func TestBuilder_Basic(t *testing.T) {
	validator := NewBuilder().
		MaxSize(10 * MB).
		Accept("image/png").
		Extensions(".png").
		Build()

	constraints := validator.GetConstraints()

	if constraints.MaxFileSize != 10*MB {
		t.Errorf("MaxFileSize = %d, want %d", constraints.MaxFileSize, 10*MB)
	}
	if len(constraints.AcceptedTypes) != 1 || constraints.AcceptedTypes[0] != "image/png" {
		t.Errorf("AcceptedTypes = %v, want [image/png]", constraints.AcceptedTypes)
	}
	if len(constraints.AllowedExts) != 1 || constraints.AllowedExts[0] != ".png" {
		t.Errorf("AllowedExts = %v, want [.png]", constraints.AllowedExts)
	}
}

func TestBuilder_Chaining(t *testing.T) {
	validator := NewBuilder().
		MaxSize(5*MB).
		MinSize(1*KB).
		Accept("image/png", "image/jpeg").
		Extensions(".png", ".jpg").
		BlockExtensions(".exe").
		MaxNameLength(100).
		StrictMIME().
		RequireExtension().
		WithContentValidation().
		Build()

	c := validator.GetConstraints()

	if c.MaxFileSize != 5*MB {
		t.Error("MaxFileSize not set correctly")
	}
	if c.MinFileSize != 1*KB {
		t.Error("MinFileSize not set correctly")
	}
	if len(c.AcceptedTypes) != 2 {
		t.Error("AcceptedTypes not set correctly")
	}
	if len(c.AllowedExts) != 2 {
		t.Error("AllowedExts not set correctly")
	}
	if len(c.BlockedExts) == 0 || c.BlockedExts[len(c.BlockedExts)-1] != ".exe" {
		t.Error("BlockedExts not set correctly")
	}
	if c.MaxNameLength != 100 {
		t.Error("MaxNameLength not set correctly")
	}
	if !c.StrictMIMETypeValidation {
		t.Error("StrictMIMETypeValidation not set correctly")
	}
	if !c.RequireExtension {
		t.Error("RequireExtension not set correctly")
	}
	if !c.ContentValidationEnabled {
		t.Error("ContentValidationEnabled not set correctly")
	}
}

func TestBuilder_SizeRange(t *testing.T) {
	validator := NewBuilder().
		SizeRange(1*KB, 10*MB).
		Build()

	c := validator.GetConstraints()

	if c.MinFileSize != 1*KB {
		t.Errorf("MinFileSize = %d, want %d", c.MinFileSize, 1*KB)
	}
	if c.MaxFileSize != 10*MB {
		t.Errorf("MaxFileSize = %d, want %d", c.MaxFileSize, 10*MB)
	}
}

func TestBuilder_AcceptShortcuts(t *testing.T) {
	tests := []struct {
		name   string
		build  func() *FileValidator
		expect string
	}{
		{
			name:   "AcceptImages",
			build:  func() *FileValidator { return NewBuilder().AcceptImages().Build() },
			expect: string(AllowAllImages),
		},
		{
			name:   "AcceptDocuments",
			build:  func() *FileValidator { return NewBuilder().AcceptDocuments().Build() },
			expect: string(AllowAllDocuments),
		},
		{
			name:   "AcceptAudio",
			build:  func() *FileValidator { return NewBuilder().AcceptAudio().Build() },
			expect: string(AllowAllAudio),
		},
		{
			name:   "AcceptVideo",
			build:  func() *FileValidator { return NewBuilder().AcceptVideo().Build() },
			expect: string(AllowAllVideo),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.build()
			c := v.GetConstraints()
			found := false
			for _, at := range c.AcceptedTypes {
				if at == tt.expect {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("AcceptedTypes = %v, want to contain %s", c.AcceptedTypes, tt.expect)
			}
		})
	}
}

func TestBuilder_FileNamePattern(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z0-9_]+\.[a-z]+$`)

	validator := NewBuilder().
		FileNamePattern(pattern).
		Build()

	c := validator.GetConstraints()
	if c.FileNameRegex == nil {
		t.Error("FileNameRegex should be set")
	}
}

func TestBuilder_FileNamePatternString(t *testing.T) {
	validator := NewBuilder().
		FileNamePatternString(`^[a-z0-9_]+\.[a-z]+$`).
		Build()

	c := validator.GetConstraints()
	if c.FileNameRegex == nil {
		t.Error("FileNameRegex should be set")
	}
}

func TestBuilder_Registry(t *testing.T) {
	t.Run("WithDefaultRegistry", func(t *testing.T) {
		v := NewBuilder().WithDefaultRegistry().Build()
		c := v.GetConstraints()
		if c.ContentValidatorRegistry == nil {
			t.Error("ContentValidatorRegistry should be set")
		}
		if !c.ContentValidatorRegistry.HasValidator("image/png") {
			t.Error("Should have image/png validator")
		}
	})

	t.Run("WithMinimalRegistry", func(t *testing.T) {
		v := NewBuilder().WithMinimalRegistry().Build()
		c := v.GetConstraints()
		if c.ContentValidatorRegistry == nil {
			t.Error("ContentValidatorRegistry should be set")
		}
	})

	t.Run("WithCustomRegistry", func(t *testing.T) {
		registry := NewContentValidatorRegistry()
		v := NewBuilder().WithRegistry(registry).Build()
		c := v.GetConstraints()
		if c.ContentValidatorRegistry != registry {
			t.Error("Should use custom registry")
		}
	})
}

func TestBuilder_Empty(t *testing.T) {
	v := Empty().Build()
	c := v.GetConstraints()

	// Empty builder should have no restrictions
	if c.MaxFileSize != 0 {
		t.Errorf("MaxFileSize = %d, want 0", c.MaxFileSize)
	}
	if len(c.AcceptedTypes) != 0 {
		t.Errorf("AcceptedTypes = %v, want empty", c.AcceptedTypes)
	}
}

func TestPreset_ForImages(t *testing.T) {
	v := ForImages().Build()
	c := v.GetConstraints()

	if c.MaxFileSize != 10*MB {
		t.Errorf("MaxFileSize = %d, want %d", c.MaxFileSize, 10*MB)
	}
	if !c.ContentValidationEnabled {
		t.Error("ContentValidationEnabled should be true")
	}

	// Should accept images
	found := false
	for _, at := range c.AcceptedTypes {
		if at == string(AllowAllImages) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should accept images")
	}
}

func TestPreset_ForDocuments(t *testing.T) {
	v := ForDocuments().Build()
	c := v.GetConstraints()

	if c.MaxFileSize != 50*MB {
		t.Errorf("MaxFileSize = %d, want %d", c.MaxFileSize, 50*MB)
	}
}

func TestPreset_ForMedia(t *testing.T) {
	v := ForMedia().Build()
	c := v.GetConstraints()

	if c.MaxFileSize != 500*MB {
		t.Errorf("MaxFileSize = %d, want %d", c.MaxFileSize, 500*MB)
	}
}

func TestPreset_ForArchives(t *testing.T) {
	v := ForArchives().Build()
	c := v.GetConstraints()

	if c.MaxFileSize != 1*GB {
		t.Errorf("MaxFileSize = %d, want %d", c.MaxFileSize, 1*GB)
	}
}

func TestPreset_ForWeb(t *testing.T) {
	v := ForWeb().Build()
	c := v.GetConstraints()

	if c.MaxFileSize != 25*MB {
		t.Errorf("MaxFileSize = %d, want %d", c.MaxFileSize, 25*MB)
	}
}

func TestPreset_Strict(t *testing.T) {
	v := Strict().Build()
	c := v.GetConstraints()

	if !c.StrictMIMETypeValidation {
		t.Error("StrictMIMETypeValidation should be true")
	}
	if !c.RequireExtension {
		t.Error("RequireExtension should be true")
	}
	if !c.RequireContentValidation {
		t.Error("RequireContentValidation should be true")
	}
}

func TestPreset_Customization(t *testing.T) {
	// Presets can be customized further
	v := ForImages().
		MaxSize(5*MB).              // Override default 10MB
		Extensions(".png", ".jpg"). // Override default extensions
		Build()

	c := v.GetConstraints()

	if c.MaxFileSize != 5*MB {
		t.Errorf("MaxFileSize = %d, want %d (customized)", c.MaxFileSize, 5*MB)
	}
}

func TestBuilder_Constraints(t *testing.T) {
	b := NewBuilder().MaxSize(10 * MB)

	// Should be able to inspect constraints before building
	c := b.Constraints()
	if c.MaxFileSize != 10*MB {
		t.Error("Constraints() should return current state")
	}
}
