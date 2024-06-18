package gcm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

const (
	// FCMSendEndpoint is the endpoint for sending message to the Firebase Cloud Messaging (FCM) server.
	// See more on https://firebase.google.com/docs/cloud-messaging/server
	FCMSendEndpoint = "https://fcm.googleapis.com/fcm/send"
)

const (
	// fcmPushPriorityHigh and fcmPushPriorityNormal is priority of a delivery message options
	// See more on https://firebase.google.com/docs/cloud-messaging/concept-options?hl=en#setting-the-priority-of-a-message
	fcmPushPriorityHigh   = "high"
	fcmPushPriorityNormal = "normal"
)

const (
	// maxRegistrationIDs are max number of registration IDs in one message.
	maxRegistrationIDs = 1000

	// maxTimeToLive is max time FCM storage can store messages when the device is offline
	maxTimeToLive = 2419200 // 4 weeks
)

// Client abstracts the interaction between the application server and the
// FCM server. The developer must obtain an API key from the Google APIs
// Console page and pass it to the Client so that it can perform authorized
// requests on the application server's behalf. To send a message to one or
// more devices use the Client's Send methods.
type Client struct {
	ApiKey string
	URL    string
	Http   *http.Client
}

// NewClient returns a new sender with the given URL and apiKey.
// If one of input is empty or URL is malformed, returns error.
// It sets http.DefaultHTTP client for http connection to server.
// If you need our own configuration overwrite it.
func NewClient(urlString, apiKey string) (*Client, error) {
	if len(urlString) == 0 {
		return nil, fmt.Errorf("missing FCM endpoint url")
	}

	if len(apiKey) == 0 {
		return nil, fmt.Errorf("missing API Key")
	}

	if _, err := url.Parse(urlString); err != nil {
		return nil, fmt.Errorf("failed to parse URL %q: %s", urlString, err)
	}

	return &Client{
		URL:    urlString,
		ApiKey: apiKey,
		Http:   http.DefaultClient,
	}, nil
}

func (c *Client) SendFix(token string, title string, message string, keyPath string, projectID string) (*Response, error) {
	err := sendFCMNotification(token, title, message, keyPath, projectID)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *Client) Send(msg *Message) (*Response, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(msg); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.URL, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("key=%s", c.ApiKey))
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code %d: %s", resp.StatusCode, resp.Status)
	}

	var response Response
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return nil, err
	}

	return &response, err
}

func sendFCMNotification(token string, title string, body string, keyPath string, projectID string) error {
	ctx := context.Background()

	// Firebase Admin SDKの設定ファイルを指定
	opt := option.WithCredentialsFile(keyPath)

	config := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		return fmt.Errorf("error initializing app: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return fmt.Errorf("error getting Messaging client: %v", err)
	}

	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	fmt.Printf("Successfully sent message: %s\n", response)
	return nil
}
