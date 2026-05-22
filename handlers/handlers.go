package handlers

import (
	"html/template"
	"kanban/db"
	"net/http"
	"strconv"
)

var tmpl *template.Template

func Init(t *template.Template) {
	tmpl = t
}

// BoardData is the view model for the full board.
type BoardData struct {
	Lanes []db.Lane
	Cards map[int64][]db.Card
}

func getBoardData() BoardData {
	lanes, _ := db.GetAllLanes()
	cards := make(map[int64][]db.Card)
	for _, l := range lanes {
		cards[l.ID], _ = db.GetCardsByLane(l.ID)
	}
	return BoardData{Lanes: lanes, Cards: cards}
}

func renderBoard(w http.ResponseWriter) {
	tmpl.ExecuteTemplate(w, "board.html", getBoardData())
}

// --- Pages ---

func Index(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "base.html", getBoardData())
}

func Board(w http.ResponseWriter, r *http.Request) {
	renderBoard(w)
}

// --- Lanes ---

func CreateLane(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	if _, err := db.CreateLane(title); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderBoard(w)
}

func UpdateLane(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	title := r.FormValue("title")
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	db.UpdateLane(id, title)
	renderBoard(w)
}

func EditLaneForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	lane, err := db.GetLane(id)
	if err != nil {
		http.Error(w, "lane not found", http.StatusNotFound)
		return
	}
	tmpl.ExecuteTemplate(w, "lane-edit.html", map[string]interface{}{"Lane": lane})
}

func DeleteLane(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	db.DeleteLane(id)
	renderBoard(w)
}

// --- Cards ---

func CreateCard(w http.ResponseWriter, r *http.Request) {
	laneID, _ := strconv.ParseInt(r.FormValue("lane_id"), 10, 64)
	title := r.FormValue("title")
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	if _, err := db.CreateCard(laneID, title); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderBoard(w)
}

func UpdateCard(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	title := r.FormValue("title")
	description := r.FormValue("description")
	db.UpdateCard(id, title, description)
	renderBoard(w)
}

func EditCardForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	card, err := db.GetCard(id)
	if err != nil {
		http.Error(w, "card not found", http.StatusNotFound)
		return
	}
	tmpl.ExecuteTemplate(w, "card-edit.html", map[string]interface{}{"Card": card})
}

func MoveCard(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	toLaneID, _ := strconv.ParseInt(r.FormValue("lane_id"), 10, 64)
	db.MoveCard(id, toLaneID)
	renderBoard(w)
}

func DeleteCard(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	db.DeleteCard(id)
	renderBoard(w)
}
