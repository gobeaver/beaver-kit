package filevalidator

import (
	"bytes"
	"testing"
)

func TestMP4Validator_ValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "valid mp4 with ftyp",
			// ftyp box: size(4) + "ftyp"(4) + brand(4)
			data:    []byte{0x00, 0x00, 0x00, 0x14, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: false,
		},
		{
			name:    "valid mp4 with moov",
			data:    []byte{0x00, 0x00, 0x00, 0x10, 'm', 'o', 'o', 'v', 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: false,
		},
		{
			name:    "invalid mp4",
			data:    []byte{0x00, 0x00, 0x00, 0x10, 'x', 'x', 'x', 'x', 0x00, 0x00, 0x00, 0x00},
			wantErr: true,
		},
		{
			name:    "too short",
			data:    []byte{0x00, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultMP4Validator()
			err := v.ValidateContent(bytes.NewReader(tt.data), int64(len(tt.data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMP3Validator_ValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid mp3 with ID3",
			data:    []byte{'I', 'D', '3', 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: false,
		},
		{
			name:    "valid mp3 frame sync",
			data:    []byte{0xFF, 0xFB, 0x90, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: false,
		},
		{
			name:    "invalid mp3",
			data:    []byte{0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultMP3Validator()
			err := v.ValidateContent(bytes.NewReader(tt.data), int64(len(tt.data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWebMValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "valid webm EBML header",
			// EBML magic: 0x1A 0x45 0xDF 0xA3
			data:    []byte{0x1A, 0x45, 0xDF, 0xA3, 0x00, 0x00, 0x00, 0x00},
			wantErr: false,
		},
		{
			name:    "invalid webm",
			data:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultWebMValidator()
			err := v.ValidateContent(bytes.NewReader(tt.data), int64(len(tt.data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWAVValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid wav RIFF/WAVE",
			data:    []byte{'R', 'I', 'F', 'F', 0x00, 0x00, 0x00, 0x00, 'W', 'A', 'V', 'E'},
			wantErr: false,
		},
		{
			name:    "invalid - RIFF but not WAVE",
			data:    []byte{'R', 'I', 'F', 'F', 0x00, 0x00, 0x00, 0x00, 'A', 'V', 'I', ' '},
			wantErr: true,
		},
		{
			name:    "invalid wav",
			data:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultWAVValidator()
			err := v.ValidateContent(bytes.NewReader(tt.data), int64(len(tt.data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestOggValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid ogg",
			data:    []byte{'O', 'g', 'g', 'S'},
			wantErr: false,
		},
		{
			name:    "invalid ogg",
			data:    []byte{'X', 'X', 'X', 'X'},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultOggValidator()
			err := v.ValidateContent(bytes.NewReader(tt.data), int64(len(tt.data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFLACValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid flac",
			data:    []byte{'f', 'L', 'a', 'C'},
			wantErr: false,
		},
		{
			name:    "invalid flac",
			data:    []byte{'F', 'L', 'A', 'C'}, // Case sensitive
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultFLACValidator()
			err := v.ValidateContent(bytes.NewReader(tt.data), int64(len(tt.data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAVIValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid avi",
			data:    []byte{'R', 'I', 'F', 'F', 0x00, 0x00, 0x00, 0x00, 'A', 'V', 'I', ' '},
			wantErr: false,
		},
		{
			name:    "invalid - RIFF but WAVE not AVI",
			data:    []byte{'R', 'I', 'F', 'F', 0x00, 0x00, 0x00, 0x00, 'W', 'A', 'V', 'E'},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultAVIValidator()
			err := v.ValidateContent(bytes.NewReader(tt.data), int64(len(tt.data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMKVValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid mkv EBML header",
			data:    []byte{0x1A, 0x45, 0xDF, 0xA3, 0x00, 0x00, 0x00, 0x00},
			wantErr: false,
		},
		{
			name:    "invalid mkv",
			data:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultMKVValidator()
			err := v.ValidateContent(bytes.NewReader(tt.data), int64(len(tt.data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAACValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid aac with ID3",
			data:    []byte{'I', 'D', '3', 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: false,
		},
		{
			name:    "valid aac ADTS sync",
			data:    []byte{0xFF, 0xF1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: false,
		},
		{
			name:    "valid aac ADIF",
			data:    []byte{'A', 'D', 'I', 'F', 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: false,
		},
		{
			name:    "invalid aac",
			data:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultAACValidator()
			err := v.ValidateContent(bytes.NewReader(tt.data), int64(len(tt.data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMediaValidators_SupportedMIMETypes(t *testing.T) {
	validators := []struct {
		name      string
		validator ContentValidator
		expected  []string
	}{
		{"MP4", DefaultMP4Validator(), []string{"video/mp4", "audio/mp4"}},
		{"MP3", DefaultMP3Validator(), []string{"audio/mpeg", "audio/mp3"}},
		{"WebM", DefaultWebMValidator(), []string{"video/webm", "audio/webm"}},
		{"WAV", DefaultWAVValidator(), []string{"audio/wav", "audio/x-wav"}},
		{"Ogg", DefaultOggValidator(), []string{"audio/ogg", "video/ogg"}},
		{"FLAC", DefaultFLACValidator(), []string{"audio/flac"}},
		{"AVI", DefaultAVIValidator(), []string{"video/x-msvideo"}},
		{"MKV", DefaultMKVValidator(), []string{"video/x-matroska"}},
		{"AAC", DefaultAACValidator(), []string{"audio/aac"}},
	}

	for _, tt := range validators {
		t.Run(tt.name, func(t *testing.T) {
			types := tt.validator.SupportedMIMETypes()
			for _, exp := range tt.expected {
				found := false
				for _, typ := range types {
					if typ == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected MIME type %s not found for %s", exp, tt.name)
				}
			}
		})
	}
}
