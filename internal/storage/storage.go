// Package storage implements methods to read and write eventsmap from/to file
package storage

import (
	"encoding/json"
	"os"

	"calendar/model"
)

func LoadEventsFromFile(filename string) (map[model.UserID][]*model.Event, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[model.UserID][]*model.Event), nil
		}
		return make(map[model.UserID][]*model.Event), err
	}
	defer file.Close()

	var data map[model.UserID][]*model.Event
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return make(map[model.UserID][]*model.Event), err
	}

	return data, nil
}

func SaveEventsToFile(filename string, events map[model.UserID][]*model.Event) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(events)
}
