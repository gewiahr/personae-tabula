package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	tableID int64
	userID  int64
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type WSError struct {
	Error   error  `json:"error"`
	Message string `json:"message"`
}

// type WSEventType string

// const (
// 	WSEventTypeJoin   WSEventType = "join"
// 	WSEventTypeLeave  WSEventType = "leave"
// 	WSEventTypeRoll   WSEventType = "roll"
// 	WSEventTypeChat   WSEventType = "chat"
// 	WSEventTypeSystem WSEventType = "system"
// )

// type WSEvent[T any] struct {
// 	Type      WSEventType `json:"type"`
// 	Content   T           `json:"content"`
// 	UserID    int64       `json:"userId"`
// 	TableID   int64       `json:"tableId"`
// 	Timestamp int64       `json:"timestamp"`
// }
