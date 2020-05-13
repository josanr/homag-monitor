package main

import (
	"encoding/json"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"strconv"
)

type okResponse struct {
	Message string `json:"message"`
}

func startControlUI(servicePort int) {
	http.HandleFunc("/setUid/", setUIDHandler)
	log.Println("CUI starting")
	err := http.ListenAndServe(":"+strconv.Itoa(servicePort), nil)
	if err != nil {
		log.Println("could not start control interface")
		log.Println(err)
		return
	}

}

func setUIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path != "/setUid/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "wrong request method", http.StatusBadRequest)
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "request form data corrupt", http.StatusBadRequest)
		return
	}
	uid := r.FormValue("uid")
	monitorConfig.UserID, _ = strconv.Atoi(uid)
	viper.Set("lastUserID", monitorConfig.UserID)
	_ = viper.WriteConfig()
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(okResponse{"action done, id set: " + strconv.Itoa(monitorConfig.UserID)})
	return
}
