package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	Username string `json:"username"`
	Text     string `json:"text"`
	IsFile   bool   `json:"isFile,omitempty"`
	FileUrl  string `json:"fileUrl,omitempty"`
	FileName string `json:"fileName,omitempty"`
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	clients[conn] = true

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			delete(clients, conn)
			break
		}
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func main() {
	go handleMessages()

	http.HandleFunc("/ws", handleWebSocket)

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		err := r.ParseMultipartForm(10 << 20) // 10 MB
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "File not found", http.StatusBadRequest)
			return
		}
		defer file.Close()

		os.MkdirAll("uploads", os.ModePerm)
		filePath := filepath.Join("uploads", handler.Filename)
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		w.Write([]byte("/download/" + handler.Filename))
	})

	http.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		filename := strings.TrimPrefix(r.URL.Path, "/download/")
		filePath := filepath.Join("uploads", filename)
		http.ServeFile(w, r, filePath)
	})

	// Статика React (поддержка SPA)
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "../dist" // для локального запуска из папки server
	}
	fs := http.FileServer(http.Dir(staticDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Если запрос к существующему файлу – отдаём его
		if _, err := os.Stat(filepath.Join(staticDir, r.URL.Path)); err == nil {
			fs.ServeHTTP(w, r)
			return
		}
		// Иначе отдаём index.html (для клиентской маршрутизации)
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	})

	// Порт из окружения (для Render) или 8080 по умолчанию
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
