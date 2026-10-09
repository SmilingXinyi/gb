package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// progressBarWidth is the number of cells in the bar drawn between brackets.
	progressBarWidth = 30
	// progressRefreshPeriod limits how often the progress line is redrawn.
	progressRefreshPeriod = 100 * time.Millisecond
	// unknownTotalBytes marks a transfer whose total size is not known in advance.
	unknownTotalBytes = int64(-1)
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
}

// newProgressReporter creates a reporter that measures elapsed time from the moment it is created.
func newProgressReporter(writer io.Writer, label string, totalBytes int64) *progressReporter {
	return &progressReporter{
		writer:     writer,
		label:      label,
		totalBytes: totalBytes,
		startedAt:  time.Now(),
		now:        time.Now,
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
func (reporter *progressReporter) render(now time.Time) {
	line := formatProgressLine(reporter.label, reporter.transferred, reporter.totalBytes, now.Sub(reporter.startedAt))
	lineLength := utf8.RuneCountInString(line)

	padding := ""
	if lineLength < reporter.lastLineLength {
		padding = strings.Repeat(" ", reporter.lastLineLength-lineLength)
	}
	// Progress output is best effort, so write errors are intentionally ignored.
	fmt.Fprintf(reporter.writer, "\r%s%s", line, padding)
	reporter.lastLineLength = lineLength
}

// formatProgressLine builds the text for one progress update, with a bar when the total is known.
func formatProgressLine(label string, transferred, totalBytes int64, elapsed time.Duration) string {
	speed := formatByteCount(bytesPerSecond(transferred, elapsed)) + "/s"
	if totalBytes == unknownTotalBytes {
		return fmt.Sprintf("%s  %s  %s", label, formatByteCount(transferred), speed)
	}

	percent := percentOf(transferred, totalBytes)
	return fmt.Sprintf("%s [%s] %3d%% %s / %s  %s",
		label,
		renderProgressBar(percent),
		percent,
		formatByteCount(transferred),
		formatByteCount(totalBytes),
		speed,
	)
}

// renderProgressBar draws a fixed-width bar where the leading arrow marks the current position.
func renderProgressBar(percent int64) string {
	filledCells := int(percent) * progressBarWidth / 100
	if filledCells >= progressBarWidth {
		return strings.Repeat("=", progressBarWidth)
	}
	return strings.Repeat("=", filledCells) + ">" + strings.Repeat(" ", progressBarWidth-filledCells-1)
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
