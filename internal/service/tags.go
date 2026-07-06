package service

import "database/sql"

type ImageWithTag struct {
	ImageName string
	TagName   string
	Digest    string
}

func UpsertTag(db *sql.DB, imageName, tagName, digest string) error {
	_, err := db.Exec("INSERT INTO tags (image_name, tag_name, digest) VALUES (?, ?, ?) ON CONFLICT(image_name, tag_name) DO UPDATE SET digest=excluded.digest", imageName, tagName, digest)
	return err
}

func GetDigestByTag(db *sql.DB, tagName string) (string, error) {
	var digest string
	err := db.QueryRow("SELECT digest FROM tags WHERE tag_name = ?", tagName).Scan(&digest)
	return digest, err
}

func ListImagesWithTags(db *sql.DB) ([]ImageWithTag, error) {
	images := make([]ImageWithTag, 0, 50)

	rows, err := db.Query("SELECT image_name, tag_name, digest FROM tags")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var img ImageWithTag
		if err := rows.Scan(&img.ImageName, &img.TagName, &img.Digest); err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}
