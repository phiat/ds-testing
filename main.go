package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"kanban/db"
	"kanban/handlers"
)

//go:embed templates/* static/*
var assets embed.FS

func main() {
	if err := db.Init("kanban.db"); err != nil {
		log.Fatal(err)
	}

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"dict":      dict,
		"tagColor":  db.TagColor,
		"tagLabel":  db.TagLabel,
		"dueStatus": db.DueStatus,
	}).ParseFS(assets, "templates/*.html"))

	handlers.Init(tmpl)

	mux := http.NewServeMux()

	staticFS, _ := fs.Sub(assets, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	mux.HandleFunc("GET /", handlers.Index)
	mux.HandleFunc("GET /board", handlers.Board)

	mux.HandleFunc("POST /lanes", handlers.CreateLane)
	mux.HandleFunc("GET /lanes/{id}/edit-form", handlers.EditLaneForm)
	mux.HandleFunc("PUT /lanes/{id}", handlers.UpdateLane)
	mux.HandleFunc("DELETE /lanes/{id}", handlers.DeleteLane)

	mux.HandleFunc("GET /cards/{id}/edit", handlers.EditCardForm)
	mux.HandleFunc("GET /cards/{id}/detail", handlers.EditCardDetail)
	mux.HandleFunc("POST /cards", handlers.CreateCard)
	mux.HandleFunc("PUT /cards/{id}", handlers.UpdateCard)
	mux.HandleFunc("PUT /cards/{id}/move", handlers.MoveCard)
	mux.HandleFunc("DELETE /cards/{id}", handlers.DeleteCard)

	mux.HandleFunc("GET /search", handlers.SearchCards)

	log.Println("kanban running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// dict creates a map from alternating key-value pairs for use in templates.
func dict(values ...interface{}) map[string]interface{} {
	if len(values)%2 != 0 {
		panic("dict: odd number of arguments")
	}
	m := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key := values[i].(string)
		m[key] = values[i+1]
	}
	return m
}
