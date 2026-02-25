package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

type WSEventType string

const (
	WSEventTypeJoin   WSEventType = "join"
	WSEventTypeLeave  WSEventType = "leave"
	WSEventTypeRoll   WSEventType = "roll"
	WSEventTypeChat   WSEventType = "chat"
	WSEventTypeSystem WSEventType = "system"
)

// WebSocket событие
type WSEvent struct {
	Type      WSEventType `json:"type"`
	TableID   int64       `json:"table_id"`
	UserID    int64       `json:"user_id"`
	Username  string      `json:"username,omitempty"`
	Content   any         `json:"content"`
	Timestamp int64       `json:"timestamp"`
}

// Преобразование в строку для хранения в БД
func (e *WSEvent) ToDBEvent() (*TableEvent, error) {
	contentBytes, err := json.Marshal(e.Content)
	if err != nil {
		return nil, err
	}

	metadata := map[string]any{
		"username": e.Username,
	}
	metadataBytes, _ := json.Marshal(metadata)

	return &TableEvent{
		TableID:   e.TableID,
		EventType: string(e.Type),
		UserID:    e.UserID,
		Content:   string(contentBytes),
		Metadata:  string(metadataBytes),
		CreatedAt: time.Unix(0, e.Timestamp*int64(time.Millisecond)),
	}, nil
}

// Создание события из строки (для чтения из БД)
func EventFromDBEvent(dbEvent *TableEvent) (*WSEvent, error) {
	var content any
	if err := json.Unmarshal([]byte(dbEvent.Content), &content); err != nil {
		content = dbEvent.Content // если не JSON, используем как есть
	}

	var metadata map[string]any
	json.Unmarshal([]byte(dbEvent.Metadata), &metadata)

	username, _ := metadata["username"].(string)

	return &WSEvent{
		Type:      WSEventType(dbEvent.EventType),
		TableID:   dbEvent.TableID,
		UserID:    dbEvent.UserID,
		Username:  username,
		Content:   content,
		Timestamp: dbEvent.CreatedAt.UnixNano() / int64(time.Millisecond),
	}, nil
}

// Создание форматированного текстового представления события
func (e *WSEvent) FormatText() string {
	switch e.Type {
	case WSEventTypeJoin:
		return fmt.Sprintf("✨ **%s** присоединился к столу", e.Username)
	case WSEventTypeLeave:
		return fmt.Sprintf("👋 **%s** покинул стол", e.Username)
	case WSEventTypeRoll:
		if roll, ok := e.Content.(map[string]any); ok {
			dice, _ := roll["dice"].(string)
			result, _ := roll["result"].(float64)
			return fmt.Sprintf("🎲 **%s** бросил %s → **%d**", e.Username, dice, int(result))
		}
		return fmt.Sprintf("🎲 **%s** сделал бросок", e.Username)
	case WSEventTypeChat:
		if text, ok := e.Content.(string); ok {
			return fmt.Sprintf("💬 **%s**: %s", e.Username, text)
		}
		if textMap, ok := e.Content.(map[string]any); ok {
			if text, ok := textMap["text"].(string); ok {
				return fmt.Sprintf("💬 **%s**: %s", e.Username, text)
			}
		}
		return fmt.Sprintf("💬 **%s**: %v", e.Username, e.Content)
	case WSEventTypeSystem:
		return fmt.Sprintf("ℹ️ %v", e.Content)
	default:
		return fmt.Sprintf("[%s] %v", e.Type, e.Content)
	}
}
