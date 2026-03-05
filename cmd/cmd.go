// Package cmd is an entrypoint to the application and a command center
package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/cleaner"
	"github.com/UnendingLoop/-Calendar--microservice/internal/engine"
	"github.com/UnendingLoop/-Calendar--microservice/internal/logger"
	"github.com/UnendingLoop/-Calendar--microservice/internal/notifier"
	"github.com/UnendingLoop/-Calendar--microservice/internal/repository"
	"github.com/UnendingLoop/-Calendar--microservice/internal/service"
	"github.com/UnendingLoop/-Calendar--microservice/internal/storage"
	"github.com/UnendingLoop/-Calendar--microservice/internal/transport"
	"github.com/wb-go/wbf/config"
)

func InitApp() {
	log.Println("Starting Calendar application...")
	// инициализировать конфиг/ считать энвы
	appConfig := config.New()
	appConfig.EnableEnv("")
	if err := appConfig.LoadEnvFiles("./.env"); err != nil {
		log.Fatalf("Failed to load envs: %s\nExiting app...", err)
	}
	// родительский контекст для всего приложения + WG
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	wg := sync.WaitGroup{}

	// запуск слушателя логов
	logCH := make(chan *logger.EventEntry, 10)
	logger.LogCollector(&wg, appConfig.GetString("LOG_MODE"), logCH)
	eventLogger := logger.NewAsyncLogger(context.Background(), logCH)

	// создаем репо, сервис и хендлеры
	emap, arch := storage.LoadActualArchiveMaps(appConfig)
	updCh := make(chan struct{}, 1)
	repo := repository.NewEventRepository(updCh, emap, arch)
	srvc := service.NewEventService(repo)
	hndlr := transport.NewEventHandler(srvc)

	// запуск слушателя/исполнителя ивентов
	notifier.RunNotifier(ctx, &wg, repo, updCh)

	// запуск клинера
	cleaner.RunEventsCleaner(ctx, &wg, repo, appConfig.GetDuration("CLEANER_FREQ"))

	// запустить сервер
	srv := engine.NewServer(context.Background(), appConfig, hndlr, eventLogger)
	go func() {
		log.Println("Launching server on port", appConfig.GetString("APP_PORT"))
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server unexpectedly stopped: %v", err)
		}
		log.Println("Server gracefully stopping...")
	}()

	// запуск слушателя прерываний
	gsWG := sync.WaitGroup{}
	gsWG.Add(1)
	go func() {
		defer gsWG.Done()

		<-ctx.Done()
		log.Println("Interrupt received. Starting shutdown sequence...")
		serverCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// отключение сервера
		if err := srv.Shutdown(serverCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		log.Println("HTTP server stopped.")

		// закрытие канала слушателя/исполнителя
		close(updCh)
		wg.Done()

		// вызов сохранения мап в файлы
		repo.SafeLockMap()
		errs := storage.SaveActualArchiveMaps(appConfig, emap, arch)
		repo.SafeUnlockMap()
		if len(errs) == 0 {
			log.Println("Saving eventsmaps to files - successfull.")
		} else {
			log.Println(errs)
		}

		// закрытие логгера
		eventLogger.Shutdown()

		log.Printf("Exiting application...")
	}()

	gsWG.Wait()
}
