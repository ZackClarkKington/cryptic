# Cryptic

API:

/register - Register a new client, parameters should be formatted as follows:

    {"key":"CLIENT_PUBLIC_KEY", "id":"CLIENT_ID"}
  
  /getKeyById - Get the public key for a client id or set of client id's, parameters should be formatted as follows:
  

    ["CLIENT_ID"]
Response will follow the format:

    {"CLIENT_ID":"CLIENT_PUBLIC_KEY"}

/sendMessage - Send a batch of messages to a set of client id's, parameters should be formatted as follows:

    {"sender": "SENDER_ID", "RECIPIENT_ID": ["MESSAGE"]}

/getMessages - Get messages buffered for a client id, once messages have been returned from this endpoint they are burned on the server, parameters should be formatted as follows:

    {"id":"CLIENT_ID"}

