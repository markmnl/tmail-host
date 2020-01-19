package main

import (
	"encoding/json"
	"io/ioutil"
	"fmt"
	"log"
	"net/http"
	"os"
)

const Version string = "0.1"
const MaxMessageSize int64 = 1000000

type Message struct {
	From string
	To string
	Time int64
	Content string
}

func main() {
	http.HandleFunc("/tmail/info", infoHandler)
	http.HandleFunc("/tmail/v1", postHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	//-----------------------------------------
	err := http.ListenAndServe(":" + port, nil)
	//-----------------------------------------
	if err != nil {
		log.Fatal(err)
	}
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "tmail-host %s", Version)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Must be POST", http.StatusBadRequest)
		return
	}
	if r.ContentLength < 1 {
		http.Error(w, "Content-Length required", http.StatusLengthRequired)
		return
	}
	if r.ContentLength > MaxMessageSize {
		http.Error(w, "Message too big", http.StatusRequestEntityTooLarge)
		return
	}
	contentType, hasContentType := r.Header["Content-Type"]
	if !hasContentType || contentType == nil || contentType[0] != "application/json" {
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
		return
	}
	// TODO can/should we validate _actual_ length? 
	body, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		log.Printf("Failed to read body: %s\n", readErr)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	var msg Message
	jsonErr := json.Unmarshal(body, &msg)
	if jsonErr != nil {
		log.Printf("Dropping message - failed to parse JSON: %s\n", jsonErr)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}


	log.Println("------- NEW MESSAGE -------")
	log.Printf("%#v\n", msg)
	log.Println("---------------------------")
}
