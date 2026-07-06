package controller

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	"lcr/internal/service"
)

type HomePageData struct {
	Images []service.ImageWithTag
}

func RenderHomePage(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		images, err := service.ListImagesWithTags(db)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		data := HomePageData{
			Images: images,
		}

		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		tmpl.Execute(w, data)
	}
}
