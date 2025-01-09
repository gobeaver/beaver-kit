package config

import (
	"os"
	"reflect"
	"testing"
)

// Test struct with various field types
type TestConfig struct {
	StringField  string `env:"TEST_STRING"`
	IntField     int    `env:"TEST_INT"`
	Int64Field   int64  `env:"TEST_INT64"`
	BoolField    bool   `env:"TEST_BOOL"`
	DefaultField string `env:"TEST_DEFAULT,default:defaultValue"`
	NoTagField   string // Field without env tag
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected TestConfig
		wantErr  bool
	}{
		{
			name: "all fields set from environment",
			envVars: map[string]string{
				"TEST_STRING": "hello",
				"TEST_INT":    "42",
				"TEST_INT64":  "9223372036854775807",
				"TEST_BOOL":   "true",
			},
			expected: TestConfig{
				StringField:  "hello",
				IntField:     42,
				Int64Field:   9223372036854775807,
				BoolField:    true,
				DefaultField: "defaultValue",
			},
		},
		{
			name: "default values used when env not set",
			envVars: map[string]string{
				"TEST_STRING": "world",
				"TEST_INT":    "0",
				"TEST_INT64":  "0",
				"TEST_BOOL":   "false",
			},
			expected: TestConfig{
				StringField:  "world",
				IntField:     0,
				Int64Field:   0,
				BoolField:    false,
				DefaultField: "defaultValue",
			},
		},
		{
			name: "override default value",
			envVars: map[string]string{
				"TEST_STRING":  "test",
				"TEST_INT":     "123",
				"TEST_INT64":   "456",
				"TEST_BOOL":    "true",
				"TEST_DEFAULT": "overridden",
			},
			expected: TestConfig{
				StringField:  "test",
				IntField:     123,
				Int64Field:   456,
				BoolField:    true,
				DefaultField: "overridden",
			},
		},
		{
			name: "invalid int value",
			envVars: map[string]string{
				"TEST_INT": "not-a-number",
			},
			wantErr: true,
		},
		{
			name: "invalid bool value",
			envVars: map[string]string{
				"TEST_BOOL": "not-a-bool",
			},
			wantErr: true,
		},
		{
			name:    "empty environment leaves zero values",
			envVars: map[string]string{},
			expected: TestConfig{
				DefaultField: "defaultValue",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all test environment variables
			os.Unsetenv("TEST_STRING")
			os.Unsetenv("TEST_INT")
			os.Unsetenv("TEST_INT64")
			os.Unsetenv("TEST_BOOL")
			os.Unsetenv("TEST_DEFAULT")

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg := &TestConfig{}
			err := Load(cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if cfg.StringField != tt.expected.StringField {
					t.Errorf("StringField = %v, want %v", cfg.StringField, tt.expected.StringField)
				}
				if cfg.IntField != tt.expected.IntField {
					t.Errorf("IntField = %v, want %v", cfg.IntField, tt.expected.IntField)
				}
				if cfg.Int64Field != tt.expected.Int64Field {
					t.Errorf("Int64Field = %v, want %v", cfg.Int64Field, tt.expected.Int64Field)
				}
				if cfg.BoolField != tt.expected.BoolField {
					t.Errorf("BoolField = %v, want %v", cfg.BoolField, tt.expected.BoolField)
				}
				if cfg.DefaultField != tt.expected.DefaultField {
					t.Errorf("DefaultField = %v, want %v", cfg.DefaultField, tt.expected.DefaultField)
				}
			}
		})
	}
}

func TestLoadWithDebug(t *testing.T) {
	// Test debug output
	os.Setenv("BEAVER_CONFIG_DEBUG", "true")
	os.Setenv("TEST_STRING", "debug-test")
	defer os.Unsetenv("BEAVER_CONFIG_DEBUG")
	defer os.Unsetenv("TEST_STRING")

	cfg := &TestConfig{}
	err := Load(cfg)
	if err != nil {
		t.Errorf("Load() with debug enabled failed: %v", err)
	}

	if cfg.StringField != "debug-test" {
		t.Errorf("StringField = %v, want %v", cfg.StringField, "debug-test")
	}
}

func TestSetFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		fieldType string
		value     string
		wantErr   bool
	}{
		{
			name:      "valid string",
			fieldType: "string",
			value:     "test",
		},
		{
			name:      "valid int",
			fieldType: "int",
			value:     "123",
		},
		{
			name:      "valid int64",
			fieldType: "int64",
			value:     "9223372036854775807",
		},
		{
			name:      "valid bool true",
			fieldType: "bool",
			value:     "true",
		},
		{
			name:      "valid bool false",
			fieldType: "bool",
			value:     "false",
		},
		{
			name:      "valid bool 1",
			fieldType: "bool",
			value:     "1",
		},
		{
			name:      "valid bool 0",
			fieldType: "bool",
			value:     "0",
		},
		{
			name:      "invalid int",
			fieldType: "int",
			value:     "abc",
			wantErr:   true,
		},
		{
			name:      "invalid bool",
			fieldType: "bool",
			value:     "yes",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg interface{}
			switch tt.fieldType {
			case "string":
				cfg = &struct{ Field string }{}
			case "int":
				cfg = &struct{ Field int }{}
			case "int64":
				cfg = &struct{ Field int64 }{}
			case "bool":
				cfg = &struct{ Field bool }{}
			}

			v := reflect.ValueOf(cfg).Elem()
			field := v.Field(0)

			err := setFieldValue(field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("setFieldValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestComplexEnvTag(t *testing.T) {
	type ComplexConfig struct {
		Field1 string `env:"COMPLEX_FIELD1,default:value1"`
		Field2 string `env:"COMPLEX_FIELD2,default:value2,other:ignored"`
		Field3 string `env:"COMPLEX_FIELD3,something,default:value3"`
	}

	cfg := &ComplexConfig{}
	err := Load(cfg)
	if err != nil {
		t.Errorf("Load() failed: %v", err)
	}

	// Check that default values are properly parsed
	if cfg.Field1 != "value1" {
		t.Errorf("Field1 = %v, want %v", cfg.Field1, "value1")
	}
	if cfg.Field2 != "value2" {
		t.Errorf("Field2 = %v, want %v", cfg.Field2, "value2")
	}
	if cfg.Field3 != "value3" {
		t.Errorf("Field3 = %v, want %v", cfg.Field3, "value3")
	}
}

func TestUnsupportedFieldType(t *testing.T) {
	type UnsupportedConfig struct {
		FloatField float64 `env:"TEST_FLOAT"`
	}

	os.Setenv("TEST_FLOAT", "3.14")
	defer os.Unsetenv("TEST_FLOAT")

	cfg := &UnsupportedConfig{}
	err := Load(cfg)
	if err != nil {
		t.Errorf("Load() should not error for unsupported types, got: %v", err)
	}

	// Field should remain at zero value since float64 is not supported
	if cfg.FloatField != 0 {
		t.Errorf("FloatField = %v, want %v", cfg.FloatField, 0)
	}
}
