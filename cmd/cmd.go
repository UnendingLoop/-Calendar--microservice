// Package cmd is an entrypoint to the application and a command center
package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"calendar/config"
	"calendar/internal/handler"
	"calendar/internal/repository"
	"calendar/internal/service"
	"calendar/internal/storage"
	"calendar/mwlogger"

	"github.com/go-chi/chi/v5"
)

func InitApp() {
	// считать порт и имя файла из env
	port, filename := config.GetEnvs()

	// создать карту с событиями - считать из файла при запуске и записать в файл при выходе - грейсфул шатдаун
	emap, err := storage.LoadEventsFromFile(filename)
	if err != nil {
		log.Printf("Failed to load eventsmap from file: %v.\nUsing a new clean eventsmap.\n", err)
	} else {
		log.Println("Eventsmap successfully loaded from file.")
	}

	// запустить сервер через middleware
	repo := repository.NewEventRepository(emap)
	srvc := service.NewEventService(repo)
	hndlr := handler.NewEventHandler(srvc)

	r := chi.NewRouter()
	r.Post("/create_event", hndlr.CreateEvent)       // создание нового события
	r.Post("/update_event", hndlr.UpdateEvent)       // обновление существующего // должен быть метод PATCH, но задание требует использование именно POST
	r.Post("/delete_event", hndlr.DeleteEvent)       // удаление // по заданию POST, должно быть DELETE
	r.Get("/events_for_day", hndlr.GetDayEvents)     // получить все события на день ?user_id=1&date=2023-12-31
	r.Get("/events_for_week", hndlr.GetWeekEvents)   // события на неделю ?user_id=1&date=2023-12-31
	r.Get("/events_for_month", hndlr.GetMonthEvents) // события на месяц ?user_id=1&date=2023-12-31

	// оборачиваем mux в middleware
	loggedRouter := mwlogger.MWLogger(r)

	srv := http.Server{
		Addr:         ":" + port,
		Handler:      loggedRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Println("Launching server on port", port)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server stopped: %v", err)
		}
		log.Println("Server gracefully stopping...")
	}()

	// слушатель прерываний
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		<-sig
		log.Println("Interrupt received. Starting shutdown sequence...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// отключение сервера
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		log.Println("HTTP server stopped. Writing eventsmap to file...")

		// вызов сохранения карты в файл
		repo.SafeLockMap()
		if err := storage.SaveEventsToFile(filename, emap); err != nil {
			log.Printf("Failed to save eventsmap to file: %v", err)
		} else {
			log.Println("Eventsmap successfully saved to file.")
		}
		repo.SafeUnlockMap()

		log.Printf("Exiting application...")
		wg.Done()
	}()

	wg.Wait()
}
