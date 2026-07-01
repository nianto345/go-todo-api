package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os" // Untuk membaca environment variable
	"strconv"
	"strings"
)

// ============================================================
// MODEL - Struktur data To-Do
// ============================================================

type Todo struct {
	ID   int    `json:"id"`
	Task string `json:"task"`
	Done bool   `json:"done"`
}

// Penyimpanan sementara (in-memory)
var todos = []Todo{
	{ID: 1, Task: "Belajar Golang dasar", Done: true},
	{ID: 2, Task: "Buat REST API pertama", Done: false},
	{ID: 3, Task: "Deploy ke Railway", Done: false},
}
var nextID = 4

// ============================================================
// RESPONSE HELPER - Format respons JSON
// ============================================================

type Response struct {
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ============================================================
// HANDLERS - Fungsi pengolah request
// ============================================================

// GET / → Halaman utama
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `
		<h1>🚀 Go To-Do API</h1>
		<p>Selamat datang! Berikut endpoint yang tersedia:</p>
		<ul>
			<li><b>GET</b>  /todos        → Ambil semua to-do</li>
			<li><b>POST</b> /todos        → Tambah to-do baru</li>
			<li><b>PUT</b>  /todos/{id}   → Update to-do</li>
			<li><b>DELETE</b> /todos/{id} → Hapus to-do</li>
		</ul>
	`)
}

// GET /todos → Ambil semua to-do
func getTodos(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Response{Data: todos})
}

// POST /todos → Tambah to-do baru
func createTodo(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Task string `json:"task"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Task == "" {
		writeJSON(w, http.StatusBadRequest, Response{Message: "Field 'task' wajib diisi"})
		return
	}

	newTodo := Todo{ID: nextID, Task: input.Task, Done: false}
	todos = append(todos, newTodo)
	nextID++

	writeJSON(w, http.StatusCreated, Response{
		Message: "To-do berhasil ditambahkan!",
		Data:    newTodo,
	})
}

// PUT /todos/{id} → Tandai selesai / update task
func updateTodo(w http.ResponseWriter, r *http.Request, id int) {
	var input struct {
		Task string `json:"task"`
		Done *bool  `json:"done"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Message: "Format JSON tidak valid"})
		return
	}

	for i, t := range todos {
		if t.ID == id {
			if input.Task != "" {
				todos[i].Task = input.Task
			}
			if input.Done != nil {
				todos[i].Done = *input.Done
			}
			writeJSON(w, http.StatusOK, Response{
				Message: "To-do berhasil diupdate!",
				Data:    todos[i],
			})
			return
		}
	}

	writeJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("To-do dengan ID %d tidak ditemukan", id)})
}

// DELETE /todos/{id} → Hapus to-do
func deleteTodo(w http.ResponseWriter, r *http.Request, id int) {
	for i, t := range todos {
		if t.ID == id {
			todos = append(todos[:i], todos[i+1:]...)
			writeJSON(w, http.StatusOK, Response{Message: fmt.Sprintf("To-do ID %d berhasil dihapus", id)})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("To-do dengan ID %d tidak ditemukan", id)})
}

// ============================================================
// ROUTER - Arahkan request ke handler yang tepat
// ============================================================

func todosRouter(w http.ResponseWriter, r *http.Request) {
	// /todos → list & create
	if r.URL.Path == "/todos" || r.URL.Path == "/todos/" {
		switch r.Method {
		case http.MethodGet:
			getTodos(w, r)
		case http.MethodPost:
			createTodo(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, Response{Message: "Method tidak diizinkan"})
		}
		return
	}

	// /todos/{id} → update & delete
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/todos/"), "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Message: "ID harus berupa angka"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		updateTodo(w, r, id)
	case http.MethodDelete:
		deleteTodo(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Message: "Method tidak diizinkan"})
	}
}

// ============================================================
// MAIN - Jalankan server
// ============================================================

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/todos", todosRouter)
	http.HandleFunc("/todos/", todosRouter)
	// Menentukan port dari environment variable atau default ke 8080
	port := os.Getenv("PORT")
	if port == "" {
    port = "8080"
	}
	fmt.Printf("✅ Server berjalan di http://localhost:%s\n", port)
	fmt.Println("📋 Endpoint tersedia:")
	fmt.Println("   GET    /todos")
	fmt.Println("   POST   /todos")
	fmt.Println("   PUT    /todos/{id}")
	fmt.Println("   DELETE /todos/{id}")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
