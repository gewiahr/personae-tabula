package websocket

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"personae-tabula/internal/domain"

	"github.com/gorilla/websocket"
)

func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, r, err := c.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error from user %d: %v", c.userID, err)
			}
			break
		}

		var wsEvent domain.WSEvent[*domain.TableWSEvent[any]]
		if err := json.NewDecoder(r).Decode(&wsEvent); err != nil {
			log.Printf("JSON decode error from user %d: %v", c.userID, err)
			c.sendError(err, "invalid message format")
			continue
		}

		// Enrich with client info
		wsEvent.TableID = c.tableID
		wsEvent.UserID = c.userID
		wsEvent.Timestamp = time.Now().Unix()

		// Process asynchronously with timeout
		go c.processTableEvent(&wsEvent)
	}
}

func (c *WSClient) processTableEvent(event *domain.WSEvent[*domain.TableWSEvent[any]]) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	c.hub.handleTableEvent(ctx, event)
}

func (c *WSClient) sendError(err error, msg string) {
	// errEvent := &domain.WSEvent[*domain.SystemWSEvent[any]]{
	// 	Type: domain.WSEventTypeSystem,
	// 	Event: &domain.SystemWSEvent[any]{
	// 		Type: domain.SystemWSEventTypeError,
	// 		Content: &WSError{
	// 			Error:   err,
	// 			Message: msg,
	// 		},
	// 	},
	// 	UserID:    c.userID,
	// 	TableID:   c.tableID,
	// 	Timestamp: time.Now().Unix(),
	// }
	//c.conn.WriteJSON(errEvent)
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
