package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/fiatjaf/eventstore/sqlite3"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
	"github.com/nbd-wtf/go-nostr"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

type RequestPayload struct {
	Image string `json:"image"`
}

type ResponsePayload struct {
	Result interface{} `json:"result"`
}

var (
	relay = khatru.NewRelay()
)

func main() {

	relay.Info.Name = "CATSTRR"
	relay.Info.PubKey = "f1f9b0996d4ff1bf75e79e4cc8577c89eb633e68415c7faf74cf17a07bf80bd8"
	relay.Info.Description = "A relay accepting only notes with cat pictures 🐱"
	relay.Info.Icon = "https://image.nostr.build/aadc540f6d6d0a6afeb2d97b78e2961aa55a6ea344ba347791ab09413c874e3a.jpg"

	db := sqlite3.SQLite3Backend{DatabaseURL: "./db/db"}
	if err := db.Init(); err != nil {
		panic(err)
	}

	relay.RejectEvent = append(relay.RejectEvent,
		policies.EventIPRateLimiter(5, time.Minute*1, 30),
		policies.RejectEventsWithBase64Media,
	)

	relay.RejectFilter = append(relay.RejectFilter,
		policies.NoEmptyFilters,
		policies.NoComplexFilters,
	)

	relay.RejectConnection = append(relay.RejectConnection,
		policies.ConnectionRateLimiter(10, time.Minute*2, 30),
	)

	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)
	relay.RejectEvent = append(relay.RejectEvent,
		func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
			imageURLPattern := `(?i)(https?://[^\s]+(\.jpg|\.jpeg|\.png|\.gif|\.bmp|\.svg|\.webp|\.tiff))`
			regex := regexp.MustCompile(imageURLPattern)

			imageUrl := regex.FindString(event.Content)

			if imageUrl == "" {
				catEmojisRegex := regexp.MustCompile(`[\u{1F408}\u{1F431}\u{1F63A}\u{1F638}\u{1F639}\u{1F63B}\u{1F63C}\u{1F63D}\u{1F640}]`)
				catWordRegex := regexp.MustCompile(`\bcat(s?)\b`)

				if catEmojisRegex.MatchString(event.Content) || catWordRegex.MatchString(event.Content) {
					return false, ""
				}

				return true, "No cat emoji or word found."
			}

			if !isCatImage(imageUrl) {
				return true, "Not a cat image"
			}
			return false, ""
		})

	fmt.Println("running on :3388")

	http.ListenAndServe(":3388", relay)
}

func isCatImage(imageUrl string) bool {
	payload := RequestPayload{
		Image: imageUrl,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return false
	}

	resp, err := http.Post("http://@node:3003/process-image", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return false
	}

	// Parse the response JSON
	var responsePayload ResponsePayload
	err = json.Unmarshal(body, &responsePayload)
	if err != nil {
		fmt.Printf("Error parsing JSON response: %v\n", err)
		return false
	}

	// Print the result from the Node.js worker
	fmt.Printf("Result from node.js: %+v\n", responsePayload.Result)
	return responsePayload.Result == true
}
