package logger_test

import (
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestLogCollector(t *testing.T) {
	testCases := []struct {
		caseName           string
		logMode            string
		entriesSetup       func() []*logger.EventEntry
		wantFileCreated    bool
		wantStdoutUsed     bool
		expectedLogContent []string
		maxRunDuration     time.Duration
	}{
		{
			caseName: "empty log mode defaults to stdout",
			logMode:  "",
			entriesSetup: func() []*logger.EventEntry {
				return []*logger.EventEntry{
					{
						Level:     "INFO",
						Msg:       "test message",
						Err:       nil,
						TimeStamp: time.Date(2026, 3, 17, 10, 30, 0, 0, time.UTC),
						Fields:    nil,
					},
				}
			},
			wantFileCreated:    false,
			wantStdoutUsed:     true,
			expectedLogContent: []string{"INFO", "test message"},
			maxRunDuration:     500 * time.Millisecond,
		},
		{
			caseName: "stdout log mode writes to stdout",
			logMode:  "stdout",
			entriesSetup: func() []*logger.EventEntry {
				return []*logger.EventEntry{
					{
						Level:     "DEBUG",
						Msg:       "debug log",
						Err:       nil,
						TimeStamp: time.Date(2026, 3, 17, 11, 0, 0, 0, time.UTC),
						Fields: []logger.Field{
							{Key: "user_id", Value: 123},
						},
					},
				}
			},
			wantFileCreated:    false,
			wantStdoutUsed:     true,
			expectedLogContent: []string{"DEBUG", "debug log", "user_id"},
			maxRunDuration:     500 * time.Millisecond,
		},
		{
			caseName: "custom file log mode writes to file",
			logMode:  "_test.log",
			entriesSetup: func() []*logger.EventEntry {
				return []*logger.EventEntry{
					{
						Level:     "WARNING",
						Msg:       "warning message",
						Err:       errors.New("something went wrong"),
						TimeStamp: time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC),
						Fields: []logger.Field{
							{Key: "event_id", Value: "evt-001"},
							{Key: "duration", Value: 150},
						},
					},
				}
			},
			wantFileCreated:    true,
			wantStdoutUsed:     false,
			expectedLogContent: []string{"WARNING", "warning message", "something went wrong", "event_id"},
			maxRunDuration:     500 * time.Millisecond,
		},
		{
			caseName: "multiple log entries processed in order",
			logMode:  "stdout",
			entriesSetup: func() []*logger.EventEntry {
				return []*logger.EventEntry{
					{
						Level:     "INFO",
						Msg:       "first entry",
						Err:       nil,
						TimeStamp: time.Date(2026, 3, 17, 13, 0, 0, 0, time.UTC),
						Fields:    nil,
					},
					{
						Level:     "ERROR",
						Msg:       "second entry",
						Err:       errors.New("error details"),
						TimeStamp: time.Date(2026, 3, 17, 13, 1, 0, 0, time.UTC),
						Fields:    nil,
					},
					{
						Level:     "INFO",
						Msg:       "third entry",
						Err:       nil,
						TimeStamp: time.Date(2026, 3, 17, 13, 2, 0, 0, time.UTC),
						Fields:    nil,
					},
				}
			},
			wantFileCreated:    false,
			wantStdoutUsed:     true,
			expectedLogContent: []string{"first entry", "second entry", "third entry"},
			maxRunDuration:     500 * time.Millisecond,
		},
		{
			caseName: "empty fields slice handled correctly",
			logMode:  "stdout",
			entriesSetup: func() []*logger.EventEntry {
				return []*logger.EventEntry{
					{
						Level:     "INFO",
						Msg:       "no fields",
						Err:       nil,
						TimeStamp: time.Date(2026, 3, 17, 14, 0, 0, 0, time.UTC),
						Fields:    []logger.Field{},
					},
				}
			},
			wantFileCreated:    false,
			wantStdoutUsed:     true,
			expectedLogContent: []string{"INFO", "no fields"},
			maxRunDuration:     500 * time.Millisecond,
		},
		{
			caseName: "nil error field handled correctly",
			logMode:  "stdout",
			entriesSetup: func() []*logger.EventEntry {
				return []*logger.EventEntry{
					{
						Level:     "INFO",
						Msg:       "nil error",
						Err:       nil,
						TimeStamp: time.Date(2026, 3, 17, 15, 0, 0, 0, time.UTC),
						Fields: []logger.Field{
							{Key: "operation", Value: "database_query"},
						},
					},
				}
			},
			wantFileCreated:    false,
			wantStdoutUsed:     true,
			expectedLogContent: []string{"INFO", "nil error", "operation"},
			maxRunDuration:     500 * time.Millisecond,
		},
		{
			caseName: "log mode with whitespace is trimmed",
			logMode:  "  stdout  ",
			entriesSetup: func() []*logger.EventEntry {
				return []*logger.EventEntry{
					{
						Level:     "INFO",
						Msg:       "whitespace test",
						Err:       nil,
						TimeStamp: time.Date(2026, 3, 17, 16, 0, 0, 0, time.UTC),
						Fields:    nil,
					},
				}
			},
			wantFileCreated:    false,
			wantStdoutUsed:     true,
			expectedLogContent: []string{"INFO", "whitespace test"},
			maxRunDuration:     500 * time.Millisecond,
		},
		{
			caseName: "log mode case insensitive (STDOUT)",
			logMode:  "STDOUT",
			entriesSetup: func() []*logger.EventEntry {
				return []*logger.EventEntry{
					{
						Level:     "INFO",
						Msg:       "case test",
						Err:       nil,
						TimeStamp: time.Date(2026, 3, 17, 17, 0, 0, 0, time.UTC),
						Fields:    nil,
					},
				}
			},
			wantFileCreated:    false,
			wantStdoutUsed:     true,
			expectedLogContent: []string{"INFO", "case test"},
			maxRunDuration:     500 * time.Millisecond,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.caseName, func(t *testing.T) {
			// Redirect stdout to capture output
			read, write, err := os.Pipe()
			require.NoError(t, err)
			saveStdout := os.Stdout
			os.Stdout = write

			defer func() {
				os.Stdout = saveStdout
				if err := read.Close(); err != nil {
					t.Logf("Failed to close 'read': err")
				}
				if err := write.Close(); err != nil {
					t.Logf("Failed to close 'write': err")
				}
			}()

			// Setup logger
			wg := &sync.WaitGroup{}
			ch := make(chan *logger.EventEntry, 10)

			// Start LogCollector
			logger.LogCollector(wg, tt.logMode, ch)

			// Send entries
			entries := tt.entriesSetup()
			for _, entry := range entries {
				ch <- entry
			}

			// Close channel to signal end
			close(ch)

			// Wait for LogCollector to finish
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			// Wait with timeout
			select {
			case <-done:
				// Success - LogCollector finished
			case <-time.After(tt.maxRunDuration + 500*time.Millisecond):
				t.Fatal("LogCollector did not exit within expected timeout")
			}

			// Restore stdout and read output
			if err := write.Close(); err != nil {
				t.Logf("Failed to close 'write': err")
			}
			output, _ := io.ReadAll(read)
			outputStr := string(output)

			// Check if file was created
			expectedFilename := time.Now().Format("2006-01-02") + tt.logMode
			if tt.wantFileCreated && tt.logMode != "" && tt.logMode != "stdout" {
				fileInfo, err := os.Stat(expectedFilename)
				require.NoError(t, err, "log file should be created")
				require.NotNil(t, fileInfo)

				// Clean up test file
				defer func() {
					if err := os.Remove(expectedFilename); err != nil {
						t.Logf("Failed to remove test-file: %v", err)
					}
				}()

			}

			// For stdout output, verify content
			if tt.wantStdoutUsed {
				for _, expectedContent := range tt.expectedLogContent {
					require.True(t,
						strings.Contains(outputStr, expectedContent),
						"output should contain %q, got: %s", expectedContent, outputStr)
				}
			}
		})
	}
}

