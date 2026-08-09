// Package config loads YAML configuration and applies explicit environment overrides.
package config

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// options holds Load customization settings.
type options struct {
	dotEnvFiles []dotEnvFile
}

// dotEnvFile describes a dotenv file that should be loaded before YAML overrides.
type dotEnvFile struct {
	path     string
	optional bool
}

// Option customizes Load behavior.
type Option func(*options)

// WithDotEnv loads the provided dotenv files. Missing files are errors.
func WithDotEnv(paths ...string) Option {
	return func(settings *options) {
		for _, path := range paths {
			settings.dotEnvFiles = append(settings.dotEnvFiles, dotEnvFile{path: path})
		}
	}
}

// WithOptionalDotEnv loads dotenv files when they exist.
func WithOptionalDotEnv(paths ...string) Option {
	return func(settings *options) {
		for _, path := range paths {
			settings.dotEnvFiles = append(settings.dotEnvFiles, dotEnvFile{path: path, optional: true})
		}
	}
}

// Load decodes a YAML file, loads dotenv files, and applies special environment
// overrides such as _SERVER__PORT to the YAML path server.port.
func Load(path string, destination any, optionValues ...Option) error {
	if err := validateDestination(destination); err != nil {
		return err
	}
	var settings options
	for _, option := range optionValues {
		if option != nil {
			option(&settings)
		}
	}
	for _, file := range settings.dotEnvFiles {
		if err := godotenv.Load(file.path); err != nil {
			if file.optional && errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("load dotenv file %q: %w", file.path, err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %q: %w", path, err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode config file %q: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode config file %q: multiple YAML documents are not allowed", path)
		}
		return fmt.Errorf("decode config file %q: %w", path, err)
	}
	if err := ApplyEnv(destination, os.Environ()); err != nil {
		return fmt.Errorf("apply environment overrides: %w", err)
	}
	return nil
}

// ApplyEnv applies entries with the _SECTION__FIELD naming convention.
// A single segment such as _NAME addresses a top-level YAML field.
func ApplyEnv(destination any, environment []string) error {
	if err := validateDestination(destination); err != nil {
		return err
	}
	root := reflect.ValueOf(destination).Elem()
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		path, ok := environmentPath(name)
		if !ok {
			continue
		}
		if !hasTopLevelField(root, path[0]) {
			continue
		}
		if err := setPath(root, path, value); err != nil {
			return fmt.Errorf("override %s (%s): %w", name, strings.Join(path, "."), err)
		}
	}
	return nil
}

// hasTopLevelField reports whether the destination struct has a matching YAML field.
func hasTopLevelField(root reflect.Value, name string) bool {
	root = indirect(root)
	if root.Kind() != reflect.Struct {
		return false
	}
	_, ok := fieldByYAMLName(root, name)
	return ok
}

// validateDestination ensures destination is a non-nil pointer.
func validateDestination(destination any) error {
	if destination == nil {
		return fmt.Errorf("config destination is required")
	}
	value := reflect.ValueOf(destination)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("config destination must be a non-nil pointer")
	}
	return nil
}

// environmentPath converts _SERVER__PORT into ["server", "port"].
func environmentPath(name string) ([]string, bool) {
	if len(name) < 2 || name[0] != '_' || name[1] == '_' {
		return nil, false
	}
	segments := strings.Split(name[1:], "__")
	for index, segment := range segments {
		if segment == "" || !validEnvironmentSegment(segment) {
			return nil, false
		}
		segments[index] = strings.ToLower(segment)
	}
	return segments, true
}

// validEnvironmentSegment reports whether a segment uses only letters, digits, or underscores.
func validEnvironmentSegment(segment string) bool {
	for _, character := range segment {
		if character != '_' && !unicode.IsLetter(character) && !unicode.IsDigit(character) {
			return false
		}
	}
	return true
}

// setPath writes rawValue into the nested field addressed by path.
func setPath(root reflect.Value, path []string, rawValue string) error {
	current := indirect(root)
	for _, segment := range path[:len(path)-1] {
		if current.Kind() != reflect.Struct {
			return fmt.Errorf("%q is not a nested configuration object", segment)
		}
		field, ok := fieldByYAMLName(current, segment)
		if !ok {
			return fmt.Errorf("unknown configuration field %q", segment)
		}
		current = indirect(field)
	}
	if current.Kind() != reflect.Struct {
		return fmt.Errorf("%q is not a configuration object", strings.Join(path[:len(path)-1], "."))
	}
	field, ok := fieldByYAMLName(current, path[len(path)-1])
	if !ok {
		return fmt.Errorf("unknown configuration field %q", path[len(path)-1])
	}
	return setValue(field, rawValue)
}

// indirect dereferences pointers and allocates nil pointer intermediates.
func indirect(value reflect.Value) reflect.Value {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()
	}
	return value
}

// fieldByYAMLName looks up an exported struct field by its yaml tag or lowercased name.
func fieldByYAMLName(value reflect.Value, name string) (reflect.Value, bool) {
	typeValue := value.Type()
	for index := 0; index < value.NumField(); index++ {
		fieldType := typeValue.Field(index)
		if fieldType.PkgPath != "" {
			continue
		}
		fieldName := strings.Split(fieldType.Tag.Get("yaml"), ",")[0]
		if fieldName == "-" {
			continue
		}
		if fieldName == "" {
			fieldName = strings.ToLower(fieldType.Name)
		}
		if fieldName == name {
			return value.Field(index), true
		}
	}
	return reflect.Value{}, false
}

// setValue parses rawValue and assigns it to value.
func setValue(value reflect.Value, rawValue string) error {
	value = indirect(value)
	if !value.CanSet() {
		return fmt.Errorf("configuration field cannot be set")
	}
	if value.CanAddr() {
		if unmarshaler, ok := value.Addr().Interface().(encoding.TextUnmarshaler); ok {
			if err := unmarshaler.UnmarshalText([]byte(rawValue)); err != nil {
				return err
			}
			return nil
		}
	}

	switch value.Kind() {
	case reflect.String:
		value.SetString(rawValue)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(rawValue)
		if err != nil {
			return fmt.Errorf("parse %q as boolean: %w", rawValue, err)
		}
		value.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(rawValue, 10, value.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse %q as integer: %w", rawValue, err)
		}
		value.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(rawValue, 10, value.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse %q as unsigned integer: %w", rawValue, err)
		}
		value.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(rawValue, value.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse %q as number: %w", rawValue, err)
		}
		value.SetFloat(parsed)
	default:
		target := reflect.New(value.Type())
		if err := yaml.Unmarshal([]byte(rawValue), target.Interface()); err != nil {
			return fmt.Errorf("decode %q: %w", rawValue, err)
		}
		value.Set(target.Elem())
	}
	return nil
}

// Duration is a YAML and environment compatible time.Duration value.
type Duration time.Duration

// UnmarshalText parses a Go duration string such as "5s" or "1h30m".
func (duration *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", text, err)
	}
	*duration = Duration(parsed)
	return nil
}

// UnmarshalYAML parses a scalar YAML duration using UnmarshalText.
func (duration *Duration) UnmarshalYAML(node *yaml.Node) error {
	return duration.UnmarshalText([]byte(node.Value))
}

// Value returns the underlying time.Duration.
func (duration Duration) Value() time.Duration {
	return time.Duration(duration)
}
