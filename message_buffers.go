package main

import (
	"github.com/eapache/queue"
	"sync"
)

type MessageBuffer struct {
	messages *queue.Queue
	mutex    sync.Mutex
}

func Append(sender string, batch []string, buff MessageBuffer) {
	numMessages := len(batch)

	for i := 0; i < numMessages; i++ {
		newMessage := Message{Sender: sender, Body: batch[i]}
		Put(newMessage, buff)
	}
}

func Put(m Message, buff MessageBuffer) {
	buff.mutex.Lock()
	buff.messages.Add(m)
	buff.mutex.Unlock()
}

func Pop(buff MessageBuffer) Message {
	var m Message
	buff.mutex.Lock()
	m = buff.messages.Remove().(Message)
	buff.mutex.Unlock()
	return m
}

func NewMessageBuffer() MessageBuffer {
	var newBuff MessageBuffer
	newBuff.messages = queue.New()
	newBuff.mutex = sync.Mutex{}
	return newBuff
}
