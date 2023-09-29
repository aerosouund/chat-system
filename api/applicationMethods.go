package main

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

func (as *AppServer) GetApplication(routerVars map[string]string, w http.ResponseWriter, r *http.Request) HttpHandlerFunc {

	name := routerVars["token"]
	app, err := as.ApplicationStorage.GetApplication(name)

	return func(http.ResponseWriter, *http.Request) {
		if err != nil {
			logrus.Info("Error in fetching application", err)
		}
		jsonData, err := json.Marshal(app)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, jsonData)
	}
}

func (as *AppServer) GetApplications(routerVars map[string]string, w http.ResponseWriter, r *http.Request) HttpHandlerFunc {
	app, err := as.ApplicationStorage.GetAll()

	return func(http.ResponseWriter, *http.Request) {
		if err != nil {
			logrus.Info("Error in fetching applications", err)
		}
		jsonData, err := json.Marshal(app)
		w.Header().Set("Content-Type", "application/json")

		_, err = w.Write(jsonData)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
}

func (as *AppServer) DeleteApplication(routerVars map[string]string, w http.ResponseWriter, r *http.Request) HttpHandlerFunc {

	token := routerVars["token"]

	err := as.ApplicationStorage.DeleteApplication(token)

	return func(http.ResponseWriter, *http.Request) {
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
}

func (as *AppServer) CreateApplication(routerVars map[string]string, w http.ResponseWriter, r *http.Request) HttpHandlerFunc {
	name := routerVars["name"]

	app, err := as.ApplicationStorage.CreateApplication(name)

	return func(http.ResponseWriter, *http.Request) {
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonData, err := json.Marshal(app)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, jsonData)
	}
}
