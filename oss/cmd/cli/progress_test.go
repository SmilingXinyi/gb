package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wideTerminalColumns is a width large enough that no test line is shortened.
const wideTerminalColumns = 200

// newTestReporter returns a reporter whose clock is controlled by the caller through currentTime.
func newTestReporter(buffer *bytes.Buffer, label string, totalBytes int64, currentTime *time.Time) *progressReporter {
	startTime := *currentTime
	return &progressReporter{
		writer:     buffer,
		label:      label,
		totalBytes: totalBytes,
		startedAt:  startTime,
		now: func() time.Time {
			return *currentTime
		},
		columns: func() int {
			return wideTerminalColumns
		},
	}
}

// TestFormatByteCount converts byte counts into binary units.
func TestFormatByteCount(t *testing.T) {
	assert.Equal(t, "0 B", formatByteCount(0))
	assert.Equal(t, "512 B", formatByteCount(512))
	assert.Equal(t, "1.0 KiB", formatByteCount(1024))
	assert.Equal(t, "1.5 KiB", formatByteCount(1536))
	assert.Equal(t, "1.0 MiB", formatByteCount(1024*1024))
	assert.Equal(t, "3.0 GiB", formatByteCount(3*1024*1024*1024))
}

// TestFormatProgressLine_KnownTotal renders the bar, percentage, byte counters and speed.
func TestFormatProgressLine_KnownTotal(t *testing.T) {
	line := formatProgressLine("upload a.txt", 512, 1024, 2*time.Second, wideTerminalColumns)

	expectedBar := strings.Repeat("=", 15) + ">" + strings.Repeat(" ", 14)
	assert.Equal(t, "upload a.txt ["+expectedBar+"]  50% 512 B / 1.0 KiB  256 B/s", line)
}

// TestFormatProgressLine_UnknownTotal omits the bar when the total size is unknown.
func TestFormatProgressLine_UnknownTotal(t *testing.T) {
	line := formatProgressLine("download a.txt", 2048, unknownTotalBytes, 2*time.Second, wideTerminalColumns)

	assert.Equal(t, "download a.txt  2.0 KiB  1.0 KiB/s", line)
}

// TestFormatProgressLine_ZeroElapsedTime reports a zero speed instead of dividing by zero.
func TestFormatProgressLine_ZeroElapsedTime(t *testing.T) {
	line := formatProgressLine("upload a.txt", 0, 100, 0, wideTerminalColumns)

	assert.True(t, strings.HasSuffix(line, "0 B / 100 B  0 B/s"), line)
}

// TestFormatProgressLine_NarrowTerminalKeepsBarAndCounters shortens the label so the whole line fits.
func TestFormatProgressLine_NarrowTerminalKeepsBarAndCounters(t *testing.T) {
	label := "download unlimited-ocr/Unlimited-OCR-vllm-openai-unlimited-ocr-amd64.tar"
	const maxWidth = 80

	line := formatProgressLine(label, 512, 1024, 2*time.Second, maxWidth)

	assert.LessOrEqual(t, utf8.RuneCountInString(line), maxWidth)
	assert.Contains(t, line, "...")
	assert.Contains(t, line, "[")
	assert.True(t, strings.HasSuffix(line, "256 B/s"), line)
	assert.True(t, strings.HasPrefix(line, "download "), line)
}

// TestFormatProgressLine_VeryNarrowTerminalHardTruncates cuts the whole line when even the details do not fit.
func TestFormatProgressLine_VeryNarrowTerminalHardTruncates(t *testing.T) {
	line := formatProgressLine("download a.txt", 512, 1024, 2*time.Second, 10)

	assert.Equal(t, 10, utf8.RuneCountInString(line))
}

// TestShortenLabel keeps the action word and only cuts the object key when there is room for it.
func TestShortenLabel(t *testing.T) {
	label := "download unlimited-ocr/Unlimited-OCR-vllm-openai-unlimited-ocr-amd64.tar"

	shortened := shortenLabel(label, 30)
	assert.Equal(t, 30, utf8.RuneCountInString(shortened))
	assert.True(t, strings.HasPrefix(shortened, "download "), shortened)
	assert.Contains(t, shortened, "...")

	assert.Equal(t, "download a.txt", shortenLabel("download a.txt", 30))
	assert.Equal(t, "do...ar", shortenLabel(label, 7))
}

