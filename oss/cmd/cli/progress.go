package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	// progressBarWidth is the number of cells in the bar drawn between brackets on wide terminals.
	progressBarWidth = 30
	// minimumBarWidth is the smallest bar kept on narrow terminals before the label is shortened further.
	minimumBarWidth = 10
	// minimumLabelWidth is the label width the bar gives way to before the bar shrinks any further.
	minimumLabelWidth = 20
	// progressRefreshPeriod limits how often the progress line is redrawn.
	progressRefreshPeriod = 100 * time.Millisecond
	// unknownTotalBytes marks a transfer whose total size is not known in advance.
	unknownTotalBytes = int64(-1)
	// fallbackTerminalColumns is used when the terminal width cannot be read.
	fallbackTerminalColumns = 80
	// minimumKeyWidth is the smallest object key width that is still shown after the action word.
	minimumKeyWidth = 4
	// ellipsis marks the cut inside a label that was shortened to fit the terminal.
	ellipsis = "..."
)

// progressReporter renders a single updating progress line on a writer.
// A nil reporter is valid and ignores every call, so callers can skip progress without branching.
type progressReporter struct {
	writer         io.Writer
	label          string
	totalBytes     int64
	transferred    int64
	startedAt      time.Time
	lastRenderedAt time.Time
	lastLineLength int
	now            func() time.Time
	columns        func() int
}

// newProgressReporter creates a reporter that draws on the terminal and measures elapsed time from now.
func newProgressReporter(terminal *os.File, label string, totalBytes int64) *progressReporter {
	return &progressReporter{
		writer:     terminal,
		label:      label,
		totalBytes: totalBytes,
		startedAt:  time.Now(),
		now:        time.Now,
		columns: func() int {
			return terminalColumns(terminal)
		},
	}
}

// advance adds the given number of transferred bytes and redraws the line when due.
func (reporter *progressReporter) advance(delta int64) {
	if reporter == nil {
		return
	}
	reporter.transferred += delta
	reporter.renderIfDue()
}

// setPosition moves the transferred counter to an absolute offset after the underlying reader seeks.
func (reporter *progressReporter) setPosition(offset int64) {
	if reporter == nil {
		return
	}
	reporter.transferred = offset
	reporter.renderIfDue()
}

// finish draws the final state of the line and moves the cursor to the next line.
func (reporter *progressReporter) finish() {
	if reporter == nil {
		return
	}
	reporter.render(reporter.now())
	fmt.Fprintln(reporter.writer)
}

// renderIfDue redraws the line when the refresh period has elapsed or the transfer is complete.
func (reporter *progressReporter) renderIfDue() {
	now := reporter.now()
	isComplete := reporter.totalBytes != unknownTotalBytes && reporter.transferred >= reporter.totalBytes
	if !isComplete && now.Sub(reporter.lastRenderedAt) < progressRefreshPeriod {
		return
	}
	reporter.lastRenderedAt = now
	reporter.render(now)
}

// render writes the current state over the previous line using a carriage return.
// The line is kept shorter than the terminal, because a wrapped line cannot be overwritten by a carriage return.
func (reporter *progressReporter) render(now time.Time) {
	// One column is left free so the cursor never reaches the right edge and triggers an automatic wrap.
	maxWidth := reporter.columns() - 1
	line := formatProgressLine(reporter.label, reporter.transferred, reporter.totalBytes, now.Sub(reporter.startedAt), maxWidth)
	lineLength := utf8.RuneCountInString(line)

	padding := ""
	if lineLength < reporter.lastLineLength {
		padding = strings.Repeat(" ", reporter.lastLineLength-lineLength)
	}
	// Progress output is best effort, so write errors are intentionally ignored.
	fmt.Fprintf(reporter.writer, "\r%s%s", line, padding)
	reporter.lastLineLength = lineLength
}

// formatProgressLine builds the text for one progress update, never exceeding maxWidth runes.
// The label is shortened first, so the bar and the counters stay visible on narrow terminals.
func formatProgressLine(label string, transferred, totalBytes int64, elapsed time.Duration, maxWidth int) string {
	barWidth := progressBarWidth
	if totalBytes != unknownTotalBytes {
		fixedWidth := utf8.RuneCountInString(formatProgressDetails(transferred, totalBytes, elapsed, 0))
		barWidth = min(progressBarWidth, max(minimumBarWidth, maxWidth-fixedWidth-minimumLabelWidth))
	}
	details := formatProgressDetails(transferred, totalBytes, elapsed, barWidth)

	labelBudget := maxWidth - utf8.RuneCountInString(details)
	line := []rune(shortenLabel(label, labelBudget) + details)
	if maxWidth >= 0 && len(line) > maxWidth {
		line = line[:maxWidth]
	}
	return string(line)
}

