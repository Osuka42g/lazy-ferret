package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func verifyFacebookChallenge(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if len(q) != 3 {
	} else if q["hub.mode"][0] == "subscribe" && q["hub.verify_token"][0] == facebookVerificationToken {
		fmt.Fprintf(w, q["hub.challenge"][0])
		return
	}
	respondBadRequest(w, "Invalid verification token")
}

func parseFBRequest(r *http.Request) (string, string, string) {
	fb := fbRequest{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&fb)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()
	sender := fb.Entry[0].Messaging[0].Sender.SenderID
	kind := "invalid"
	payload := ""
	message := fb.Entry[0].Messaging[0].Message

	if message.Text != "" {
		kind = "text"
		payload = message.Text
	} else if len(message.Attachment) > 0 {
		kind = message.Attachment[0].Type
		payload = message.Attachment[0].Payload.URL
	}
	return sender, kind, payload
}

func sendFBPayload(p []byte) error {
	req, err := http.NewRequest("POST", facebookEndpoint, bytes.NewBuffer(p))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (fb fbMessage) compose() []byte {
	rs := fbResponse{}
	rs.Recipient.ID = fb.ID
	switch fb.Kind {
	case "text":
		rs.Message.Text = fb.Payload
	case "typing":
		rs.SenderAction = "typing_on"
	}
	payload, _ := json.Marshal(rs)
	return payload
}

type standardResponse struct {
	Message string `json:"message"`
}

type fbRequest struct {
	Entry []fbRequestEntry `json:"entry"`
}

type fbRequestEntry struct {
	Messaging []fbRequestMessaging `json:"messaging"`
}

type fbRequestMessaging struct {
	Sender struct {
		SenderID string `json:"id"`
	} `json:"sender"`
	Message struct {
		Text       string                `json:"text"`
		Attachment []fbRequestAttachment `json:"attachments"`
	} `json:"message"`
}

type fbRequestAttachment struct {
	Type    string `json:"type"`
	Payload struct {
		URL string `json:"url"`
	} `json:"payload"`
}

type fbMessage struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"` // Alias for `type`, reserved world in go
	Payload string `json:"payload"`
}

type fbSimpleText struct {
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
}

type fbTyping struct {
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	SenderAction string `json:"sender_action"`
}

type fbResponse struct {
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
	SenderAction string `json:"sender_action"`
}
