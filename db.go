package staticbackend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"

	"github.com/staticbackendhq/core/cache"
)

type Database struct {
	cache *cache.Cache
}

func (database *Database) dbreq(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if len(r.URL.Query().Get("bulk")) > 0 {
			database.bulkAdd(w, r)
		} else {
			database.add(w, r)
		}
	} else if r.Method == http.MethodPut {
		database.update(w, r)
	} else if r.Method == http.MethodDelete {
		database.del(w, r)
	} else if r.Method == http.MethodGet {
		p := r.URL.Path
		if strings.HasSuffix(p, "/") == false {
			p += "/"
		}

		parts := strings.Split(p, "/")

		if len(parts) == 4 {
			database.list(w, r)
		} else {
			database.get(w, r)
		}
	} else {
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (database *Database) add(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, r.URL.Path = ShiftPath(r.URL.Path)
	col, _ := ShiftPath(r.URL.Path)

	var v interface{}
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	doc, ok := v.(map[string]interface{})
	if !ok {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	doc, err = datastore.CreateDocument(auth, conf.Name, col, doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusCreated, doc)
}

func (database *Database) bulkAdd(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, r.URL.Path = ShiftPath(r.URL.Path)
	col, _ := ShiftPath(r.URL.Path)

	var v []interface{}
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := datastore.BulkCreateDocument(auth, conf.Name, col, v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusCreated, true)
}

func (database *Database) list(w http.ResponseWriter, r *http.Request) {
	page, size := getPagination(r.URL)

	params := internal.ListParams{
		Page:           page,
		Size:           size,
		SortDescending: len(r.URL.Query().Get("desc")) > 0,
	}

	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, r.URL.Path = ShiftPath(r.URL.Path)
	col, _ := ShiftPath(r.URL.Path)

	result, err := datastore.ListDocuments(auth, conf.Name, col, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, result)
}

func (database *Database) get(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	col, id := "", ""

	_, r.URL.Path = ShiftPath(r.URL.Path)
	col, r.URL.Path = ShiftPath(r.URL.Path)
	id, r.URL.Path = ShiftPath(r.URL.Path)

	result, err := datastore.GetDocumentByID(auth, conf.Name, col, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, result)
}

func (database *Database) query(w http.ResponseWriter, r *http.Request) {
	var clauses [][]interface{}
	if err := json.NewDecoder(r.Body).Decode(&clauses); err != nil {
		fmt.Println("error parsing body", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter, err := datastore.ParseQuery(clauses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	page, size := getPagination(r.URL)

	sort := r.URL.Query().Get("sort")

	params := internal.ListParams{
		Page:           page,
		Size:           size,
		SortBy:         sort,
		SortDescending: len(r.URL.Query().Get("desc")) > 0,
	}

	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		fmt.Println("error extracting conf and auth", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var col string

	_, r.URL.Path = ShiftPath(r.URL.Path)
	col, r.URL.Path = ShiftPath(r.URL.Path)

	result, err := datastore.QueryDocuments(auth, conf.Name, col, filter, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, result)
}

func (database *Database) update(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	col, id := "", ""

	_, r.URL.Path = ShiftPath(r.URL.Path)
	col, r.URL.Path = ShiftPath(r.URL.Path)
	id, r.URL.Path = ShiftPath(r.URL.Path)

	var v interface{}
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	doc, ok := v.(map[string]interface{})
	if !ok {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	result, err := datastore.UpdateDocument(auth, conf.Name, col, id, doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, result)
}

func (database *Database) increase(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// /db/col/id
	col := getURLPart(r.URL.Path, 2)
	id := getURLPart(r.URL.Path, 3)

	var v = new(struct {
		Field string `json:"field"`
		Range int    `json:"range"`
	})
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := datastore.IncrementValue(auth, conf.Name, col, id, v.Field, v.Range); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (database *Database) del(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	col, id := "", ""

	_, r.URL.Path = ShiftPath(r.URL.Path)
	col, r.URL.Path = ShiftPath(r.URL.Path)
	id, r.URL.Path = ShiftPath(r.URL.Path)

	count, err := datastore.DeleteDocument(auth, conf.Name, col, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, count)
}

func (database *Database) newID(w http.ResponseWriter, r *http.Request) {
	id := datastore.NewID()
	respond(w, http.StatusOK, id)
}

func (database *Database) listCollections(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	names, err := datastore.ListCollections(conf.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, names)
}

func (database *Database) index(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not implemented", http.StatusNotImplemented)
		return
	}

	col := r.URL.Query().Get("col")
	field := r.URL.Query().Get("field")

	if err := datastore.CreateIndex(conf.Name, col, field); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func getPagination(u *url.URL) (page int64, size int64) {
	var err error

	page, err = strconv.ParseInt(u.Query().Get("page"), 10, 64)
	if err != nil {
		page = 1
	}

	size, err = strconv.ParseInt(u.Query().Get("size"), 10, 64)
	if err != nil {
		size = 25
	}

	return
}
