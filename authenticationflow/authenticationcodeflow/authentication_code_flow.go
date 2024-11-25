package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequest("POST", "http://localhost:8080/realms/fullcycle/protocol/openid-connect/auth", nil)
		if err != nil {
			fmt.Printf("error when creating request %v\n", err)
			os.Exit(1)
		}
		q := req.URL.Query()
		q.Add("client_id", "fullcycle-client")
		q.Add("redirect_uri", "http://localhost:8000/callback")
		q.Add("response_type", "code")
		q.Add("scope", "openid")
		req.URL.RawQuery = q.Encode()
		http.Redirect(w, r, req.URL.String(), http.StatusFound)
	})
	mux.HandleFunc("GET /callback", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequest("POST", "http://localhost:8080/realms/fullcycle/protocol/openid-connect/token", nil)
		if err != nil {
			if _, errWrite := fmt.Fprintf(w, "error %v\n", err); errWrite != nil {
				log.Printf("error when trying to write into the client %v\n", errWrite)
			}
			return
		}
		defer func() {
			if errClose := req.Body.Close(); errClose != nil {
				log.Printf("error when trying to close request's body %v\n", errClose)
			}
			return
		}()
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		q := req.URL.Query()
		q.Add("client_id", "fullcycle-client")
		q.Add("grant_type", "authorization_code")
		q.Add("code", r.URL.Query().Get("code"))
		q.Add("redirect_uri", "http://localhost:8000/callback")
		reader := bytes.NewReader([]byte(q.Encode()))
		req.Body = io.NopCloser(reader)
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			if _, errWrite := fmt.Fprintf(w, "invalid url: %v\n", err); errWrite != nil {
				log.Printf("error when trying to write into the client %v\n", errWrite)
			}
			return
		}
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			if _, errWrite := fmt.Fprintf(w, "error reading response's body from keycloak %v\n", err); errWrite != nil {
				log.Printf("error when trying to write into the client %v\n", errWrite)
			}
		}
		var jsonData interface{}
		if errUnmarshal := json.Unmarshal(respBody, &jsonData); errUnmarshal != nil {
			log.Printf("error unmarshalling response body: %v\n", errUnmarshal)
			http.Error(w, "invalid JSON response", http.StatusInternalServerError)
			return
		}
		defer func() {
			if errClose := resp.Body.Close(); errClose != nil {
				log.Printf("error when trying to close request's body %v\n", errClose)
			}
			return
		}()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "    ")
		if errWrite := encoder.Encode(jsonData); errWrite != nil {
			log.Printf("error when trying to write into the client %v\n", errWrite)
		}
	})
	fmt.Println("server started at http://localhost:8000/login")
	log.Fatalln(http.ListenAndServe(":8000", mux))
}
