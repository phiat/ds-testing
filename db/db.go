package db

import (
	"database/sql"
	"log"
	"strings"

	_ "modernc.org/sqlite"
)

type Lane struct {
	ID        int64
	Title     string
	Position  int
	CreatedAt string
}

type Card struct {
	ID          int64
	LaneID      int64
	Title       string
	Description string
	Tag         string
	DueDate     string
	ImageURL    string
	Position    int
	CreatedAt   string
}

var database *sql.DB

func Init(path string) error {
	var err error
	database, err = sql.Open("sqlite", path)
	if err != nil {
		return err
	}

	if _, err = database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return err
	}
	if _, err = database.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return err
	}

	_, err = database.Exec(`
		CREATE TABLE IF NOT EXISTS lanes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			lane_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			position INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (lane_id) REFERENCES lanes(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Migrations: add new columns if they don't exist
	migrations := []string{
		"ALTER TABLE cards ADD COLUMN tag TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE cards ADD COLUMN due_date TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE cards ADD COLUMN image_url TEXT NOT NULL DEFAULT ''",
	}
	for _, m := range migrations {
		database.Exec(m) // ignore error if column already exists
	}

	log.Println("database initialized")
	return nil
}

func DB() *sql.DB { return database }

// TagColors maps tag names to their hex colors.
var TagColors = map[string]string{
	"bug":     "#e0556a",
	"feature": "#6ebf8b",
	"urgent":  "#ffc49b",
	"chore":   "#8e9aaf",
	"idea":    "#8eb8e5",
}

var TagNames = []string{"bug", "feature", "urgent", "chore", "idea"}

func TagColor(tag string) string {
	if c, ok := TagColors[tag]; ok {
		return c
	}
	return "#adb6c4"
}

func TagLabel(tag string) string {
	switch tag {
	case "bug":
		return "Bug"
	case "feature":
		return "Feature"
	case "urgent":
		return "Urgent"
	case "chore":
		return "Chore"
	case "idea":
		return "Idea"
	default:
		return ""
	}
}

// --- Lanes ---

func CreateLane(title string) (*Lane, error) {
	var pos int
	database.QueryRow("SELECT COALESCE(MAX(position), -1) + 1 FROM lanes").Scan(&pos)
	r, err := database.Exec("INSERT INTO lanes (title, position) VALUES (?, ?)", title, pos)
	if err != nil {
		return nil, err
	}
	id, _ := r.LastInsertId()
	return GetLane(id)
}

func GetAllLanes() ([]Lane, error) {
	rows, err := database.Query("SELECT id, title, position, created_at FROM lanes ORDER BY position")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lanes []Lane
	for rows.Next() {
		var l Lane
		if err := rows.Scan(&l.ID, &l.Title, &l.Position, &l.CreatedAt); err != nil {
			return nil, err
		}
		lanes = append(lanes, l)
	}
	if lanes == nil {
		lanes = []Lane{}
	}
	return lanes, nil
}

func GetLane(id int64) (*Lane, error) {
	var l Lane
	err := database.QueryRow("SELECT id, title, position, created_at FROM lanes WHERE id = ?", id).
		Scan(&l.ID, &l.Title, &l.Position, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func UpdateLane(id int64, title string) error {
	_, err := database.Exec("UPDATE lanes SET title = ? WHERE id = ?", title, id)
	return err
}

func DeleteLane(id int64) error {
	_, err := database.Exec("DELETE FROM lanes WHERE id = ?", id)
	return err
}

// --- Cards ---

func CreateCard(laneID int64, title string) (*Card, error) {
	var pos int
	database.QueryRow("SELECT COALESCE(MAX(position), -1) + 1 FROM cards WHERE lane_id = ?", laneID).Scan(&pos)
	r, err := database.Exec("INSERT INTO cards (lane_id, title, position) VALUES (?, ?, ?)", laneID, title, pos)
	if err != nil {
		return nil, err
	}
	id, _ := r.LastInsertId()
	return GetCard(id)
}

func scanCard(scanner interface{ Scan(...interface{}) error }) (*Card, error) {
	var c Card
	err := scanner.Scan(&c.ID, &c.LaneID, &c.Title, &c.Description, &c.Tag, &c.DueDate, &c.ImageURL, &c.Position, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

const cardCols = "id, lane_id, title, description, tag, due_date, image_url, position, created_at"

func GetCardsByLane(laneID int64) ([]Card, error) {
	rows, err := database.Query("SELECT "+cardCols+" FROM cards WHERE lane_id = ? ORDER BY position", laneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []Card
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, *c)
	}
	if cards == nil {
		cards = []Card{}
	}
	return cards, nil
}

func GetCard(id int64) (*Card, error) {
	return scanCard(database.QueryRow("SELECT "+cardCols+" FROM cards WHERE id = ?", id))
}

func UpdateCard(id int64, title, description, tag, dueDate, imageURL string) error {
	_, err := database.Exec(
		"UPDATE cards SET title = ?, description = ?, tag = ?, due_date = ?, image_url = ? WHERE id = ?",
		title, description, tag, dueDate, imageURL, id,
	)
	return err
}

func MoveCard(id int64, toLaneID int64) error {
	var pos int
	database.QueryRow("SELECT COALESCE(MAX(position), -1) + 1 FROM cards WHERE lane_id = ?", toLaneID).Scan(&pos)
	_, err := database.Exec("UPDATE cards SET lane_id = ?, position = ? WHERE id = ?", toLaneID, pos, id)
	return err
}

func DeleteCard(id int64) error {
	_, err := database.Exec("DELETE FROM cards WHERE id = ?", id)
	return err
}

// SearchCards returns cards matching a text query and optional tag filter.
func SearchCards(query, tag string) ([]Card, error) {
	var rows *sql.Rows
	var err error
	if tag != "" && query != "" {
		rows, err = database.Query(
			"SELECT "+cardCols+" FROM cards WHERE tag = ? AND title LIKE ? ORDER BY position",
			tag, "%"+query+"%",
		)
	} else if tag != "" {
		rows, err = database.Query(
			"SELECT "+cardCols+" FROM cards WHERE tag = ? ORDER BY position", tag,
		)
	} else if query != "" {
		rows, err = database.Query(
			"SELECT "+cardCols+" FROM cards WHERE title LIKE ? ORDER BY position", "%"+query+"%",
		)
	} else {
		rows, err = database.Query("SELECT " + cardCols + " FROM cards ORDER BY position")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []Card
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, *c)
	}
	if cards == nil {
		cards = []Card{}
	}
	return cards, nil
}

// DueStatus returns a CSS class for the due date.
func DueStatus(dueDate string) string {
	if dueDate == "" {
		return ""
	}
	now := strings.Split(strings.Split(dueDate, "T")[0], " ")[0]
	due := strings.Split(dueDate, "T")[0]
	if due < now {
		return "overdue"
	}
	if due == now {
		return "due-today"
	}
	return "due-future"
}
