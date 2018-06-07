package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type Client struct {
	Key         string `json "key"`
	Id          string `json "id"`
	MessageBuff MessageBuffer
}

type Message struct {
	Sender string `json "sender"`
	Body   string `json "body"`
}

var clients map[string]Client

func initClients() {
	clients = make(map[string]Client)
}

func main() {
	initClients()
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/register", RegistrationHandler)
	http.HandleFunc("/getKeyById", GetKeyById)
	http.HandleFunc("/sendMessage", SendMessages)
	http.HandleFunc("/getMessages", GetMessagesForId)
	log.Fatal(http.ListenAndServeTLS(":443", "server.crt", "server.key", nil))
}

func PostParameterError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Invalid request, no post parameters could be read"))
}

func InvalidJSONError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Invalid request, could not read JSON"))
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Cryptic\n"))
}

/*
	RegistrationHandler(w, r) - Registers new clients with the server

	Parses JSON content in the POST body and populates a new client struct,
	JSON content should be formatted as such:
	{
		"key": "CLIENT_KEY",
		"id": "CLIENT_ID"
	}
*/
func RegistrationHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		PostParameterError(w)
		return
	}

	newClient := &Client{}
	if err = json.Unmarshal(body, newClient); err != nil {
		InvalidJSONError(w)
		return
	}

	newClient.MessageBuff = NewMessageBuffer()

	if _, ok := clients[newClient.Id]; !ok {
		clients[newClient.Id] = *newClient
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Success"))
}

/*
	GetKeyById(w, r) - Gets the keys for a set of client ids

	Client Ids are contained within a JSON array passed in the POST body.

	Returns JSON in the following format:
	{
		"CLIENT_ID": "CLIENT_KEY"
	}
*/
func GetKeyById(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		PostParameterError(w)
		return
	}

	var ids []string

	if err = json.Unmarshal(body, &ids); err != nil {
		InvalidJSONError(w)
		return
	}

	var keys map[string]string = make(map[string]string)
	/*
		For each specified client id:
		- Check if the client exists
		- If the client exists then add the client key to the array of keys to be returned
	*/
	for i := 0; i < len(ids); i++ {
		id := ids[i]
		if client, ok := clients[id]; ok {
			keys[id] = client.Key
		}
	}

	output, err := json.Marshal(keys)
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

/*
	SendMessages(w, r) - Sends the specified messages to the specified client ids.

	Parameters are passed as JSON formatted as so:
	{
		"RECIPIENT_ID": ["MESSAGE_BODY"],
		"sender": "SENDER_ID"
	}

	Note that multiple recipients can be sent multiple messages within this construct,
	but there can only ever be one sender.
*/
func SendMessages(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		PostParameterError(w)
		return
	}

	var recipients map[string][]string
	var sender string

	if err = json.Unmarshal(body, &recipients); err != nil {
		log.Println(err.Error())
		InvalidJSONError(w)
		return
	}
	/*
		Check that the message batch includes a sender id,
		if it doesn't then an error must be returned
	*/
	if sender_str, ok := recipients["sender"]; ok {
		sender = sender_str[0]
	} else {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err.Error())
		w.Write([]byte("Invalid request, no sender id given"))
		return
	}

	/*
		For each field in the map of recipients, first check if it is the sender field
		 - if not then append all messages for that recipient to the recipient's message
		buffer.
	*/
	for k, v := range recipients {
		if k == "sender" {
			continue
		} else {
			if client, ok := clients[k]; ok {
				/*
					Here we use a goroutine to prevent appending multiple messages from blocking the
					server responding to the sender before timeout occurs
				*/
				go Append(sender, v, client.MessageBuff)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Success"))
}

/*
	GetMessagesForId(w, r) - Gets all the messages buffered on the server for a specified client id

	Parameters to pass (in JSON):
	{
		"id": "CLIENT_ID"
	}

	Returns:
	[
		{
			"Sender": "SENDER_ID",
			"Body": "MESSAGE_BODY"
		}
	]

	Once the server has sent a buffered message to its' intended client it will no longer be stored on the server.
*/
func GetMessagesForId(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		PostParameterError(w)
		return
	}

	var params map[string]string

	if err = json.Unmarshal(body, &params); err != nil {
		InvalidJSONError(w)
		return
	}
	var messages []Message
	recipient_id := params["id"]

	/*
		First check that the requested client exists, then:
		- Create an array of Message objects
		- Pop each message in the client's buffer into this array
	*/
	if recipient, ok := clients[recipient_id]; ok {
		message_buff := recipient.MessageBuff
		num_messages := message_buff.messages.Length()
		messages = make([]Message, num_messages)
		for i := 0; i < num_messages; i++ {
			messages[i] = Pop(message_buff)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request, recipient id not found"))
		return
	}

	/*
		Encode the array of Message objects into JSON and return to the client
	*/
	var encoded_messages []byte
	encoded_messages, err = json.Marshal(messages)

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Server side error encountered, please try again later"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(encoded_messages)
}
