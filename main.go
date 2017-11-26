package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/api/googleapi/transport"
	vision "google.golang.org/api/vision/v1"
)

var (
	_                         = godotenv.Load()
	facebookVerificationToken = os.Getenv("FB_VERIFICATION_TOKEN")
	facebookAccessToken       = os.Getenv("FB_ACCESS_TOKEN")
	googleVerificationKey     = os.Getenv("GOOGLE_API_KEY")
	facebookEndpoint          = "https://graph.facebook.com/v2.6/me/messages?access_token=" + facebookAccessToken
	port                      = ":" + os.Getenv("PORT")
)

func main() {
	fmt.Println("Starting engine...")
	http.HandleFunc("/messenger", routeMessage)
	http.HandleFunc("/health", routeMessage)
	http.ListenAndServe(port, nil)
}

func displayHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	res := standardResponse{"ok"}
	json.NewEncoder(w).Encode(res)
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

	if ss.Kind == "text" {
		if isCommand(ss.Payload) {
			execCommand(ss)
			return
		}

		ss.Payload = randomStandardResponse()
	}

	if ss.Kind == "image" {
		ss.Kind = "text"
		imagePath, err := saveImage(ss.Payload)
		if err != nil {
			fmt.Println(err)
		}
		res := sendToGV(imagePath)

		vl := []visionLabels{}
		_ = json.Unmarshal(res, &vl)

		for _, value := range vl {
			ss.Payload = "No ferret!"

			if value.Description == "ferret" {
				ss.Payload = "Ferret! I see a ferret!"
				break
			}
		}

	}

	sendFBPayload(ss.compose())
}

func respondBadRequest(w http.ResponseWriter, m string) {
	w.WriteHeader(http.StatusBadRequest)
	res := standardResponse{m}
	json.NewEncoder(w).Encode(res)
}

func isCommand(c string) bool {
	switch c {
	case "help", "ayuda":
		return true
	}
	return false
}

func execCommand(ss fbMessage) {
	switch ss.Payload {
	case "help", "ayuda":
		ss.Payload = "I'm a ferretbot!"
		sendFBPayload(ss.compose())
		ss.Payload = "You can find my sourcecode in https://github.com/osuka42/lazy-ferret"
	}
	sendFBPayload(ss.compose())
}

func randomStandardResponse() string {
	responses := []string{
		"Show me the ferrets!",
		"I don't care! Show me the ferrets.",
		"The ferrets!!",
		"Zzz...",
		"Too much talk and no ferrets!",
	}
	return responses[rand.Intn(5)]
}

func createDownloadsDir() {
	dir := "downloads"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, os.ModePerm)
	}
}

func saveImage(URL string) (filepath string, err error) {
	p, _ := url.ParseRequestURI(URL)
	ext := path.Ext(strings.Split(p.RequestURI(), "?")[0]) // Get extension of the file, without downloading yet
	now := int(time.Now().Unix())
	filepath = "./downloads/" + strconv.Itoa(now) + ext

	createDownloadsDir()
	img, err := os.Create(filepath)
	resp, err := http.Get(URL)
	w, err := io.Copy(img, resp.Body)
	if err != nil {
		return
	}

	fmt.Println("Saved " + filepath + " " + strconv.Itoa(int(w)) + "bytes")
	return
}

func sendToGV(f string) []byte {
	data, err := ioutil.ReadFile(f)

	enc := base64.StdEncoding.EncodeToString(data)
	img := &vision.Image{Content: enc}

	feature := &vision.Feature{
		Type:       "LABEL_DETECTION",
		MaxResults: 10,
	}

	req := &vision.AnnotateImageRequest{
		Image:    img,
		Features: []*vision.Feature{feature},
	}

	batch := &vision.BatchAnnotateImagesRequest{
		Requests: []*vision.AnnotateImageRequest{req},
	}

	client := &http.Client{
		Transport: &transport.APIKey{Key: "AIzaSyDZnbKlOILDA-JEUZ_oQPmF34xrheMCwxk"},
	}
	svc, err := vision.New(client)
	if err != nil {
		log.Fatal(err)
	}
	res, err := svc.Images.Annotate(batch).Do()
	if err != nil {
		log.Fatal(err)
	}

	body, err := json.Marshal(res.Responses[0].LabelAnnotations)
	return body
}

type visionLabels struct {
	Description string  `json:"description"`
	Mid         string  `json:"mid"`
	Score       float64 `json:"score"`
}
