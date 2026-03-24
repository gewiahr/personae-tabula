package domain

type WSEventType string

const (
	WSEventTypeTable  WSEventType = "table"
	WSEventTypeChat   WSEventType = "chat"
	WSEventTypeSystem WSEventType = "system"
)

type WSEvent[T any] struct {
	Type      WSEventType `json:"type"`
	Event     T           `json:"event"`
	UserID    int64       `json:"userId"`
	TableID   int64       `json:"tableId"`
	Timestamp int64       `json:"timestamp"`
}

type TableWSEventType string

const (
	TableWSEventTypeRoll         TableWSEventType = "roll"
	TableWSEventTypeCharacterAdd TableWSEventType = "char_add"
	TableWSEventTypeCharacterRm  TableWSEventType = "char_rm"
)

type TableWSEvent[T any] struct {
	Id      int64            `json:"id"`
	Type    TableWSEventType `json:"type"`
	Content T                `json:"content"`
}

type SystemWSEventType string

const (
	SystemWSEventTypeJoin  SystemWSEventType = "join"
	SystemWSEventTypeLeave SystemWSEventType = "leave"
	SystemWSEventTypeError SystemWSEventType = "error"
)

type SystemWSEvent[T any] struct {
	Type    SystemWSEventType `json:"type"`
	Content T                 `json:"content"`
}

type RollRequestEvent struct {
	Type    string      `json:"type"`
	Formula RollFormula `json:"formula"`
	Title   string      `json:"title"`
	Roller  string      `json:"roller"`
}

type RollFormula struct {
	Rolls []DiceRoll `json:"rolls"`
	Mod   int        `json:"mod"`
}

type DiceRoll struct {
	Dice   int `json:"dice"`
	Amount int `json:"amount"`
}

type ServiceEvent struct {
}

// Преобразование в строку для хранения в БД
// func (e *WSEvent) ToDBEvent() (*TableEvent, error) {
// 	contentBytes, err := json.Marshal(e.Content)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// metadata := map[string]any{
// 	// 	"username": e.Username,
// 	// }
// 	// metadataBytes, _ := json.Marshal(metadata)

// 	return &TableEvent{
// 		TableID:   e.TableID,
// 		EventType: string(e.Type),
// 		UserID:    e.UserID,
// 		Content:   string(contentBytes),
// 		Metadata:  "", //string(metadataBytes),
// 		CreatedAt: time.Unix(0, e.Timestamp*int64(time.Millisecond)),
// 	}, nil
// }

// // Создание события из строки (для чтения из БД)
// func EventFromDBEvent(dbEvent *TableEvent) (*WSEvent, error) {
// 	var content any
// 	if err := json.Unmarshal([]byte(dbEvent.Content), &content); err != nil {
// 		content = dbEvent.Content // если не JSON, используем как есть
// 	}

// 	var metadata map[string]any
// 	json.Unmarshal([]byte(dbEvent.Metadata), &metadata)

// 	//username, _ := metadata["username"].(string)

// 	return &WSEvent{
// 		Type:      WSEventType(dbEvent.EventType),
// 		TableID:   dbEvent.TableID,
// 		UserID:    dbEvent.UserID,
// 		Content:   content,
// 		Timestamp: dbEvent.CreatedAt.UnixNano() / int64(time.Millisecond),
// 	}, nil
// }

// // Создание форматированного текстового представления события
// func (e *WSEvent) FormatText() string {
// 	switch e.Type {
// 	case WSEventTypeJoin:
// 		return fmt.Sprintf("✨ **%d** присоединился к столу", e.UserID)
// 	case WSEventTypeLeave:
// 		return fmt.Sprintf("👋 **%d** покинул стол", e.UserID)
// 	case WSEventTypeRoll:
// 		if roll, ok := e.Content.(map[string]any); ok {
// 			dice, _ := roll["dice"].(string)
// 			result, _ := roll["result"].(float64)
// 			return fmt.Sprintf("🎲 **%d** бросил %s → **%d**", e.UserID, dice, int(result))
// 		}
// 		return fmt.Sprintf("🎲 **%d** сделал бросок", e.UserID)
// 	case WSEventTypeChat:
// 		if text, ok := e.Content.(string); ok {
// 			return fmt.Sprintf("💬 **%d**: %s", e.UserID, text)
// 		}
// 		if textMap, ok := e.Content.(map[string]any); ok {
// 			if text, ok := textMap["text"].(string); ok {
// 				return fmt.Sprintf("💬 **%d**: %s", e.UserID, text)
// 			}
// 		}
// 		return fmt.Sprintf("💬 **%d**: %v", e.UserID, e.Content)
// 	case WSEventTypeSystem:
// 		return fmt.Sprintf("ℹ️ %v", e.Content)
// 	default:
// 		return fmt.Sprintf("[%s] %v", e.Type, e.Content)
// 	}
// }
