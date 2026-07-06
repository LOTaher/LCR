package service

import "database/sql"

func InsertImage(db *sql.DB, digest string, manifest string) error {
	_, err := db.Exec("INSERT INTO images (digest, manifest) VALUES (?, ?) ON CONFLICT(digest) DO NOTHING", digest, manifest)
	return err
}

func GetManifestByDigest(db *sql.DB, digest string) (string, error) {
	var manifest string
	err := db.QueryRow("SELECT manifest FROM images WHERE digest = ?", digest).Scan(&manifest)
	return manifest, err
}