// TestTruncateMiddle keeps both ends of the text and marks the cut with an ellipsis.
func TestTruncateMiddle(t *testing.T) {
	assert.Equal(t, "", truncateMiddle("abcdef", 0))
	assert.Equal(t, "abc", truncateMiddle("abc", 10))
	assert.Equal(t, "a...", truncateMiddle("abcdefgh", 4))
	assert.Equal(t, "ab...gh", truncateMiddle("abcdefgh", 7))
	assert.Equal(t, 9, utf8.RuneCountInString(truncateMiddle("你好世界你好世界你好", 9)))
}

// TestRenderProgressBar_Edges covers the empty and full bar states.
func TestRenderProgressBar_Edges(t *testing.T) {
	assert.Equal(t, ">"+strings.Repeat(" ", progressBarWidth-1), renderProgressBar(0, progressBarWidth))
	assert.Equal(t, strings.Repeat("=", progressBarWidth), renderProgressBar(100, progressBarWidth))
	assert.Equal(t, ">"+strings.Repeat(" ", 9), renderProgressBar(0, 10))
}

// TestFormatProgressLine_EightyColumnsKeepsActionWord shrinks the bar so the action word and part of the key survive.
func TestFormatProgressLine_EightyColumnsKeepsActionWord(t *testing.T) {
	label := "download unlimited-ocr/Unlimited-OCR-vllm-openai-unlimited-ocr-amd64.tar"
	const maxWidth = 79

	line := formatProgressLine(label, 20*1024*1024, 20*1024*1024, 2*time.Second, maxWidth)

	assert.LessOrEqual(t, utf8.RuneCountInString(line), maxWidth)
	assert.True(t, strings.HasPrefix(line, "download "), line)
	assert.Contains(t, line, "...")
	assert.Contains(t, line, "100%")
	assert.True(t, strings.HasSuffix(line, "MiB/s"), line)
}

// TestPercentOf clamps the value and treats an empty transfer as complete.
func TestPercentOf(t *testing.T) {
	assert.Equal(t, int64(100), percentOf(0, 0))
	assert.Equal(t, int64(0), percentOf(0, 100))
	assert.Equal(t, int64(50), percentOf(50, 100))
	assert.Equal(t, int64(100), percentOf(150, 100))
}

// TestProgressReporter_NarrowTerminalNeverWraps keeps every redraw on one physical line.
func TestProgressReporter_NarrowTerminalNeverWraps(t *testing.T) {
	buffer := &bytes.Buffer{}
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	reporter := newTestReporter(buffer, "download unlimited-ocr/Unlimited-OCR-vllm-openai-unlimited-ocr-amd64.tar", 1000, &currentTime)
	const terminalColumnsForTest = 60
	reporter.columns = func() int {
		return terminalColumnsForTest
	}

	for step := 1; step <= 5; step++ {
		currentTime = currentTime.Add(200 * time.Millisecond)
		reporter.advance(100)
	}
	reporter.finish()

	segments := strings.Split(buffer.String(), "\r")
	for _, segment := range segments[1:] {
		visible := strings.TrimRight(strings.TrimSuffix(segment, "\n"), " ")
		assert.LessOrEqual(t, utf8.RuneCountInString(visible), terminalColumnsForTest-1, visible)
	}
	assert.Equal(t, 1, strings.Count(buffer.String(), "\n"))
}

// TestTerminalColumns_FallsBackWhenNotATerminal uses the default width for regular files.
func TestTerminalColumns_FallsBackWhenNotATerminal(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "columns")
	require.NoError(t, err)
	defer file.Close()

	assert.Equal(t, fallbackTerminalColumns, terminalColumns(file))
}

// TestProgressReader_CountsBytes reports every byte that passes through the wrapper.
func TestProgressReader_CountsBytes(t *testing.T) {
	buffer := &bytes.Buffer{}
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	reporter := newTestReporter(buffer, "download a.bin", 10000, &currentTime)

	source := bytes.NewReader(make([]byte, 10000))
	wrapper := &progressReader{reader: source, reporter: reporter}
	copied, err := io.Copy(io.Discard, wrapper)

	require.NoError(t, err)
	assert.Equal(t, int64(10000), copied)
	assert.Equal(t, int64(10000), reporter.transferred)
	assert.Contains(t, buffer.String(), "100%")
}