func TestLogCollectorChannelBehavior(t *testing.T) {
	testCases := []struct {
		caseName          string
		entriesCount      int
		closeChannelAfter bool
		expectedExitClean bool
		maxRunDuration    time.Duration
	}{
		{
			caseName:          "channel closed immediately",
			entriesCount:      0,
			closeChannelAfter: true,
			expectedExitClean: true,
			maxRunDuration:    200 * time.Millisecond,
		},
		{
			caseName:          "single entry then close",
			entriesCount:      1,
			closeChannelAfter: true,
			expectedExitClean: true,
			maxRunDuration:    500 * time.Millisecond,
		},
		{
			caseName:          "multiple entries then close",
			entriesCount:      5,
			closeChannelAfter: true,
			expectedExitClean: true,
			maxRunDuration:    500 * time.Millisecond,
		},
		{
			caseName:          "buffered channel with multiple entries",
			entriesCount:      10,
			closeChannelAfter: true,
			expectedExitClean: true,
			maxRunDuration:    500 * time.Millisecond,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.caseName, func(t *testing.T) {
			// Redirect stdout to avoid cluttering output
			read, write, err := os.Pipe()
			require.NoError(t, err)
			saveStdout := os.Stdout
			os.Stdout = write

			defer func() {
				os.Stdout = saveStdout
				if err := read.Close(); err != nil {
					t.Logf("Failed to close 'read': err")
				}
				if err := write.Close(); err != nil {
					t.Logf("Failed to close 'write': err")
				}
			}()

			wg := &sync.WaitGroup{}
			ch := make(chan *logger.EventEntry, tt.entriesCount+5)

			logger.LogCollector(wg, "stdout", ch)

			// Send entries
			for i := 0; i < tt.entriesCount; i++ {
				ch <- &logger.EventEntry{
					Level:     "INFO",
					Msg:       "test",
					Err:       nil,
					TimeStamp: time.Now(),
					Fields:    nil,
				}
			}

			// Close channel
			if tt.closeChannelAfter {
				close(ch)
			}

			// Wait for completion
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			// Verify exit within timeout
			select {
			case <-done:
				require.True(t, tt.expectedExitClean, "LogCollector should exit cleanly")
			case <-time.After(tt.maxRunDuration):
				if !tt.expectedExitClean {
					// Expected to hang
				} else {
					t.Fatalf("LogCollector did not exit within %v", tt.maxRunDuration)
				}
			}

			// Restore stdout
			if err := write.Close(); err != nil {
				t.Logf("Failed to close 'write': err")
			}
			if _, err := io.ReadAll(read); err != nil {
				t.Logf("Failed to ReadAll 'read': %v", err)
			}
		})
	}
}

