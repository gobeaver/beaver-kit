package config

import (
	"os"
	"testing"
	"time"
)

type TestConfig struct {
	Host     string   `env:"HOST" envDefault:"localhost"`
	Port     int      `env:"PORT" envDefault:"8080"`
	Debug    bool     `env:"DEBUG"`
	Tags     []string `env:"TAGS" envSeparator:","`
	Required string   `env:"REQUIRED"`
}

func TestLoad(t *testing.T) {
	// Clean up
	defer func() {
		os.Unsetenv("BEAVER_HOST")
		os.Unsetenv("BEAVER_PORT")
		os.Unsetenv("BEAVER_DEBUG")
		os.Unsetenv("BEAVER_TAGS")
	}()

	os.Setenv("BEAVER_HOST", "example.com")
	os.Setenv("BEAVER_PORT", "3000")
	os.Setenv("BEAVER_DEBUG", "true")
	os.Setenv("BEAVER_TAGS", "api,web,backend")

	var cfg TestConfig
	if err := Load(&cfg, WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Host != "example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "example.com")
	}
	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want %d", cfg.Port, 3000)
	}
	if !cfg.Debug {
		t.Error("Debug = false, want true")
	}
	if len(cfg.Tags) != 3 || cfg.Tags[0] != "api" {
		t.Errorf("Tags = %v, want [api web backend]", cfg.Tags)
	}
}

func TestLoadWithPrefix(t *testing.T) {
	defer os.Unsetenv("CUSTOM_HOST")

	os.Setenv("CUSTOM_HOST", "custom.example.com")

	var cfg TestConfig
	if err := Load(&cfg, WithPrefix("CUSTOM_"), WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Host != "custom.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "custom.example.com")
	}
}

func TestLoadWithNoPrefix(t *testing.T) {
	defer os.Unsetenv("HOST")

	os.Setenv("HOST", "noprefix.example.com")

	var cfg TestConfig
	if err := Load(&cfg, WithPrefix(""), WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Host != "noprefix.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "noprefix.example.com")
	}
}

func TestLoadDefaults(t *testing.T) {
	var cfg TestConfig
	if err := Load(&cfg, WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
}

func TestMustLoadPanics(t *testing.T) {
	type RequiredConfig struct {
		Value string `env:"MUST_EXIST,required"`
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoad should panic on missing required field")
		}
	}()

	var cfg RequiredConfig
	MustLoad(&cfg, WithoutDotEnv())
}

func TestLoadWithRequired(t *testing.T) {
	type RequiredFieldsConfig struct {
		APIKey   string `env:"API_KEY"`
		Optional string `env:"OPTIONAL" envDefault:"default"`
	}

	// Without WithRequired, missing fields without defaults are allowed
	var cfg1 RequiredFieldsConfig
	if err := Load(&cfg1, WithoutDotEnv()); err != nil {
		t.Errorf("Load without WithRequired should not fail: %v", err)
	}

	// With WithRequired, missing fields without defaults should fail
	var cfg2 RequiredFieldsConfig
	err := Load(&cfg2, WithRequired(), WithoutDotEnv())
	if err == nil {
		t.Error("Load with WithRequired should fail for missing API_KEY")
	}
}

func TestLoadDuration(t *testing.T) {
	type DurationConfig struct {
		Timeout time.Duration `env:"TIMEOUT" envDefault:"30s"`
	}

	defer os.Unsetenv("BEAVER_TIMEOUT")

	// Test default
	var cfg1 DurationConfig
	if err := Load(&cfg1, WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg1.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg1.Timeout)
	}

	// Test custom value
	os.Setenv("BEAVER_TIMEOUT", "1h30m")
	var cfg2 DurationConfig
	if err := Load(&cfg2, WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg2.Timeout != 90*time.Minute {
		t.Errorf("Timeout = %v, want 1h30m", cfg2.Timeout)
	}
}

func TestLoadSlices(t *testing.T) {
	type SliceConfig struct {
		Hosts []string `env:"HOSTS" envSeparator:","`
		Ports []int    `env:"PORTS" envSeparator:","`
	}

	defer func() {
		os.Unsetenv("BEAVER_HOSTS")
		os.Unsetenv("BEAVER_PORTS")
	}()

	os.Setenv("BEAVER_HOSTS", "host1,host2,host3")
	os.Setenv("BEAVER_PORTS", "8080,8081,8082")

	var cfg SliceConfig
	if err := Load(&cfg, WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Hosts) != 3 || cfg.Hosts[0] != "host1" || cfg.Hosts[2] != "host3" {
		t.Errorf("Hosts = %v, want [host1 host2 host3]", cfg.Hosts)
	}
	if len(cfg.Ports) != 3 || cfg.Ports[0] != 8080 || cfg.Ports[2] != 8082 {
		t.Errorf("Ports = %v, want [8080 8081 8082]", cfg.Ports)
	}
}

