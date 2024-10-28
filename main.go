package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/caltechlibrary/crossrefapi"
)

func main() {
	cclient, err := crossrefapi.NewCrossRefClient("citer", "pbalbi@dc.uba.ar")
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	http.HandleFunc("/bibtex/", func(w http.ResponseWriter, r *http.Request) {
		doi := strings.TrimPrefix(r.URL.Path, "/bibtex/")
		bibtex, err := getBibtextForDOI(doi)
		if err != nil {
			log.Printf("failed to get bibtex: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(bibtex))
	})

	http.HandleFunc("/refs/", func(w http.ResponseWriter, r *http.Request) {
		doi := strings.TrimPrefix(r.URL.Path, "/refs/")

		works, err := cclient.Works(doi)
		if err != nil {
			log.Printf("failed to get works: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl, err := template.ParseFiles("index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		type singleRef struct {
			DOI string
			Key string
		}

		collectedRefs := make([]singleRef, 0, len(works.Message.Reference))
		refs := works.Message.Reference

		// sort by key
		slices.SortFunc(refs, func(a, b *crossrefapi.Reference) int {
			return strings.Compare(a.Key, b.Key)
		})

		for _, ref := range works.Message.Reference {
			collectedRefs = append(collectedRefs, singleRef{
				DOI: ref.DOI,
				Key: ref.Key,
			})
		}

		data := struct {
			Title string
			Refs  []singleRef
		}{
			Title: strings.Join(works.Message.Title, " "),
			Refs:  collectedRefs,
		}
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func getBibtextForDOI(doi string) (string, error) {
	// http://api.crossref.org/works/10.5555/12345678/transform/application/x-bibtex
	cl := http.Client{Timeout: time.Second * 5}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://api.crossref.org/works/%s/transform/application/x-bibtex", doi), nil)
	if err != nil {
		return "", err
	}

	res, err := cl.Do(req)
	if err != nil {
		return "", err
	}

	bs, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(bs), nil
}
