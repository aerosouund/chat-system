package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (ms *MessageServer) GetMuxVarsMiddleware(f ExecFunc) HttpHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		f(vars, w, r)
	}
}
