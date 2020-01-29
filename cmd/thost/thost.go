package main

import (
	"crypto/sha256"
	"errors"
	"encoding/json"
	"encoding/base64"
	"io/ioutil"
	"fmt"
	"log"
	"net/http"
	"os"
	"github.com/markmnl/tmail-store/tstore/pkg"
	"github.com/markmnl/tmail-store-stdout/tstore-stdout/pkg"
)

// Version of this server
const Version string = "0.1"
// MaxMessageSize this server will entertain
const MaxMessageSize int64 = 1000000


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
		http.Error(w, "", http.StatusNotFound)
		return
	}
	if r.ContentLength < 1 {
		http.Error(w, "", http.StatusLengthRequired)
		return
	}
	if r.ContentLength > MaxMessageSize {
		http.Error(w, "", http.StatusRequestEntityTooLarge)
		return
	}
	contentType, hasContentType := r.Header["Content-Type"]
	if !hasContentType || contentType == nil || contentType[0] != "application/json" {
		http.Error(w, "", http.StatusUnsupportedMediaType)
		return
	}
	// TODO can/should we validate _actual_ length? 
	body, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		log.Printf("Failed to read body: %s\n", readErr)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// parse..
	var msg tstore.Msg
	msgPtr := &msg
	jsonErr := json.Unmarshal(body, msgPtr)
	if jsonErr != nil {
		log.Printf("ERROR Dropping message - failed to parse JSON: %s\n", jsonErr)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if msg.PID64 != "" {
		if len(msg.PID64) != 44 {
			http.Error(w, "Invalid pid length", http.StatusBadRequest)
			return
		}
		pid64Bytes, parsePIDErr := base64.StdEncoding.DecodeString(msg.PID64)
		if parsePIDErr != nil {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		copy(pid64Bytes, msg.PID[:])
	}

	// validate..
	isValid, validErr := validateMsg(msgPtr)
	if validErr != nil {
		log.Printf("ERROR validating msg: %s\n", validErr)
	}
	if !isValid {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// calc id..
	msg.ID = sha256.Sum256(body)

	// store..
	storeErr := tstdout.Store(msgPtr)
	if storeErr != nil {
		log.Printf("ERROR Failed to store msg: %s\n", storeErr)
		http.Error(w, "Failed to store message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func validateMsg(msg *tstore.Msg) (bool, error) {

	// TODO ensure no attempt to provide id - we calculate it to ensure integrity..
	// if msg.ID != nil {
	// 	return false, errors.New("id cannot be provided")
	// }

	// if has pid verify exists..
	if msg.PID64 != "" {
		exists, err := tstdout.ParentExists(msg)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, errors.New("pid not found")
		}
	}

	// TODO:
	// verify from and to addrs
	// verify time
	// verify attachments

	return true, nil
}