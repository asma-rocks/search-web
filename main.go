package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/blevesearch/bleve"
)

// SearchParams is a collection of user search params
type SearchParams struct {
	Query     string
	From      int
	Size      int
	FacetSize int
}

// NewParams translates URL to required parameter struct
func NewParams(u *url.URL) *SearchParams {
	q := u.Query()
	query := q.Get("q")
	from, err := strconv.ParseInt(q.Get("f"), 10, 8)
	if err != nil {
		from = 0
	}
	size, err := strconv.ParseInt(q.Get("s"), 10, 8)
	if err != nil {
		size = 10
	}
	facets, err := strconv.ParseInt(q.Get("fa"), 10, 8)
	if err != nil {
		facets = 8
	}
	return &SearchParams{
		Query:     query,
		From:      int(from),
		Size:      int(size),
		FacetSize: int(facets),
	}
}

//
// remove index references from hits
//
func removeIndexFromHits(sr *bleve.SearchResult) {
	for i := 0; i < len(sr.Hits); i++ {
		sr.Hits[i].Index = ""
	}
}

func main() {
	indexDir := flag.String("i", "asma.bleve", "A path to an bleve index")
	asmaDir := flag.String("a", "", "A path to asma archive directory")
	flag.Parse()

	Index, err := bleve.Open(*indexDir)
	if err != nil {
		log.Fatalln("Index directory not found")
	}

	fmt.Println(Index)

	fs := http.FileServer(http.Dir(*asmaDir))

	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		params := NewParams(r.URL)
		sq := bleve.NewQueryStringQuery(params.Query)
		newSearch := bleve.NewSearchRequest(sq)
		newSearch.Size = params.Size
		newSearch.From = params.From

		yearsFacet := bleve.NewFacetRequest("Date", params.FacetSize)
		newSearch.AddFacet("Date", yearsFacet)

		searchResults, searchErr := Index.Search(newSearch)
		if searchErr != nil {
			log.Fatalln("Search failed")
		}

		removeIndexFromHits(searchResults)

		out, err := json.Marshal(searchResults)
		if err != nil {
			log.Fatalln("Failed to serialize")
		}

		w.Header().Add("content-type", "application/json")
		w.Header().Add("access-control-allow-origin", "*")
		w.Write([]byte(out))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
