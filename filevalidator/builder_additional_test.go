package filevalidator

import "testing"

// Additional tests for builder.go to increase coverage

func TestBuilder_AcceptAll(t *testing.T) {
	v := NewBuilder().AcceptAll().Build()
	c := v.GetConstraints()

	found := false
	for _, at := range c.AcceptedTypes {
		if at == string(AllowAll) {
			found = true
			break
		}
	}
	if !found {
		t.Error("AcceptAll should add AllowAll to accepted types")
	}
}

func TestBuilder_AllowNoExtension(t *testing.T) {
	v := NewBuilder().AllowNoExtension().Build()
	c := v.GetConstraints()

	if c.RequireExtension {
		t.Error("AllowNoExtension should set RequireExtension to false")
	}
}

func TestBuilder_WithoutContentValidation(t *testing.T) {
	v := NewBuilder().
		WithContentValidation().
		WithoutContentValidation().
		Build()

	c := v.GetConstraints()
	if c.ContentValidationEnabled {
		t.Error("WithoutContentValidation should disable content validation")
	}
}

func TestBuilder_AcceptMedia(t *testing.T) {
	v := NewBuilder().AcceptMedia().Build()
	c := v.GetConstraints()

	hasAudio := false
	hasVideo := false
	for _, at := range c.AcceptedTypes {
		if at == string(AllowAllAudio) {
			hasAudio = true
		}
		if at == string(AllowAllVideo) {
			hasVideo = true
		}
	}

	if !hasAudio || !hasVideo {
		t.Error("AcceptMedia should accept both audio and video")
	}
}

func TestBuilder_DangerousChars(t *testing.T) {
	v := NewBuilder().DangerousChars("../", ";", "|").Build()
	c := v.GetConstraints()

	if len(c.DangerousChars) < 3 {
		t.Error("DangerousChars should add dangerous characters")
	}
}
