package staticbackend

import (
	"net/http"

	"github.com/staticbackendhq/core/function"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"
)

type functions struct {
	dbName    string
	datastore internal.Persister
}

func (f *functions) add(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var data internal.ExecData
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := datastore.AddFunction(conf.Name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (f *functions) update(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := new(struct {
		ID      string `json:"id"`
		Code    string `json:"code"`
		Trigger string `json:"trigger"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := datastore.UpdateFunction(conf.Name, data.ID, data.Code, data.Trigger); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (f *functions) del(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	name := getURLPart(r.URL.Path, 3)
	if err := datastore.DeleteFunction(conf.Name, name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (f *functions) exec(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	//TODONOW: this is not needed as only the fn name is required here
	/*var data internal.ExecData
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}*/

	functionName := getURLPart(r.URL.Path, 3)

	fn, err := datastore.GetFunctionForExecution(conf.Name, functionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	env := &function.ExecutionEnvironment{
		Auth:      auth,
		BaseName:  conf.Name,
		DataStore: datastore,
		Data:      fn,
	}

	if err := env.Execute(r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (f *functions) list(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	results, err := datastore.ListFunctions(conf.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, results)
}

func (f *functions) info(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	name := getURLPart(r.URL.Path, 3)

	fn, err := datastore.GetFunctionByName(conf.Name, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, fn)
}
