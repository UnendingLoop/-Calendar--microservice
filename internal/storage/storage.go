// Package storage implements methods to read and write eventsmap from/to file
package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/model"
	"github.com/wb-go/wbf/config"
)

func LoadActualArchiveMaps(c *config.Config) (map[uint]model.EventHeap, map[uint]model.EventHeap) {
	emap, err := loadEventsFromFile(c.GetString("FILENAME"))
	if err != nil {
		log.Printf("Failed to load eventsmap from file: %v.\nUsing a new clean eventsMap.\n", err)
	} else {
		log.Println("Eventsmap successfully loaded from file.")
	}

	arch, err := loadEventsFromFile(c.GetString("ARCHIVE"))
	if err != nil {
		log.Printf("Failed to load archive from file: %v.\nUsing a new clean archiveMap.\n", err)
	} else {
		log.Println("Archive successfully loaded from file.")
	}

	return emap, arch
}

func SaveActualArchiveMaps(c *config.Config, actual, archive map[uint]model.EventHeap) []error {
	now := time.Now().Format("2006-01-02")
	res := []error{}
	actualName := c.GetString("FILENAME")
	if actualName == "" {
		actualName = now + "_actualMap.txt"
	}

	archiveName := c.GetString("ARCHIVE")
	if archiveName == "" {
		archiveName = now + "_archiveMap.txt"
	}

	if err := saveEventsToFile(actualName, actual); err != nil {
		res = append(res, fmt.Errorf("failed to save ActualMap to file: %w", err))
	}

	if err := saveEventsToFile(archiveName, archive); err != nil {
		res = append(res, fmt.Errorf("failed to save ArchiveMap to file: %w", err))
	}

	return res
}

func loadEventsFromFile(filename string) (map[uint]model.EventHeap, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[uint]model.EventHeap), nil
		}
		return make(map[uint]model.EventHeap), err
	}
	defer file.Close()

	var data map[uint]model.EventHeap
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return make(map[uint]model.EventHeap), err
	}

	return data, nil
}

func saveEventsToFile(filename string, events map[uint]model.EventHeap) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(events)
}
