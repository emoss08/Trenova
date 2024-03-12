package ws

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"trenova/app/middleware"
)

type Client struct {
	id  string
	hub *Hub

	conn *websocket.Conn
	send chan []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	pongWait       = 60 * time.Second
	maxMessageSize = 512
)

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	userID, ok := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

	if !ok {
		log.Println("user id not found")
		return
	}

	client := &Client{
		id:   userID.String(),
		hub:  hub,
		conn: conn,
		send: make(chan []byte),
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		err := c.conn.Close()
		if err != nil {
			return
		}

		c.hub.unregister <- c
	}()

	c.conn.SetReadLimit(maxMessageSize)
	err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		return
	}
	c.conn.SetPongHandler(func(appData string) error {
		err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			return err
		}

		return nil
	})

	for {
		_, text, err := c.conn.ReadMessage()
		log.Printf("message: %s", text)

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}

			break
		}

		msg := &WSMessage{}
		reader := bytes.NewReader(text)
		decoder := json.NewDecoder(reader)
		err = decoder.Decode(msg)
		if err != nil {
			log.Printf("error: %v", err)
			break
		}
		c.hub.broadcast <- &Message{ClientID: c.id, Text: string(text)}
	}
}

func (c *Client) writePump() {}
