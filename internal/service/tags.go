package service

import (
	"database/sql"
	"time"
)

type ImageWithTag struct {
	ImageName string
	TagName   string
	Digest    string
	UpdatedAt string
	Downloads string
}

func UpsertTag(db *sql.DB, imageName, tagName, digest string) error {
	lastUpdated := time.Now().Format(time.RFC1123)
	_, err := db.Exec("INSERT INTO tags (image_name, tag_name, digest, updated_at) VALUES (?, ?, ?, ?) ON CONFLICT(image_name, tag_name) DO UPDATE SET digest=excluded.digest, updated_at=excluded.updated_at", imageName, tagName, digest, lastUpdated)
	return err
}

func GetDigestByTag(db *sql.DB, tagName string) (string, error) {
	var digest string
	err := db.QueryRow("SELECT digest FROM tags WHERE tag_name = ?", tagName).Scan(&digest)
	return digest, err
}

func ListImagesWithTags(db *sql.DB) ([]ImageWithTag, error) {
	images := make([]ImageWithTag, 0, 50)

	rows, err := db.Query("SELECT image_name, tag_name, digest, updated_at, downloads FROM tags")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var img ImageWithTag
		if err := rows.Scan(&img.ImageName, &img.TagName, &img.Digest, &img.UpdatedAt, &img.Downloads); err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}

func IncrementTagDownload(db *sql.DB, tagName string) error {
	_, err := db.Exec("UPDATE tags SET downloads = downloads + 1 WHERE tag_name = ?", tagName)
	if err != nil {
		return err
	}

	return nil
}