// shortenLabel fits the label into width runes, keeping the action word such as "download" and cutting the object key.
func shortenLabel(label string, width int) string {
	if utf8.RuneCountInString(label) <= width {
		return label
	}
	action, key, hasKey := strings.Cut(label, " ")
	if !hasKey {
		return truncateMiddle(label, width)
	}
	keyWidth := width - utf8.RuneCountInString(action) - 1
	if keyWidth < minimumKeyWidth {
		return truncateMiddle(label, width)
	}
	return action + " " + truncateMiddle(key, keyWidth)
}

// formatProgressDetails builds the part of the line that follows the label: bar, counters and speed.
func formatProgressDetails(transferred, totalBytes int64, elapsed time.Duration, barWidth int) string {
	speed := formatByteCount(bytesPerSecond(transferred, elapsed)) + "/s"
	if totalBytes == unknownTotalBytes {
		return fmt.Sprintf("  %s  %s", formatByteCount(transferred), speed)
	}

	percent := percentOf(transferred, totalBytes)
	return fmt.Sprintf(" [%s] %3d%% %s / %s  %s",
		renderProgressBar(percent, barWidth),
		percent,
		formatByteCount(transferred),
		formatByteCount(totalBytes),
		speed,
	)
}

// truncateMiddle shortens text to width runes by keeping its start and end around an ellipsis.
func truncateMiddle(text string, width int) string {
	textRunes := []rune(text)
	if width <= 0 {
		return ""
	}
	if len(textRunes) <= width {
		return text
	}
	ellipsisLength := len(ellipsis)
	if width <= ellipsisLength {
		return string(textRunes[:width])
	}
	keptLength := width - ellipsisLength
	headLength := (keptLength + 1) / 2
	tailLength := keptLength - headLength
	return string(textRunes[:headLength]) + ellipsis + string(textRunes[len(textRunes)-tailLength:])
}

// renderProgressBar draws a bar of the given width where the leading arrow marks the current position.
func renderProgressBar(percent int64, barWidth int) string {
	filledCells := int(percent) * barWidth / 100
	if filledCells >= barWidth {
		return strings.Repeat("=", barWidth)
	}
	return strings.Repeat("=", filledCells) + ">" + strings.Repeat(" ", barWidth-filledCells-1)
}

// percentOf returns the completion percentage clamped to [0, 100]. An empty transfer counts as complete.
func percentOf(transferred, totalBytes int64) int64 {
	if totalBytes <= 0 {
		return 100
	}
	percent := transferred * 100 / totalBytes
	if percent > 100 {
		return 100
	}
	return percent
}

// bytesPerSecond returns the average transfer rate over the elapsed duration.
func bytesPerSecond(transferred int64, elapsed time.Duration) int64 {
	if elapsed <= 0 {
		return 0
	}
	return int64(float64(transferred) / elapsed.Seconds())
}

// formatByteCount converts a byte count into a human-readable binary unit string.
func formatByteCount(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(bytes)
	unitIndex := -1
	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}
	return fmt.Sprintf("%.1f %s", value, units[unitIndex])
}

// terminalColumns returns the width of the terminal attached to the file, or a fallback when it cannot be read.
func terminalColumns(terminal *os.File) int {
	width, _, err := term.GetSize(int(terminal.Fd()))
	if err != nil || width <= 0 {
		return fallbackTerminalColumns
	}
	return width
}

// progressReader reports every byte read through it to a progress reporter.
type progressReader struct {
	reader   io.Reader
	reporter *progressReporter
}

// Read delegates to the wrapped reader and records how many bytes were returned.
func (wrapper *progressReader) Read(buffer []byte) (int, error) {
	count, err := wrapper.reader.Read(buffer)
	wrapper.reporter.advance(int64(count))
	return count, err
}

// progressReadSeeker reports reads and keeps the counter aligned when an SDK seeks back to retry an upload.
type progressReadSeeker struct {
	reader   io.ReadSeeker
	reporter *progressReporter
}

// Read delegates to the wrapped reader and records how many bytes were returned.
func (wrapper *progressReadSeeker) Read(buffer []byte) (int, error) {
	count, err := wrapper.reader.Read(buffer)
	wrapper.reporter.advance(int64(count))
	return count, err
}

// Seek delegates to the wrapped reader and resets the progress counter to the new absolute position.
func (wrapper *progressReadSeeker) Seek(offset int64, whence int) (int64, error) {
	position, err := wrapper.reader.Seek(offset, whence)
	if err != nil {
		return position, err
	}
	wrapper.reporter.setPosition(position)
	return position, nil
}

// shouldShowProgress reports whether a progress bar should be drawn, which requires an interactive stderr.
func shouldShowProgress(disabled bool) bool {
	return !disabled && isTerminal(os.Stderr)
}

// isTerminal reports whether the file is a character device such as a TTY.
func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
