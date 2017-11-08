package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var (
	_                         = godotenv.Load()
	facebookVerificationToken = os.Getenv("FB_VERIFICATION_TOKEN")
	facebookAccessToken       = os.Getenv("FB_ACCESS_TOKEN")
	googleVerificationKey     = os.Getenv("GOOGLE_API_KEY")
	facebookEndpoint          = "https://graph.facebook.com/v2.6/me/messages?access_token=" + facebookAccessToken
)

func main() {
	http.HandleFunc("/messenger", routeMessage)
	http.ListenAndServe(":8080", nil)
}

func routeMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET": // The only GET method facebook will send us, is for the verification challenge.
		verifyFacebookChallenge(w, r)
	case "POST":
		handleFBPostRequest(w, r)
	default:
		respondBadRequest(w, "Method not supported")
	}
}

func handleFBPostRequest(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(standardResponse{"ok"})
	ss := fbMessage{}
	ss.ID, ss.Kind, ss.Payload = parseFBRequest(r)

	if ss.Kind == "invalid" {
		return
	}

	sendFBPayload(ss.compose())
}

func respondBadRequest(w http.ResponseWriter, m string) {
	w.WriteHeader(http.StatusBadRequest)
	res := standardResponse{m}
	json.NewEncoder(w).Encode(res)
}
