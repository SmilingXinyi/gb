package jinja2

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	rendered, err := Render("Hello {{ name | capitalize }}!", map[string]any{
		"name": "bob",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello Bob!", rendered)
}

func TestRenderNilData(t *testing.T) {
	rendered, err := Render("plain", nil)
	require.NoError(t, err)
	assert.Equal(t, "plain", rendered)
}

func TestRenderLoopAndComment(t *testing.T) {
	source := "{% for item in items %}{{ item }}{% if not loop.last %},{% endif %}{% endfor %}{# hidden #}"
	rendered, err := Render(source, map[string]any{
		"items": []string{"a", "b", "c"},
	})
	require.NoError(t, err)
	assert.Equal(t, "a,b,c", rendered)
}

func TestRenderNestedMap(t *testing.T) {
	rendered, err := Render("{{ user.name }}", map[string]any{
		"user": map[string]any{"name": "Ada"},
	})
	require.NoError(t, err)
	assert.Equal(t, "Ada", rendered)
}

func TestRenderSyntaxError(t *testing.T) {
	_, err := Render("{{ ", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jinja2: parse template")
}

func TestParseReuse(t *testing.T) {
	template, err := Parse("{{ name }}")
	require.NoError(t, err)

	first, err := template.Execute(map[string]any{"name": "first"})
	require.NoError(t, err)
	second, err := template.Execute(map[string]any{"name": "second"})
	require.NoError(t, err)

	assert.Equal(t, "first", first)
	assert.Equal(t, "second", second)
}

func TestExecuteWriterAndBytes(t *testing.T) {
	template, err := Parse("{{ value }}")
	require.NoError(t, err)

	var buffer bytes.Buffer
	err = template.ExecuteWriter(&buffer, map[string]any{"value": "writer"})
	require.NoError(t, err)
	assert.Equal(t, "writer", buffer.String())

	rendered, err := template.ExecuteBytes(map[string]any{"value": "bytes"})
	require.NoError(t, err)
	assert.Equal(t, []byte("bytes"), rendered)
}

func TestEngineExecute(t *testing.T) {
	engine := New(DefaultConfig())
	var buffer bytes.Buffer
	err := engine.Execute(&buffer, "n={{ count }}", map[string]any{"count": 3})
	require.NoError(t, err)
	assert.Equal(t, "n=3", buffer.String())
}

func TestStrictUndefined(t *testing.T) {
	config := DefaultConfig()
	config.StrictUndefined = true
	engine := New(config)

	_, err := engine.Render("{{ missing }}", map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jinja2: render template")
}

func TestAutoEscape(t *testing.T) {
	config := DefaultConfig()
	config.AutoEscape = true
	engine := New(config)

	rendered, err := engine.Render("{{ value }}", map[string]any{"value": "<b>"})
	require.NoError(t, err)
	assert.Equal(t, "&lt;b&gt;", rendered)
}

func TestTrimBlocks(t *testing.T) {
	source := "{% if ready %}\nyes\n{% endif %}\n"
	data := map[string]any{"ready": true}

	plain, err := Render(source, data)
	require.NoError(t, err)

	config := DefaultConfig()
	config.TrimBlocks = true
	trimmed, err := New(config).Render(source, data)
	require.NoError(t, err)

	assert.NotEqual(t, plain, trimmed)
	assert.Contains(t, trimmed, "yes")
}

func TestCustomDelimiters(t *testing.T) {
	config := DefaultConfig()
	config.VariableStartString = "[["
	config.VariableEndString = "]]"
	engine := New(config)

	rendered, err := engine.Render("Hello [[ name ]]", map[string]any{"name": "Lin"})
	require.NoError(t, err)
	assert.Equal(t, "Hello Lin", rendered)
}

func TestRenderFileAndInclude(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "partial.j2"), []byte("{{ name }}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "page.j2"), []byte("Hello {% include \"partial.j2\" %}"), 0o644))

	config := DefaultConfig()
	config.SearchPath = directory
	engine := New(config)

	rendered, err := engine.RenderFile("page.j2", map[string]any{"name": "world"})
	require.NoError(t, err)
	assert.Equal(t, "Hello world", rendered)
}

func TestRenderFileWithoutSearchPath(t *testing.T) {
	directory := t.TempDir()
	filePath := filepath.Join(directory, "hello.j2")
	require.NoError(t, os.WriteFile(filePath, []byte("Hi {{ name }}"), 0o644))

	rendered, err := RenderFile(filePath, map[string]any{"name": "file"})
	require.NoError(t, err)
	assert.Equal(t, "Hi file", rendered)
}

func TestRenderFileMissing(t *testing.T) {
	_, err := RenderFile(filepath.Join(t.TempDir(), "missing.j2"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jinja2: parse template")
}

func TestStringIncludeFromSearchPath(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "snippet.j2"), []byte("snippet"), 0o644))

	config := DefaultConfig()
	config.SearchPath = directory
	engine := New(config)

	rendered, err := engine.Render("{% include 'snippet.j2' %}", nil)
	require.NoError(t, err)
	assert.Equal(t, "snippet", rendered)
}

func TestSetup(t *testing.T) {
	t.Cleanup(func() {
		Setup(DefaultConfig())
	})

	config := DefaultConfig()
	config.VariableStartString = "[["
	config.VariableEndString = "]]"
	Setup(config)

	rendered, err := Render("[[ name ]]", map[string]any{"name": "setup"})
	require.NoError(t, err)
	assert.Equal(t, "setup", rendered)
	assert.Equal(t, "[[", currentEngine().Config().VariableStartString)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, "{{", config.VariableStartString)
	assert.Equal(t, "}}", config.VariableEndString)
	assert.Equal(t, "{%", config.BlockStartString)
	assert.Equal(t, "%}", config.BlockEndString)
	assert.Equal(t, "{#", config.CommentStartString)
	assert.Equal(t, "#}", config.CommentEndString)
	assert.Equal(t, "\n", config.NewlineSequence)
	assert.False(t, config.StrictUndefined)
	assert.False(t, config.AutoEscape)
}

func TestPartialConfigKeepsDefaultDelimiters(t *testing.T) {
	engine := New(Config{StrictUndefined: true})
	rendered, err := engine.Render("{{ name }}", map[string]any{"name": "ok"})
	require.NoError(t, err)
	assert.Equal(t, "ok", rendered)
}

func TestRenderBytes(t *testing.T) {
	engine := New(DefaultConfig())
	rendered, err := engine.RenderBytes([]byte("{{ name }}"), map[string]any{"name": "bytes"})
	require.NoError(t, err)
	assert.Equal(t, "bytes", rendered)
}

func TestInvalidSearchPath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o644))

	engine := New(Config{SearchPath: filePath})
	_, err := engine.Render("{{ name }}", map[string]any{"name": "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jinja2: create loader")
}
