package main

import (
	"context"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	log.Info().Msg(">>> CR Test <<<")
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	block := make(chan bool, 1)
	signals := make(chan os.Signal, 1)
	//signal.Notify(quit, os.Interrupt)
	signal.Notify(signals)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", 8080),
		Handler:      &myHandler{&quit, &block},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	go func() {
		for s := range signals {
			log.Info().Msgf("Received signal: %s, %d", s.String(), s.(syscall.Signal))
		}
	}()

	go func() {
		<-quit
		log.Info().Msg("Shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to shudown")
		}
		close(done)
	}()

	go func() {
		<-block
		log.Info().Msg("Shutting down and block")

		if err := server.Shutdown(nil); err != nil {
			log.Fatal().Err(err).Msg("failed to shudown")
		}
	}()

	go func() {
		log.Info().Msgf("Listening on %s", server.Addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal().Msgf("Failed to listen on: %v", err)
		}
	}()

	<-done
	close(signals)
}

type myHandler struct {
	quit  *chan os.Signal
	block *chan bool
}

func (m *myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("All good!\n"))
	case "/quit":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Quitting...\n"))
		close(*m.quit)
	case "/close":
		closeConnection(w)
		close(*m.block)
	case "/crash":
		os.Exit(1)
	case "/noResponse":
		closeConnection(w)
	case "/oom":
		malloc()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("All good!\n"))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func closeConnection(w http.ResponseWriter) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "looks like we can't hijack to connection\n", http.StatusInternalServerError)
		return
	}

	conn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	conn.Close()
}

func malloc() {
	size := 1024 * 1024 // 1mB
	count := 10240      // 10g
	chunks := [][]byte{}

	chunk := make([]byte, size)
	for i := 0; i < size; i++ {
		chunk[i] = byte('x')
	}

	for i := 0; i < count; i++ {
		if i%10 == 0 {
			log.Info().Msgf("Allocated %d MB", i)
		}

		tmp := make([]byte, size)
		copy(tmp, chunk)
		chunks = append(chunks, tmp)
	}

	log.Info().Msgf("Done allocation %d MB", count)
}
