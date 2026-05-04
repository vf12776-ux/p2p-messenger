package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type Message struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	To        string `json:"to,omitempty"`
	Text      string `json:"text"`
	IsFile    bool   `json:"isFile,omitempty"`
	FileUrl   string `json:"fileUrl,omitempty"`
	FileName  string `json:"fileName,omitempty"`
	Type      string `json:"type"` // "msg", "delete", "userList"
	Timestamp int64  `json:"timestamp"`
}

type Client struct {
	conn     *websocket.Conn
	username string
}

var (
	clients    = make(map[*Client]bool)
	mu         sync.Mutex
	broadcast  = make(chan Message)
	history    = []Message{}
	historyMu  sync.RWMutex
	maxHistory = 200
)

func main() {
	go handleMessages()

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/download/", downloadHandler)

	// Static files (React build)
	staticDir := "./dist"
	fs := http.FileServer(http.Dir(staticDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(filepath.Join(staticDir, r.URL.Path)); err == nil {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	// Temporary client
	tempClient := &Client{conn: conn, username: ""}

	// Read first message (must contain username)
	var msg Message
	err = conn.ReadJSON(&msg)
	if err != nil || msg.Type != "hello" || msg.Username == "" {
		conn.WriteJSON(Message{Type: "error", Text: "Invalid hello"})
		return
	}

	tempClient.username = msg.Username

	mu.Lock()
	clients[tempClient] = true
	// Send current user list to all
	userList := []string{}
	for c := range clients {
		userList = append(userList, c.username)
	}
	mu.Unlock()
	broadcastUserList()

	// Send history to new client
	historyMu.RLock()
	for _, h := range history {
		conn.WriteJSON(h)
	}
	historyMu.RUnlock()

	// Listen for messages
	for {
		var incoming Message
		err := conn.ReadJSON(&incoming)
		if err != nil {
			mu.Lock()
			delete(clients, tempClient)
			mu.Unlock()
			broadcastUserList()
			break
		}
		incoming.Username = tempClient.username
		if incoming.Type == "" {
			incoming.Type = "msg"
		}
		if incoming.ID == "" {
			incoming.ID = uuid.New().String()
		}
		if incoming.Timestamp == 0 {
			incoming.Timestamp = time.Now().Unix()
		}
		broadcast <- incoming
	}
}

func broadcastUserList() {
	mu.Lock()
	userList := []string{}
	for c := range clients {
		userList = append(userList, c.username)
	}
	mu.Unlock()
	msg := Message{Type: "userList", Text: strings.Join(userList, ",")}
	for client := range clients {
		client.conn.WriteJSON(msg)
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		// Store in history (only non-delete and non-userList messages)
		if msg.Type == "msg" && msg.To == "" { // broadcast messages only? We'll store all msg type except delete? Better store all except delete and userList
			historyMu.Lock()
			history = append(history, msg)
			if len(history) > maxHistory {
				history = history[len(history)-maxHistory:]
			}
			historyMu.Unlock()
		} else if msg.Type == "msg" && msg.To != "" {
			// store private messages as well? Yes, store for history (both users will retrieve)
			historyMu.Lock()
			history = append(history, msg)
			if len(history) > maxHistory {
				history = history[len(history)-maxHistory:]
			}
			historyMu.Unlock()
		} else if msg.Type == "delete" {
			// Remove from history
			historyMu.Lock()
			newHistory := []Message{}
			for _, m := range history {
				if m.ID != msg.ID {
					newHistory = append(newHistory, m)
				}
			}
			history = newHistory
			historyMu.Unlock()
			// Notify all clients to delete this message from their UI
			for client := range clients {
				client.conn.WriteJSON(Message{Type: "delete", ID: msg.ID})
			}
			continue
		}

		// Send to appropriate recipient(s)
		mu.Lock()
		if msg.To == "" {
			// broadcast to all
			for client := range clients {
				client.conn.WriteJSON(msg)
			}
		} else {
			// private message: send to specific username and also to sender (for echo)
			for client := range clients {
				if client.username == msg.To || client.username == msg.Username {
					client.conn.WriteJSON(msg)
				}
			}
		}
		mu.Unlock()
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
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
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/download/")
	filePath := filepath.Join("uploads", filename)
	http.ServeFile(w, r, filePath)
}
