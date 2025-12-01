package filevalidator

import "testing"

func TestAddCustomMediaTypeMapping(t *testing.T) {
	// Add a custom mapping
	AddCustomMediaTypeMapping(".custom", "application/x-custom")

	// Verify it was added
	mime := MIMETypeForExtension(".custom")
	if mime != "application/x-custom" {
		t.Errorf("MIMETypeForExtension(.custom) = %s, want application/x-custom", mime)
	}
}

func TestAddCustomMediaTypeGroupMapping(t *testing.T) {
	// Add custom MIME types to a group
	customMimes := []string{"application/x-custom1", "application/x-custom2"}
	AddCustomMediaTypeGroupMapping(AllowAllDocuments, customMimes)

	// Verify they were added
	expanded := ExpandAcceptedTypes([]string{string(AllowAllDocuments)})

	found1 := false
	found2 := false
	for _, mime := range expanded {
		if mime == "application/x-custom1" {
			found1 = true
		}
		if mime == "application/x-custom2" {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Custom MIME types were not added to the group")
	}
}

func TestAddCustomMediaTypeGroupMapping_NewGroup(t *testing.T) {
	// Create a new group
	newGroup := MediaTypeGroup("custom/*")
	customMimes := []string{"custom/type1", "custom/type2"}
	AddCustomMediaTypeGroupMapping(newGroup, customMimes)

	// Verify they were added
	expanded := ExpandAcceptedTypes([]string{string(newGroup)})

	if len(expanded) != 2 {
		t.Errorf("Expected 2 MIME types, got %d", len(expanded))
	}
}
