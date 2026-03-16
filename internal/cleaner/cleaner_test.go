package cleaner

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type mockEventRepository struct {
	archiveExpiredCalls int
	mu                  sync.Mutex
}

func (m *mockEventRepository) ArchiveExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.archiveExpiredCalls++
	return 1 // Возвращаем фиктивное значение
}

func (m *mockEventRepository) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.archiveExpiredCalls
}

func TestRunEventsCleaner(t *testing.T) {
	mockRepo := &mockEventRepository{}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Запускаем cleaner с частотой 10ms
	freq := 10 * time.Millisecond
	RunEventsCleaner(ctx, &wg, mockRepo, freq)

	// Ждем около 50ms, чтобы прошло несколько тиков (5-6 вызовов)
	time.Sleep(50 * time.Millisecond)

	// Проверяем, что ArchiveExpired был вызван несколько раз
	calls := mockRepo.getCalls()
	require.Greater(t, calls, 3, fmt.Sprintf("Expected at least 3 calls to ArchiveExpired, got %d", calls))

	// Отменяем контекст
	cancel()

	// Ждем завершения горутины
	wg.Wait()

	// Проверяем, что после отмены вызовов больше не было
	finalCalls := mockRepo.getCalls()
	require.Equal(t, calls, finalCalls, fmt.Sprintf("Expected 0 calls after cancel, but got %d", finalCalls-calls))
}
