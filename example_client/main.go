package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type KeyPair struct {
	PublicKey  rsa.PublicKey
	PrivateKey rsa.PrivateKey
}

type Client struct {
	Key string `json "key"`
	Id  string `json "id"`
}

type Message struct {
	Sender string `json "sender"`
	Body   string `json "body"`
}

func PrintCommandHelp() {
	fmt.Println("Argument syntax:")
	fmt.Println("go run main.go send MESSAGE_BODY RECIPIENT_ID SENDER_ID")
	fmt.Println("go run main.go receive RECIPIENT_ID")
	fmt.Println("go run main.go newuser CLIENT_ID")
}

var SERVER_URL string
var SERVER_PORT string

//Just parse the arguments and decide what execution path to follow
func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		PrintCommandHelp()
		return
	}

	config := GetConfig()

	if url, ok := config["host"]; !ok {
		log.Fatal("client.conf missing host parameter")
	} else {
		SERVER_URL = url
	}

	if port, ok := config["port"]; !ok {
		log.Fatal("client.conf missing port parameter")
	} else {
		SERVER_PORT = port
	}

	switch args[0] {
	case "receive":
		CheckAndReceive(args[1:])
		break
	case "send":
		Send(args[1:])
		break
	case "newuser":
		GenerateKeyPair()
		RegisterClient(args[1:])
		break
	default:
		PrintCommandHelp()
		break
	}
}

func GetConfig() map[string]string {
	var config map[string]string
	data, err := ioutil.ReadFile("client.conf")

	if err != nil {
		log.Fatal("Could not read client.conf")
	}

	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatal("client.conf not in valid JSON format")
	}

	return config
}

//Makes a post request
func MakePostRequest(body []byte, path string) []byte {

	/*
		DO NOT USE THIS IN PRODUCTION:
		Allows this client to be used with a cryptic server whose SSL certs have not been verified
	*/
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	hc := http.Client{Transport: tr}
	fullPath := "https://" + SERVER_URL + ":" + SERVER_PORT + path
	req, err := http.NewRequest("POST", fullPath, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		log.Fatal("Error constructing request")
	}

	response, err := hc.Do(req)

	if err != nil {
		log.Fatal("Error contacting server")
	}

	defer response.Body.Close()
	content, err := ioutil.ReadAll(response.Body)

	return content
}

//Parses interfaces into JSON
func GenerateBody(i interface{}) []byte {
	b, err := json.Marshal(i)

	if err != nil {
		log.Fatal(err.Error())
	}

	return b
}

//Register a new client with the cryptic messaging server
func RegisterClient(args []string) {
	kp := ReadKeyPair()
	pubKey := x509.MarshalPKCS1PublicKey(&kp.PublicKey)

	params := Client{
		Id:  args[0],
		Key: base64.StdEncoding.EncodeToString(pubKey),
	}

	body := GenerateBody(params)

	MakePostRequest(body, "/register")

	log.Println("Client successfully registered with id: " + params.Id)
}

func CheckAndReceive(args []string) {
	kp := ReadKeyPair()

	id := args[0]

	var params map[string]string = make(map[string]string)
	params["id"] = id

	body := GenerateBody(params)
	response := MakePostRequest(body, "/getMessages")

	var messages []Message

	if err := json.Unmarshal(response, &messages); err != nil {
		log.Fatal("Could not decode response from server " + err.Error())
	}

	num_messages := len(messages)

	for i := 0; i < num_messages; i++ {
		sender := messages[i].Sender
		encrypted_body_b64 := messages[i].Body
		encrypted_body, err := base64.StdEncoding.DecodeString(encrypted_body_b64)
		if err != nil {
			log.Printf("Encountered an error whilst decoding message: %s\n", err.Error())
			continue
		}
		decrypted_body, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, &kp.PrivateKey, encrypted_body, []byte("cryptic_msg"))

		if err != nil {
			log.Printf("Encountered an error whilst decrypting message: %s\n", err.Error())
		} else {
			log.Println("New message from " + sender + ": " + string(decrypted_body))
		}
	}
}

func Send(args []string) {
	//First get the key for the recipient_id
	recipients := []string{args[1]}

	body := GenerateBody(recipients)
	response := MakePostRequest(body, "/getKeyById")

	var id_map map[string]string

	if err := json.Unmarshal(response, &id_map); err != nil {
		log.Fatal("Could not decode response from server " + err.Error())
	}

	pub_key_b64_encoded := id_map[args[1]]
	pub_key_encoded, err := base64.StdEncoding.DecodeString(pub_key_b64_encoded)

	if err != nil {
		log.Fatal("Encountered an error whilst decoding recipient public key: " + err.Error())
	}

	pub_key, err := x509.ParsePKCS1PublicKey(pub_key_encoded)

	if err != nil {
		log.Fatal("Encountered an error whilst decoding recipient public key: " + err.Error())
	}

	//Encrypt the message using the recipient's public key
	msg_body, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub_key, []byte(args[0]), []byte("cryptic_msg"))

	if err != nil {
		log.Fatal("Encountered an error whilst encrypting message: " + err.Error())
	}

	//Send the encrypted message
	msg_body_str := base64.StdEncoding.EncodeToString(msg_body)
	var message_params map[string][]string = make(map[string][]string)
	message_params["sender"] = []string{args[2]}
	message_params[args[1]] = []string{msg_body_str}

	body = GenerateBody(message_params)
	MakePostRequest(body, "/sendMessage")
}

func ReadKeyPair() KeyPair {
	kp := KeyPair{}
	data, err := ioutil.ReadFile("client_priv.pem")

	if err != nil {
		log.Fatal("Unable to read private key file")
	}

	priv_key_block, _ := pem.Decode(data)
	priv_key, err := x509.ParsePKCS1PrivateKey(priv_key_block.Bytes)

	kp.PrivateKey = *priv_key

	if err != nil {
		log.Fatal("Unable to parse private key")
	}

	data, err = ioutil.ReadFile("client_pub.pem")

	if err != nil {
		log.Fatal("Unable to read public key file")
	}

	pub_key_block, _ := pem.Decode(data)

	pub_key, err := x509.ParsePKCS1PublicKey(pub_key_block.Bytes)
	kp.PublicKey = *pub_key
	if err != nil {
		log.Fatal("Unable to parse private key")
	}

	return kp
}

func GenerateKeyPair() {
	reader := rand.Reader
	key, err := rsa.GenerateKey(reader, 2048)
	publicKey := key.PublicKey
	if err != nil {
		log.Println("Error generating key " + err.Error())
	}

	outfile, err := os.Create("client_priv.pem")

	if err != nil {
		log.Println("Error creating PEM " + err.Error())
	}

	defer outfile.Close()

	priv_block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err = pem.Encode(outfile, priv_block)

	if err != nil {
		log.Println("Error saving PEM " + err.Error())
	}

	outfile, err = os.Create("client_pub.pem")

	if err != nil {
		log.Println("Error creating PEM " + err.Error())
	}

	defer outfile.Close()

	pub_key_bytes := x509.MarshalPKCS1PublicKey(&publicKey)

	if err != nil {
		log.Println("Error encoding public key " + err.Error())
	}
	pub_block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pub_key_bytes,
	}

	err = pem.Encode(outfile, pub_block)

	if err != nil {
		log.Println("Error saving PEM " + err.Error())
	}
}
