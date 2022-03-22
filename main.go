package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"

	"backup/consts"
	"backup/internal/config"
	"backup/internal/scan"
	"backup/internal/token"
	"backup/internal/watch"
	"backup/pkg/logger"
)

var commands = map[string]string{
	"windows": "start",
	"darwin":  "open",
	"linux":   "xdg-open",
}

func OpenBrowser() error {
	run, ok := commands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("don't know how to open things on %s platform", runtime.GOOS)
	}

	cmd := exec.Command(run, fmt.Sprintf(consts.AuthorizationCodeUrl, config.Config.PcsConfig.AppKey))
	return cmd.Run()
}

func main() {
	addr := fmt.Sprintf("%s:%d", config.Config.ServerConfig.Host, config.Config.ServerConfig.Port)
	mux := http.NewServeMux()
	mux.HandleFunc("/getToken", func(writer http.ResponseWriter, request *http.Request) {
		code := request.URL.Query().Get("code")
		if code == "" {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte("invalid code"))
			return
		}

		err := token.RefreshTokenFromServerByCode(code)
		if err != nil {
			logger.Logger.WithError(err).Error("refresh token by code fail")
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Write([]byte("server error"))
			return
		}

		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("success"))
	})

	mux.HandleFunc("/open", func(writer http.ResponseWriter, request *http.Request) {
		log.Println(OpenBrowser())
	})

	mux.HandleFunc("/addBackupPath", func(writer http.ResponseWriter, request *http.Request) {
		path := request.URL.Query().Get("path")
		if path == "" {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte("path should not be empty"))
			return
		}
		if !filepath.IsAbs(path) {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte("path should be a absolute path"))
			return
		}
		watcher, err := watch.NewWatcher(context.Background(), path)
		if err != nil {
			logger.Logger.WithError(err).Error("create watcher fail")
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Write([]byte("server error"))
			return
		}
		scanner, err := scan.NewScanner(context.Background(), path)
		if err != nil {
			logger.Logger.WithError(err).Error("create scanner fail")
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Write([]byte("server error"))
			return
		}

		scanner.Scan()
		watcher.Start()
	})

	log.Fatal(http.ListenAndServe(addr, mux))
}
