package db

import (
	"database/sql"
	"log"

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

	log.Println("database initialized")
	return nil
}

func DB() *sql.DB { return database }

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

func GetCardsByLane(laneID int64) ([]Card, error) {
	rows, err := database.Query(
		"SELECT id, lane_id, title, description, position, created_at FROM cards WHERE lane_id = ? ORDER BY position",
		laneID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []Card
	for rows.Next() {
		var c Card
		if err := rows.Scan(&c.ID, &c.LaneID, &c.Title, &c.Description, &c.Position, &c.CreatedAt); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	if cards == nil {
		cards = []Card{}
	}
	return cards, nil
}

func GetCard(id int64) (*Card, error) {
	var c Card
	err := database.QueryRow("SELECT id, lane_id, title, description, position, created_at FROM cards WHERE id = ?", id).
		Scan(&c.ID, &c.LaneID, &c.Title, &c.Description, &c.Position, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func UpdateCard(id int64, title, description string) error {
	_, err := database.Exec("UPDATE cards SET title = ?, description = ? WHERE id = ?", title, description, id)
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
