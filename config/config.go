package config

import (
	"fmt"
	"github.com/gobeaver/beaver-kit/config/dotenv"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Load populates a struct from .env file and environment variables
func Load(cfg interface{}) error {
	// Silently try to load .env file, ignore if not found
	dotenv.Load()

	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()
	printDebug := os.Getenv("BEAVER_CONFIG_DEBUG") == "true"

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		envTag := field.Tag.Get("env")
		if envTag == "" {
			continue
		}

		parts := strings.Split(envTag, ",")
		envName := parts[0]
		defaultValue := ""

		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "default:") {
				defaultValue = strings.TrimPrefix(part, "default:")
				break
			}
		}

		value := os.Getenv(envName)
		if value == "" {
			value = defaultValue
		}
		if printDebug || os.Getenv("env") == "development" || os.Getenv("env") == "test" || os.Getenv("env") == "dev" {
			fmt.Printf("[BEAVER] %s=%s\n", envName, value)
		}

		if value != "" {
			if err := setFieldValue(v.Field(i), value); err != nil {
				return err
			}
		}
	}

	return nil
}

func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)
	}
	return nil
}
