package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"backup/internal/scanner"
	"backup/pkg/util"
	"backup/ui"
	"backup/ui/theme"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	scanner.Manager.Start(util.NewContext())
	backupApp := app.New()
	backupApp.Settings().SetTheme(theme.CustomTheme)
	backupApp.SetIcon(resourceIconPng)

	w := backupApp.NewWindow("网盘备份")
	w.Resize(fyne.NewSize(1000, 600))

	w.SetContent(ui.Create(w))
	w.ShowAndRun()
}

//func PCSTab(window fyne.Window) fyne.CanvasObject {

//configContainer := container.New(configLayout)
//}

//func startGUI() {
//	backupApp := app.New()
//
//}

//func startServer() {
//	addr := fmt.Sprintf("%s:%d", config.Config.ServerConfig.Host, config.Config.ServerConfig.Port)
//	mux := http.NewServeMux()
//	mux.HandleFunc("/getToken", func(writer http.ResponseWriter, request *http.Request) {
//		code := request.URL.Query().Get("code")
//		if code == "" {
//			writer.WriteHeader(http.StatusBadRequest)
//			writer.Write([]byte("invalid code"))
//			return
//		}
//
//		err := token.RefreshTokenFromServerByCode(code)
//		if err != nil {
//			logger.Logger.WithError(err).Error("refresh token by code fail")
//			writer.WriteHeader(http.StatusInternalServerError)
//			writer.Write([]byte("server error"))
//			return
//		}
//
//		writer.WriteHeader(http.StatusOK)
//		writer.Write([]byte("success"))
//	})
//
//	mux.HandleFunc("/open", func(writer http.ResponseWriter, request *http.Request) {
//		log.Println(util.OpenBrowser())
//	})
//
//	mux.HandleFunc("/addBackupPath", func(writer http.ResponseWriter, request *http.Request) {
//		path := request.URL.Query().Get("path")
//		if path == "" {
//			writer.WriteHeader(http.StatusBadRequest)
//			writer.Write([]byte("path should not be empty"))
//			return
//		}
//		if !filepath.IsAbs(path) {
//			writer.WriteHeader(http.StatusBadRequest)
//			writer.Write([]byte("path should be a absolute path"))
//			return
//		}
//		watcher, err := watch.NewUploadWatcher(context.Background(), path)
//		if err != nil {
//			logger.Logger.WithError(err).Error("create watcher fail")
//			writer.WriteHeader(http.StatusInternalServerError)
//			writer.Write([]byte("server error"))
//			return
//		}
//		scanner, err := scanner.NewScanner(context.Background(), path)
//		if err != nil {
//			logger.Logger.WithError(err).Error("create scanner fail")
//			writer.WriteHeader(http.StatusInternalServerError)
//			writer.Write([]byte("server error"))
//			return
//		}
//
//		scanner.Scan()
//		watcher.Start()
//	})
//
//	log.Fatal(http.ListenAndServe(addr, mux))
//}
