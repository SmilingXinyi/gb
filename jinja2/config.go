package jinja2

import gonjaconfig "github.com/nikolalohinski/gonja/v2/config"

// Config holds Jinja2 engine settings.
// Empty delimiter strings keep gonja defaults so a partial Config stays valid.
type Config struct {
	// SearchPath is the directory used to resolve template files and includes.
	// An empty SearchPath resolves includes from the current working directory,
	// and file templates from the directory that contains the template file.
	SearchPath string
	// BlockStartString marks the start of a statement block. Defaults to "{%".
	BlockStartString string
	// BlockEndString marks the end of a statement block. Defaults to "%}".
	BlockEndString string
	// VariableStartString marks the start of a print expression. Defaults to "{{".
	VariableStartString string
	// VariableEndString marks the end of a print expression. Defaults to "}}".
	VariableEndString string
	// CommentStartString marks the start of a comment. Defaults to "{#".
	CommentStartString string
	// CommentEndString marks the end of a comment. Defaults to "#}".
	CommentEndString string
	// AutoEscape enables HTML autoescaping for printed values.
	AutoEscape bool
	// StrictUndefined returns an error when a variable or attribute is missing.
	StrictUndefined bool
	// TrimBlocks removes the first newline after a statement block.
	TrimBlocks bool
	// LeftStripBlocks strips leading spaces and tabs before a statement block.
	LeftStripBlocks bool
	// KeepTrailingNewline preserves a single trailing newline at the end of the template.
	KeepTrailingNewline bool
	// NewlineSequence is written for rendered newlines. Defaults to "\n".
	NewlineSequence string
	// LineStatementPrefix turns a whole line into a statement when set.
	LineStatementPrefix string
	// LineCommentPrefix turns the rest of a line into a comment when set.
	LineCommentPrefix string
}

// DefaultConfig returns gonja's default Jinja2 settings.
func DefaultConfig() Config {
	defaults := gonjaconfig.New()
	return Config{
		BlockStartString:    defaults.BlockStartString,
		BlockEndString:      defaults.BlockEndString,
		VariableStartString: defaults.VariableStartString,
		VariableEndString:   defaults.VariableEndString,
		CommentStartString:  defaults.CommentStartString,
		CommentEndString:    defaults.CommentEndString,
		AutoEscape:          defaults.AutoEscape,
		StrictUndefined:     defaults.StrictUndefined,
		TrimBlocks:          defaults.TrimBlocks,
		LeftStripBlocks:     defaults.LeftStripBlocks,
		KeepTrailingNewline: defaults.KeepTrailingNewline,
		NewlineSequence:     defaults.NewlineSequence,
		LineStatementPrefix: defaults.LineStatementPrefix,
		LineCommentPrefix:   defaults.LineCommentPrefix,
	}
}

// toGonja converts Config into the gonja parser configuration.
// Empty delimiter fields fall back to gonja defaults.
func (config Config) toGonja() *gonjaconfig.Config {
	merged := gonjaconfig.New()
	if config.BlockStartString != "" {
		merged.BlockStartString = config.BlockStartString
	}
	if config.BlockEndString != "" {
		merged.BlockEndString = config.BlockEndString
	}
	if config.VariableStartString != "" {
		merged.VariableStartString = config.VariableStartString
	}
	if config.VariableEndString != "" {
		merged.VariableEndString = config.VariableEndString
	}
	if config.CommentStartString != "" {
		merged.CommentStartString = config.CommentStartString
	}
	if config.CommentEndString != "" {
		merged.CommentEndString = config.CommentEndString
	}
	if config.NewlineSequence != "" {
		merged.NewlineSequence = config.NewlineSequence
	}
	merged.AutoEscape = config.AutoEscape
	merged.StrictUndefined = config.StrictUndefined
	merged.TrimBlocks = config.TrimBlocks
	merged.LeftStripBlocks = config.LeftStripBlocks
	merged.KeepTrailingNewline = config.KeepTrailingNewline
	merged.LineStatementPrefix = config.LineStatementPrefix
	merged.LineCommentPrefix = config.LineCommentPrefix
	return merged
}
