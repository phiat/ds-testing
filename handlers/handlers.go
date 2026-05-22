package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"io"
	"kanban/db"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var tmpl *template.Template

func Init(t *template.Template) {
	tmpl = t
}

type BoardData struct {
	Lanes []db.Lane
	Cards map[int64][]db.Card
	Query string
	Tag   string
}

func getBoardData() BoardData {
	return getFilteredBoardData("", "")
}

func getFilteredBoardData(query, tag string) BoardData {
	lanes, _ := db.GetAllLanes()

	var allCards []db.Card
	if query != "" || tag != "" {
		allCards, _ = db.SearchCards(query, tag)
	} else {
		for _, l := range lanes {
			cards, _ := db.GetCardsByLane(l.ID)
			allCards = append(allCards, cards...)
		}
	}

	cards := make(map[int64][]db.Card)
	for _, c := range allCards {
		cards[c.LaneID] = append(cards[c.LaneID], c)
	}
	// Ensure empty lanes still appear
	for _, l := range lanes {
		if cards[l.ID] == nil {
			cards[l.ID] = []db.Card{}
		}
	}
	return BoardData{Lanes: lanes, Cards: cards, Query: query, Tag: tag}
}

func renderBoard(w http.ResponseWriter) {
	tmpl.ExecuteTemplate(w, "board.html", getBoardData())
}

func renderBoardWithData(w http.ResponseWriter, data BoardData) {
	tmpl.ExecuteTemplate(w, "board.html", data)
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
	card, err := db.CreateCard(laneID, title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Apply optional fields from quick create
	if tag := r.FormValue("tag"); tag != "" {
		db.UpdateCard(card.ID, card.Title, "", tag, "", "")
	}
	if imageURL := r.FormValue("image_url"); imageURL != "" {
		db.UpdateCard(card.ID, card.Title, "", "", "", imageURL)
	}
	renderBoard(w)
}

func UpdateCard(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	// Parse multipart form for file uploads (32 MB max)
	r.ParseMultipartForm(32 << 20)

	title := r.FormValue("title")
	description := r.FormValue("description")
	tag := r.FormValue("tag")
	dueDate := r.FormValue("due_date")
	imageURL := r.FormValue("image_url")

	// Handle file upload if present
	if file, header, err := r.FormFile("image_file"); err == nil {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
			name := randomName() + ext
			if err := saveUpload(file, name); err == nil {
				imageURL = "/static/uploads/" + name
			}
		}
	}

	db.UpdateCard(id, title, description, tag, dueDate, imageURL)
	renderBoard(w)
}

func randomName() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func saveUpload(src io.Reader, name string) error {
	os.MkdirAll("static/uploads", 0755)
	dst, err := os.Create(filepath.Join("static/uploads", name))
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func EditCardForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	card, err := db.GetCard(id)
	if err != nil {
		http.Error(w, "card not found", http.StatusNotFound)
		return
	}
	lanes, _ := db.GetAllLanes()
	tmpl.ExecuteTemplate(w, "card-edit.html", map[string]interface{}{
		"Card":  card,
		"Lanes": lanes,
		"Tags":  db.TagNames,
	})
}

func EditCardDetail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	card, err := db.GetCard(id)
	if err != nil {
		http.Error(w, "card not found", http.StatusNotFound)
		return
	}
	lanes, _ := db.GetAllLanes()
	tmpl.ExecuteTemplate(w, "card-detail.html", map[string]interface{}{
		"Card":  card,
		"Lanes": lanes,
		"Tags":  db.TagNames,
	})
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

// --- Search ---

func SearchCards(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("q")
	tag := r.FormValue("tag")
	data := getFilteredBoardData(q, tag)
	renderBoardWithData(w, data)
}
