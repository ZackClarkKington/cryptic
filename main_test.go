package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func MakePostRequest(body []byte, path string, t *testing.T, h http.HandlerFunc) httptest.ResponseRecorder {
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	return *rr
}

func GenerateBody(i interface{}, t *testing.T) []byte {
	b, err := json.Marshal(i)

	if err != nil {
		t.Fatal(err)
	}

	return b
}

func RegisterTestClient(testClient Client, t *testing.T) Client {
	body := GenerateBody(testClient, t)
	MakePostRequest(body, "/register", t, RegistrationHandler)
	registered_client, ok := clients[testClient.Id]

	if !ok {
		t.Error("Client has not been registered")
	}

	return registered_client
}

func TestRegistrationHandler(t *testing.T) {
	initClients()
	testClient := Client{
		Key: "This-is-a-key",
		Id:  "test-id"}

	registered_client := RegisterTestClient(testClient, t)
	if registered_client.Id != testClient.Id {
		t.Errorf("Client has been assigned wrong id: got %v want %v", registered_client.Id, testClient.Id)
	}

	if registered_client.Key != testClient.Key {
		t.Errorf("Client has been assigned wrong key: got %v want %v", registered_client.Key, testClient.Key)
	}
}

func TestGetKeyById(t *testing.T) {
	initClients()
	testClient := Client{
		Key: "This-is-a-key",
		Id:  "test-id"}

	RegisterTestClient(testClient, t)

	id := []string{"test-id"}

	body := GenerateBody(id, t)

	rr := MakePostRequest(body, "getKeyById", t, GetKeyById)
	expected := "{\"test-id\":\"This-is-a-key\"}"
	if rr.Body.String() != expected {
		t.Errorf("Returned wrong key for client: got %v want %v", rr.Body.String(), expected)
	}
}

func SendTestMessages(sender_id string, recipient_id string, message_strings []string, t *testing.T) {
	var messages map[string][]string = make(map[string][]string)
	messages[recipient_id] = message_strings
	messages["sender"] = []string{sender_id}

	body := GenerateBody(messages, t)
	MakePostRequest(body, "/sendMessage", t, SendMessages)
}

func TestSendMessages(t *testing.T) {
	initClients()
	testClient := Client{
		Key: "This-is-a-key",
		Id:  "test-id"}

	testClient1 := Client{
		Key: testClient.Key,
		Id:  "test-id-1"}

	message_strings := []string{"test-message", "another-test-message"}

	RegisterTestClient(testClient, t)
	RegisterTestClient(testClient1, t)
	SendTestMessages(testClient1.Id, testClient.Id, message_strings, t)
	time.Sleep(time.Millisecond * 100)
	//Messages are added asynchronously so we need to wait for 100ms
	num_received := clients[testClient.Id].MessageBuff.messages.Length()
	if num_received != len(message_strings) {
		t.Errorf("Not enough messages received: got %v want %v", num_received, len(message_strings))
	}

	for i := 0; i < num_received; i++ {
		msg := Pop(clients[testClient.Id].MessageBuff)
		if msg.Sender != testClient1.Id {
			t.Errorf("Returned wrong sender for message: got %v want %v", msg.Sender, testClient1.Id)
		}

		if msg.Body != message_strings[i] {
			t.Errorf("Returned wrong message body: got %v want %v", msg.Body, message_strings[i])
		}
	}
}

func TestGetMessagesForId(t *testing.T) {
	initClients()
	testClient := Client{
		Key: "This-is-a-key",
		Id:  "test-id"}

	testClient1 := Client{
		Key: testClient.Key,
		Id:  "test-id-1"}

	message_strings := []string{"test-message", "another-test-message"}

	RegisterTestClient(testClient, t)
	RegisterTestClient(testClient1, t)
	SendTestMessages(testClient1.Id, testClient.Id, message_strings, t)
	time.Sleep(time.Millisecond * 100)

	var params map[string]string = make(map[string]string)
	params["id"] = testClient.Id

	body := GenerateBody(params, t)
	rr := MakePostRequest(body, "/getMessages", t, GetMessagesForId)
	expected := "[{\"Sender\":\"test-id-1\",\"Body\":\"test-message\"},{\"Sender\":\"test-id-1\",\"Body\":\"another-test-message\"}]"

	if rr.Body.String() != expected {
		t.Errorf("Returned wrong messages: got %v want %v", rr.Body.String(), expected)
	}
}
