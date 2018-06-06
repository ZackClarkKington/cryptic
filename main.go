package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type Group struct {
	id      string             `json "id"`
	clients map[string]*Client `json "clients"`
}

type Client struct {
	Key         []byte `json "key"`
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

	var keys map[string][]byte = make(map[string][]byte)

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

func SendMessages(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		PostParameterError(w)
		return
	}

	var messages map[string][]string
	var sender string

	if err = json.Unmarshal(body, &messages); err != nil {
		InvalidJSONError(w)
		return
	}

	if sender_str, ok := messages["sender"]; ok {
		sender = sender_str[0]
	} else {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err.Error())
		w.Write([]byte("Invalid request, no sender id given"))
		return
	}

	for k, v := range messages {
		if k == "sender" {
			continue
		} else {
			if client, ok := clients[k]; ok {
				go Append(sender, v, client.MessageBuff)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Success"))
}

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
