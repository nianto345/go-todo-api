package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

// ============================================================
// DATABASE - Koneksi ke PostgreSQL
// ============================================================

var db *sql.DB

func initDB() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("❌ DATABASE_URL tidak ditemukan!")
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("❌ Gagal koneksi database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("❌ Database tidak merespon:", err)
	}

	// Buat tabel todos kalau belum ada
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id   SERIAL PRIMARY KEY,
			task TEXT    NOT NULL,
			done BOOLEAN NOT NULL DEFAULT false
		)
	`)
	if err != nil {
		log.Fatal("❌ Gagal buat tabel:", err)
	}

	fmt.Println("✅ Database terhubung!")
}

// ============================================================
// MODEL
// ============================================================

type Todo struct {
	ID   int    `json:"id"`
	Task string `json:"task"`
	Done bool   `json:"done"`
}

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
// HANDLERS
// ============================================================

// GET /
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `
		<h1>🚀 Go To-Do API + PostgreSQL</h1>
		<p>Selamat datang! Berikut endpoint yang tersedia:</p>
		<ul>
			<li><b>GET</b>    /todos        → Ambil semua to-do</li>
			<li><b>POST</b>   /todos        → Tambah to-do baru</li>
			<li><b>PUT</b>    /todos/{id}   → Update to-do</li>
			<li><b>DELETE</b> /todos/{id}   → Hapus to-do</li>
		</ul>
	`)
}

// GET /todos → Ambil semua to-do dari database
func getTodos(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, task, done FROM todos ORDER BY id")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Message: "Gagal ambil data"})
		return
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		rows.Scan(&t.ID, &t.Task, &t.Done)
		todos = append(todos, t)
	}

	if todos == nil {
		todos = []Todo{}
	}

	writeJSON(w, http.StatusOK, Response{Data: todos})
}

// POST /todos → Tambah to-do baru ke database
func createTodo(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Task string `json:"task"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Task == "" {
		writeJSON(w, http.StatusBadRequest, Response{Message: "Field 'task' wajib diisi"})
		return
	}

	var newTodo Todo
	err := db.QueryRow(
		"INSERT INTO todos (task, done) VALUES ($1, false) RETURNING id, task, done",
		input.Task,
	).Scan(&newTodo.ID, &newTodo.Task, &newTodo.Done)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Message: "Gagal tambah data"})
		return
	}

	writeJSON(w, http.StatusCreated, Response{
		Message: "To-do berhasil ditambahkan!",
		Data:    newTodo,
	})
}

// PUT /todos/{id} → Update to-do di database
func updateTodo(w http.ResponseWriter, r *http.Request, id int) {
	var input struct {
		Task string `json:"task"`
		Done *bool  `json:"done"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Message: "Format JSON tidak valid"})
		return
	}

	// Ambil data lama dulu
	var t Todo
	err := db.QueryRow("SELECT id, task, done FROM todos WHERE id=$1", id).
		Scan(&t.ID, &t.Task, &t.Done)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("To-do ID %d tidak ditemukan", id)})
		return
	}

	// Terapkan perubahan
	if input.Task != "" {
		t.Task = input.Task
	}
	if input.Done != nil {
		t.Done = *input.Done
	}

	_, err = db.Exec("UPDATE todos SET task=$1, done=$2 WHERE id=$3", t.Task, t.Done, t.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Message: "Gagal update data"})
		return
	}

	writeJSON(w, http.StatusOK, Response{Message: "To-do berhasil diupdate!", Data: t})
}

// DELETE /todos/{id} → Hapus to-do dari database
func deleteTodo(w http.ResponseWriter, r *http.Request, id int) {
	result, err := db.Exec("DELETE FROM todos WHERE id=$1", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Message: "Gagal hapus data"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("To-do ID %d tidak ditemukan", id)})
		return
	}

	writeJSON(w, http.StatusOK, Response{Message: fmt.Sprintf("To-do ID %d berhasil dihapus", id)})
}

// ============================================================
// ROUTER
// ============================================================

func todosRouter(w http.ResponseWriter, r *http.Request) {
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
// MAIN
// ============================================================

func main() {
	initDB()

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/todos", todosRouter)
	http.HandleFunc("/todos/", todosRouter)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("✅ Server berjalan di http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
