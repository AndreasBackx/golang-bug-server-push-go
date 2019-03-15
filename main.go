package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

const publicDirectory = "public"

var i = 1
var iMutex = sync.Mutex{}

var paths = []string{}

func pushHandler(writer http.ResponseWriter, request *http.Request) {
	pusher, supported := writer.(http.Pusher)
	if !supported {
		writer.WriteHeader(http.StatusHTTPVersionNotSupported)
		writer.Write([]byte("HTTP/2 push not supported."))
		return
	}

	html := "<html><body><h1>Golang Server Push</h1>"
	iMutex.Lock()
	i = 1
	iMutex.Unlock()

	for _, path := range paths {
		log.Printf("Pushing %s...\n", path)
		if err := pusher.Push(path, nil); err != nil {
			log.Printf("Failed to push: %v", err)
		}

		html += fmt.Sprintf("<video src=\"%s\"></video>", path)
	}

	fmt.Fprint(writer, html)
}

type responseWriterLogger struct {
	http.ResponseWriter
	status int
}

func (w *responseWriterLogger) WriteHeader(status int) {
	log.Printf("Writing status %d\n", status)
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func wrapFileServer(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wl := &responseWriterLogger{ResponseWriter: w}
		log.Printf("[...] %s %s\n", r.Method, r.URL)
		h.ServeHTTP(wl, r)

		iMutex.Lock()
		log.Printf("[%d]: %s %s (%d/%d)\n", wl.status, r.Method, r.URL, i, len(paths))
		i++
		iMutex.Unlock()
	}
}

func main() {
	args := os.Args

	if len(args) < 2 {
		log.Fatalln("Please specify an IPv4 address to listen on.")
	}

	host := args[1]

	files, err := ioutil.ReadDir(publicDirectory)
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".m4s") {
			paths = append(paths, fmt.Sprintf("/%s/%s", publicDirectory, file.Name()))
		}
	}

	http.Handle("/public/", wrapFileServer(http.StripPrefix("/public/", http.FileServer(http.Dir("public/")))))
	http.HandleFunc("/push", pushHandler)

	log.Println("Listening...")

	go func() {
		if err := http.ListenAndServe(host+":8080", nil); err != nil {
			log.Fatal(err)
		}
	}()

	if err := http.ListenAndServeTLS(host+":4430", "server.crt", "server.key", nil); err != nil {
		log.Fatal(err)
	}

}