func TestLoadMaps(t *testing.T) {
	type MapConfig struct {
		Metadata map[string]string `env:"METADATA"`
	}

	defer os.Unsetenv("BEAVER_METADATA")

	os.Setenv("BEAVER_METADATA", "key1:val1,key2:val2")

	var cfg MapConfig
	if err := Load(&cfg, WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Metadata["key1"] != "val1" || cfg.Metadata["key2"] != "val2" {
		t.Errorf("Metadata = %v, want map[key1:val1 key2:val2]", cfg.Metadata)
	}
}

func TestLoadNestedStructs(t *testing.T) {
	type DatabaseConfig struct {
		Host string `env:"HOST" envDefault:"localhost"`
		Port int    `env:"PORT" envDefault:"5432"`
	}

	type AppConfig struct {
		Name     string         `env:"NAME" envDefault:"myapp"`
		Database DatabaseConfig `envPrefix:"DB_"`
	}

	defer func() {
		os.Unsetenv("BEAVER_NAME")
		os.Unsetenv("BEAVER_DB_HOST")
		os.Unsetenv("BEAVER_DB_PORT")
	}()

	os.Setenv("BEAVER_NAME", "testapp")
	os.Setenv("BEAVER_DB_HOST", "db.example.com")
	os.Setenv("BEAVER_DB_PORT", "3306")

	var cfg AppConfig
	if err := Load(&cfg, WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Name != "testapp" {
		t.Errorf("Name = %q, want %q", cfg.Name, "testapp")
	}
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 3306)
	}
}

func TestLoadFloat(t *testing.T) {
	type FloatConfig struct {
		Rate float64 `env:"RATE" envDefault:"0.5"`
	}

	defer os.Unsetenv("BEAVER_RATE")

	// Test default
	var cfg1 FloatConfig
	if err := Load(&cfg1, WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg1.Rate != 0.5 {
		t.Errorf("Rate = %v, want 0.5", cfg1.Rate)
	}

	// Test custom value
	os.Setenv("BEAVER_RATE", "3.14159")
	var cfg2 FloatConfig
	if err := Load(&cfg2, WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg2.Rate != 3.14159 {
		t.Errorf("Rate = %v, want 3.14159", cfg2.Rate)
	}
}

func TestLoadAllIntTypes(t *testing.T) {
	type IntConfig struct {
		Int8Val   int8   `env:"INT8" envDefault:"127"`
		Int16Val  int16  `env:"INT16" envDefault:"32767"`
		Int32Val  int32  `env:"INT32" envDefault:"2147483647"`
		Int64Val  int64  `env:"INT64" envDefault:"9223372036854775807"`
		Uint8Val  uint8  `env:"UINT8" envDefault:"255"`
		Uint16Val uint16 `env:"UINT16" envDefault:"65535"`
		Uint32Val uint32 `env:"UINT32" envDefault:"4294967295"`
		Uint64Val uint64 `env:"UINT64" envDefault:"18446744073709551615"`
	}

	var cfg IntConfig
	if err := Load(&cfg, WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Int8Val != 127 {
		t.Errorf("Int8Val = %d, want 127", cfg.Int8Val)
	}
	if cfg.Int64Val != 9223372036854775807 {
		t.Errorf("Int64Val = %d, want 9223372036854775807", cfg.Int64Val)
	}
	if cfg.Uint8Val != 255 {
		t.Errorf("Uint8Val = %d, want 255", cfg.Uint8Val)
	}
	if cfg.Uint64Val != 18446744073709551615 {
		t.Errorf("Uint64Val = %d, want 18446744073709551615", cfg.Uint64Val)
	}
}

func TestWithEnvFiles(t *testing.T) {
	// This test verifies that WithEnvFiles sets the option correctly
	// Actual file loading is tested via integration tests
	options := Options{
		Prefix:   DefaultPrefix,
		EnvFiles: []string{".env"},
	}

	opt := WithEnvFiles(".env.local", ".env.production")
	opt(&options)

	if len(options.EnvFiles) != 2 {
		t.Errorf("EnvFiles length = %d, want 2", len(options.EnvFiles))
	}
	if options.EnvFiles[0] != ".env.local" {
		t.Errorf("EnvFiles[0] = %q, want %q", options.EnvFiles[0], ".env.local")
	}
}