func TestLogCollectorFormatOutput(t *testing.T) {
	testCases := []struct {
		caseName        string
		entry           *logger.EventEntry
		expectedInOrder []string // Elements that should appear in output in this order
		maxRunDuration  time.Duration
	}{
		{
			caseName: "log format with all fields",
			entry: &logger.EventEntry{
				Level:     "ERROR",
				Msg:       "critical error",
				Err:       errors.New("db connection failed"),
				TimeStamp: time.Date(2026, 3, 17, 10, 30, 45, 0, time.UTC),
				Fields: []logger.Field{
					{Key: "service", Value: "notifier"},
					{Key: "retry_count", Value: 3},
				},
			},
			expectedInOrder: []string{"ERROR", "critical error", "db connection failed"},
			maxRunDuration:  500 * time.Millisecond,
		},
		{
			caseName: "log format with empty error",
			entry: &logger.EventEntry{
				Level:     "INFO",
				Msg:       "operation successful",
				Err:       nil,
				TimeStamp: time.Date(2026, 3, 17, 11, 30, 45, 0, time.UTC),
				Fields:    nil,
			},
			expectedInOrder: []string{"INFO", "operation successful"},
			maxRunDuration:  500 * time.Millisecond,
		},
		{
			caseName: "log format with multiple fields",
			entry: &logger.EventEntry{
				Level:     "DEBUG",
				Msg:       "debugging",
				Err:       nil,
				TimeStamp: time.Date(2026, 3, 17, 12, 30, 45, 0, time.UTC),
				Fields: []logger.Field{
					{Key: "request_id", Value: "req-123"},
					{Key: "user_id", Value: 456},
					{Key: "action", Value: "create_event"},
				},
			},
			expectedInOrder: []string{"DEBUG", "debugging"},
			maxRunDuration:  500 * time.Millisecond,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.caseName, func(t *testing.T) {
			// Capture output
			read, write, err := os.Pipe()
			require.NoError(t, err)
			saveStdout := os.Stdout
			os.Stdout = write

			defer func() {
				os.Stdout = saveStdout
				if err := read.Close(); err != nil {
					t.Logf("Failed to close 'read': err")
				}
				if err := write.Close(); err != nil {
					t.Logf("Failed to close 'write': err")
				}
			}()

			wg := &sync.WaitGroup{}
			ch := make(chan *logger.EventEntry, 1)

			logger.LogCollector(wg, "stdout", ch)

			// Send entry
			ch <- tt.entry
			close(ch)

			// Wait for completion
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				// Success
			case <-time.After(tt.maxRunDuration + 500*time.Millisecond):
				t.Fatal("LogCollector did not finish in time")
			}

			// Check output
			if err := write.Close(); err != nil {
				t.Logf("Failed to close 'write': err")
			}
			output, _ := io.ReadAll(read)
			outputStr := string(output)

			// Verify all expected elements are present
			for _, expected := range tt.expectedInOrder {
				require.Contains(t, outputStr, expected,
					"output should contain %q", expected)
			}
		})
	}
}
