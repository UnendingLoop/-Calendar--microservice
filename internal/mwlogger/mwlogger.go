// Package mwlogger writes to stdOut all http-requests description and start/finish time
package mwlogger

import (
	"log"
	"net/http"
	"time"
)

// MWLogger - обёртка для логирования запросов.
func MWLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// логируем входящий запрос
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		// вызываем следующий обработчик в цепочке
		next.ServeHTTP(w, r)

		// логируем завершение
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}
