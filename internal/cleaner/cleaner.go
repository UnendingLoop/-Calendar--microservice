// Package cleaner - provides function RunEventsCleaner which runs archivation of outdated events once in provided period(freq)
package cleaner

import (
	"context"
	"log"
	"sync"
	"time"
)

type eventRepository interface {
	ArchiveExpired() int
}

func RunEventsCleaner(ctx context.Context, wg *sync.WaitGroup, repo eventRepository, freq time.Duration) {
	wg.Add(1)
	ticker := time.NewTicker(freq)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				log.Println("Cleaner's ctx is cancelled. Exiting cleaner...")
				return
			case <-ticker.C:
				n := repo.ArchiveExpired()
				log.Printf("Cleaner cleaned %d events.", n)
			}
		}
	}()
}
