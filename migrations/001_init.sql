CREATE TABLE IF NOT EXISTS images (
	digest TEXT PRIMARY KEY,
	manifest TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
	image_name TEXT NOT NULL,
	tag_name TEXT NOT NULL,
	digest TEXT NOT NULL REFERENCES images(digest),
	PRIMARY KEY (image_name, tag_name)
);

CREATE TABLE IF NOT EXISTS uploads (
	uuid TEXT PRIMARY KEY,
	image_name TEXT NOT NULL,
	range_end INTEGER NOT NULL DEFAULT 0
);
