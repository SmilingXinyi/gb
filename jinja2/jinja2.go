// Package jinja2 renders Jinja2 templates through a thin wrapper around gonja.
// Callers pass ordinary Go maps and receive rendered strings or writer output.
package jinja2

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/loaders"
)

var (
	// engineMutex guards the package-level engine replaced by Setup.
	engineMutex sync.RWMutex
	// defaultEngine renders templates for the package-level helpers.
	defaultEngine = New(DefaultConfig())
)

// Engine renders Jinja2 templates with a fixed configuration.
type Engine struct {
	config Config
}

// Template is a parsed Jinja2 template that can be executed more than once.
type Template struct {
	compiled *exec.Template
}

// New creates an Engine from the given configuration.
func New(config Config) *Engine {
	return &Engine{config: config}
}

// Setup replaces the package-level engine used by Render, RenderFile, Parse, and ParseFile.
func Setup(config Config) {
	engineMutex.Lock()
	defaultEngine = New(config)
	engineMutex.Unlock()
}

// Render renders a template string with the package-level engine.
func Render(source string, data map[string]any) (string, error) {
	return currentEngine().Render(source, data)
}

// RenderFile renders a template file with the package-level engine.
func RenderFile(filePath string, data map[string]any) (string, error) {
	return currentEngine().RenderFile(filePath, data)
}

// Parse compiles a template string with the package-level engine.
func Parse(source string) (*Template, error) {
	return currentEngine().Parse(source)
}

// ParseFile compiles a template file with the package-level engine.
func ParseFile(filePath string) (*Template, error) {
	return currentEngine().ParseFile(filePath)
}

// Config returns a copy of the engine configuration.
func (engine *Engine) Config() Config {
	return engine.config
}

// Render compiles and executes a template string.
func (engine *Engine) Render(source string, data map[string]any) (string, error) {
	template, err := engine.Parse(source)
	if err != nil {
		return "", err
	}
	return template.Execute(data)
}

// RenderBytes compiles and executes template bytes.
func (engine *Engine) RenderBytes(source []byte, data map[string]any) (string, error) {
	template, err := engine.ParseBytes(source)
	if err != nil {
		return "", err
	}
	return template.Execute(data)
}

// RenderFile compiles and executes a template file.
func (engine *Engine) RenderFile(filePath string, data map[string]any) (string, error) {
	template, err := engine.ParseFile(filePath)
	if err != nil {
		return "", err
	}
	return template.Execute(data)
}

// Execute compiles a template string and writes the rendered result.
func (engine *Engine) Execute(writer io.Writer, source string, data map[string]any) error {
	template, err := engine.Parse(source)
	if err != nil {
		return err
	}
	return template.ExecuteWriter(writer, data)
}

// Parse compiles a template string.
func (engine *Engine) Parse(source string) (*Template, error) {
	return engine.ParseBytes([]byte(source))
}

// ParseBytes compiles template bytes.
func (engine *Engine) ParseBytes(source []byte) (*Template, error) {
	compiled, err := engine.compileBytes(source)
	if err != nil {
		return nil, err
	}
	return &Template{compiled: compiled}, nil
}

// ParseFile compiles a template file.
func (engine *Engine) ParseFile(filePath string) (*Template, error) {
	compiled, err := engine.compileFile(filePath)
	if err != nil {
		return nil, err
	}
	return &Template{compiled: compiled}, nil
}

// Execute renders the compiled template to a string.
func (template *Template) Execute(data map[string]any) (string, error) {
	rendered, err := template.compiled.ExecuteToString(newContext(data))
	if err != nil {
		return "", fmt.Errorf("jinja2: render template: %w", err)
	}
	return rendered, nil
}

// ExecuteBytes renders the compiled template to bytes.
func (template *Template) ExecuteBytes(data map[string]any) ([]byte, error) {
	rendered, err := template.compiled.ExecuteToBytes(newContext(data))
	if err != nil {
		return nil, fmt.Errorf("jinja2: render template: %w", err)
	}
	return rendered, nil
}

// ExecuteWriter renders the compiled template into writer.
func (template *Template) ExecuteWriter(writer io.Writer, data map[string]any) error {
	if err := template.compiled.Execute(writer, newContext(data)); err != nil {
		return fmt.Errorf("jinja2: render template: %w", err)
	}
	return nil
}

// compileBytes parses an in-memory template and keeps filesystem includes available.
func (engine *Engine) compileBytes(source []byte) (*exec.Template, error) {
	digest := sha256.Sum256(source)
	rootID := "root-" + hex.EncodeToString(digest[:])

	loader, err := loaders.NewFileSystemLoader(engine.config.SearchPath)
	if err != nil {
		return nil, fmt.Errorf("jinja2: create loader: %w", err)
	}
	shiftedLoader, err := loaders.NewShiftedLoader(rootID, bytes.NewReader(source), loader)
	if err != nil {
		return nil, fmt.Errorf("jinja2: create loader: %w", err)
	}

	compiled, err := exec.NewTemplate(rootID, engine.config.toGonja(), shiftedLoader, gonja.DefaultEnvironment)
	if err != nil {
		return nil, fmt.Errorf("jinja2: parse template: %w", err)
	}
	return compiled, nil
}

// compileFile parses a template stored on disk.
func (engine *Engine) compileFile(filePath string) (*exec.Template, error) {
	root := engine.config.SearchPath
	identifier := filePath
	if root == "" {
		root = filepath.Dir(filePath)
		identifier = filepath.Base(filePath)
	}

	loader, err := loaders.NewFileSystemLoader(root)
	if err != nil {
		return nil, fmt.Errorf("jinja2: create loader: %w", err)
	}
	compiled, err := exec.NewTemplate(identifier, engine.config.toGonja(), loader, gonja.DefaultEnvironment)
	if err != nil {
		return nil, fmt.Errorf("jinja2: parse template %q: %w", filePath, err)
	}
	return compiled, nil
}

// currentEngine returns the engine installed by Setup or the default engine.
func currentEngine() *Engine {
	engineMutex.RLock()
	engine := defaultEngine
	engineMutex.RUnlock()
	return engine
}

// newContext copies data into a gonja context.
// A nil map renders the template without extra variables.
func newContext(data map[string]any) *exec.Context {
	if data == nil {
		return nil
	}
	copied := make(map[string]any, len(data))
	for key, value := range data {
		copied[key] = value
	}
	return exec.NewContext(copied)
}
