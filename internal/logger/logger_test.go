package logger

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLogCollector(t *testing.T) {
	tests := []struct {
		name     string
		logMode  string
		expected string
		hasFile  bool
	}{
		{
			name:     "empty logMode uses stdout",
			logMode:  "",
			expected: "logMode is not specified. Using stdout as default.",
			hasFile:  false,
		},
		{
			name:     "stdout mode",
			logMode:  "stdout",
			expected: "",
			hasFile:  false,
		},
		{
			name:     "file mode creates file",
			logMode:  ".log",
			expected: "",
			hasFile:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем канал
			ch := make(chan *EventEntry, 10)
			var wg sync.WaitGroup

			// Запускаем LogCollector
			LogCollector(&wg, tt.logMode, ch)

			// Создаем тестовую запись
			entry := &EventEntry{
				Level:     "INFO",
				Msg:       "Test message",
				Err:       nil,
				TimeStamp: time.Now(),
				Fields:    []Field{{Key: "key", Value: "value"}},
			}

			// Отправляем запись
			ch <- entry
			close(ch)

			// Ждем завершения
			wg.Wait()

			if tt.hasFile {
				// Проверяем, что файл создан
				filename := time.Now().Format("2006-01-02") + tt.logMode
				if _, err := os.Stat(filename); os.IsNotExist(err) {
					t.Errorf("Expected file %s to be created", filename)
				} else {
					// Читаем файл и проверяем содержимое
					file, err := os.Open(filename)
					if err != nil {
						t.Errorf("Failed to open file %s: %v", filename, err)
					} else {
						defer file.Close()
						content, err := io.ReadAll(file)
						if err != nil {
							t.Errorf("Failed to read file %s: %v", filename, err)
						} else {
							if !strings.Contains(string(content), "Test message") {
								t.Errorf("Expected 'Test message' in file, got %s", string(content))
							}
						}
					}
					// Удаляем файл после теста
					os.Remove(filename)
				}
			}
		})
	}
}