// TestProgressReadSeeker_SeekResetsCounter keeps the counter aligned with the absolute offset after seeking.
func TestProgressReadSeeker_SeekResetsCounter(t *testing.T) {
	buffer := &bytes.Buffer{}
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	reporter := newTestReporter(buffer, "upload a.bin", 10, &currentTime)

	wrapper := &progressReadSeeker{reader: bytes.NewReader([]byte("0123456789")), reporter: reporter}

	readBuffer := make([]byte, 4)
	count, err := wrapper.Read(readBuffer)
	require.NoError(t, err)
	assert.Equal(t, 4, count)
	assert.Equal(t, int64(4), reporter.transferred)

	position, err := wrapper.Seek(2, io.SeekStart)
	require.NoError(t, err)
	assert.Equal(t, int64(2), position)
	assert.Equal(t, int64(2), reporter.transferred)

	count, err = wrapper.Read(readBuffer[:3])
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Equal(t, int64(5), reporter.transferred)
}

// TestProgressReadSeeker_SeekErrorKeepsCounter leaves the counter untouched when the seek fails.
func TestProgressReadSeeker_SeekErrorKeepsCounter(t *testing.T) {
	buffer := &bytes.Buffer{}
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	reporter := newTestReporter(buffer, "upload a.bin", 10, &currentTime)
	reporter.transferred = 3

	wrapper := &progressReadSeeker{reader: bytes.NewReader([]byte("0123456789")), reporter: reporter}
	_, err := wrapper.Seek(-1, io.SeekStart)

	require.Error(t, err)
	assert.Equal(t, int64(3), reporter.transferred)
}

// TestProgressReporter_ThrottlesRedraws skips redraws that happen inside the refresh period.
func TestProgressReporter_ThrottlesRedraws(t *testing.T) {
	buffer := &bytes.Buffer{}
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	reporter := newTestReporter(buffer, "upload a.bin", 1000, &currentTime)

	reporter.advance(1)
	assert.Equal(t, 1, strings.Count(buffer.String(), "\r"))

	reporter.advance(1)
	currentTime = currentTime.Add(50 * time.Millisecond)
	reporter.advance(1)
	assert.Equal(t, 1, strings.Count(buffer.String(), "\r"))

	currentTime = currentTime.Add(60 * time.Millisecond)
	reporter.advance(1)
	assert.Equal(t, 2, strings.Count(buffer.String(), "\r"))
}

// TestProgressReporter_FinishWritesFinalStateAndNewline draws the completed state before ending the line.
func TestProgressReporter_FinishWritesFinalStateAndNewline(t *testing.T) {
	buffer := &bytes.Buffer{}
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	reporter := newTestReporter(buffer, "upload a.bin", 4, &currentTime)

	reporter.advance(4)
	reporter.finish()

	output := buffer.String()
	assert.True(t, strings.HasSuffix(output, "\n"))
	assert.Contains(t, output, "100%")
}

// TestProgressReporter_PadsShorterLine clears leftovers of a longer previous line.
func TestProgressReporter_PadsShorterLine(t *testing.T) {
	buffer := &bytes.Buffer{}
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	reporter := newTestReporter(buffer, "download a.bin", unknownTotalBytes, &currentTime)

	reporter.setPosition(2 * 1024 * 1024)
	currentTime = currentTime.Add(time.Second)
	reporter.setPosition(1)

	segments := strings.Split(buffer.String(), "\r")
	previousSegment := segments[len(segments)-2]
	lastSegment := segments[len(segments)-1]
	assert.Equal(t, len(previousSegment), len(lastSegment))
	assert.True(t, strings.HasSuffix(lastSegment, " "))
}

// TestProgressReporter_NilIsNoop allows callers to skip progress without branching on every call.
func TestProgressReporter_NilIsNoop(t *testing.T) {
	var reporter *progressReporter

	assert.NotPanics(t, func() {
		reporter.advance(1)
		reporter.setPosition(1)
		reporter.finish()
	})
}

// TestShouldShowProgress_DisabledByFlag never draws a bar when the user opts out.
func TestShouldShowProgress_DisabledByFlag(t *testing.T) {
	assert.False(t, shouldShowProgress(true))
}
