// Package storage implements methods to read and write eventsmap from/to file
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/model"
	"github.com/wb-go/wbf/config"
)

func LoadActualArchiveMaps(c *config.Config) (map[uint][]model.HeapEntity, map[uint][]model.Event) {
	emap, err := loadEventsFromFile(c.GetString("FILENAME"))
	if err != nil {
		log.Printf("Failed to load eventsmap from file: %v.\nUsing a new clean eventsMap.\n", err)
	} else {
		log.Println("Eventsmap successfully loaded from file.")
	}

	arch, err := loadArchiveFromFile(c.GetString("ARCHIVE"))
	if err != nil {
		log.Printf("Failed to load archive from file: %v.\nUsing a new clean archiveMap.\n", err)
	} else {
		log.Println("Archive successfully loaded from file.")
	}

	return emap, arch
}

func SaveActualArchiveMaps(c *config.Config, actual map[uint][]model.HeapEntity, archive map[uint][]model.Event) []error {
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

func loadEventsFromFile(filename string) (map[uint][]model.HeapEntity, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[uint][]model.HeapEntity), nil
		}
		return make(map[uint][]model.HeapEntity), err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file %q after loading events from it: %v", filename, err)
		}
	}()

	data := map[uint][]model.HeapEntity{}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return make(map[uint][]model.HeapEntity), err
	}

	return data, nil
}

func loadArchiveFromFile(filename string) (map[uint][]model.Event, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[uint][]model.Event), nil
		}
		return make(map[uint][]model.Event), err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file %q after loading archive from it: %v", filename, err)
		}
	}()

	data := map[uint][]model.Event{}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return make(map[uint][]model.Event), err
	}

	return data, nil
}

func saveEventsToFile[T map[uint][]model.Event | map[uint][]model.HeapEntity](filename string, events T) error {
	if events == nil {
		return errors.New("provided map is a nil-map")
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file %q after saving data in it: %v", filename, err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(events)
}
