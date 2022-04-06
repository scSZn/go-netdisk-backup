package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"gopkg.in/natefinch/lumberjack.v2"

	"backup/internal/config"
	"backup/internal/scanner"
	"backup/pkg/util"
	"backup/ui"
	"backup/ui/theme"
)

var (
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	procSetStdHandle = kernel32.MustFindProc("SetStdHandle")
)

func main() {
	fyneOutput := &lumberjack.Logger{
		LocalTime: true,
		Filename:  fmt.Sprintf("%s/%s.log", config.Config.LogConfig.Path, "fyne"),
	}
	log.SetOutput(fyneOutput)

	panicFilename := fmt.Sprintf("%s/%s.log", config.Config.LogConfig.Path, "panic")
	panicOutput, err := os.OpenFile(panicFilename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("open panic file fail, err: +%v", err)
	}
	err = panicOutput.Truncate(0)
	if err != nil {
		log.Fatalf("truncate panic file fail, err: +%v", err)
	}
	redirectStderr(panicOutput)

	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	scanner.Manager.Start(util.NewContext())
	backupApp := app.New()
	backupApp.Settings().SetTheme(theme.CustomTheme)
	backupApp.SetIcon(resourceIconPng)

	w := backupApp.NewWindow("网盘备份")
	//w.SetCloseIntercept(func() {
	//	w.Hide()
	//})
	w.Resize(fyne.NewSize(1000, 600))

	w.SetContent(ui.Create(w))
	w.ShowAndRun()
}

func setStdHandle(stdhandle int32, handle syscall.Handle) error {
	r0, _, e1 := syscall.Syscall(procSetStdHandle.Addr(), 2, uintptr(stdhandle), uintptr(handle), 0)
	if r0 == 0 {
		if e1 != 0 {
			return error(e1)
		}
		return syscall.EINVAL
	}
	return nil
}

// redirectStderr to the file passed in
func redirectStderr(f *os.File) {
	err := setStdHandle(syscall.STD_ERROR_HANDLE, syscall.Handle(f.Fd()))
	if err != nil {
		log.Fatalf("Failed to redirect stderr to file: %v", err)
	}
	// SetStdHandle does not affect prior references to stderr
	os.Stderr = f
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
