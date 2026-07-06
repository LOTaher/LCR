package service

import (
	"database/sql"

	"github.com/google/uuid"
)

func CreateUpload(db *sql.DB, imageName string) (string, error) {
	id := uuid.New().String()

	_, err := db.Exec("INSERT INTO uploads (uuid, image_name) VALUES (?, ?)", id, imageName)
	if err != nil {
		return "", err
	}

	return id, nil
}

func GetUploadRangeEnd(db *sql.DB, id string) (int, error) {
	var uploadId string
	var rangeEnd int
	err := db.QueryRow("SELECT uuid, range_end FROM uploads WHERE uuid = ?", id).Scan(&uploadId, &rangeEnd)
	if err != nil {
		return 0, err
	}

	return rangeEnd, nil
}

func UploadExists(db *sql.DB, id string) error {
	var uploadId string
	return db.QueryRow("SELECT uuid FROM uploads WHERE uuid = ?", id).Scan(&uploadId)
}

func UpdateUploadRangeEnd(db *sql.DB, id string, rangeEnd int) error {
	_, err := db.Exec("UPDATE uploads SET range_end = ? WHERE uuid = ?", rangeEnd, id)
	return err
}

func DeleteUpload(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM uploads WHERE uuid = ?", id)
	return err
}
