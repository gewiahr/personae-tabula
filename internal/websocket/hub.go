package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"personae-tabula/internal/domain"
	"personae-tabula/internal/repository/redis"
	"personae-tabula/internal/service"
)

type Hub struct {
	tables     map[int64]map[*WSClient]bool // tableID -> clients
	register   chan *WSClient
	unregister chan *WSClient
	broadcast  chan *domain.WSEvent
	mu         sync.RWMutex

	// Сервисы
	eventService *service.EventService
	tableService *service.TableService
	roomCache    *redis.RoomCache
}

func NewHub(
	eventService *service.EventService,
	tableService *service.TableService,
	roomCache *redis.RoomCache,
) *Hub {
	return &Hub{
		tables:       make(map[int64]map[*WSClient]bool),
		register:     make(chan *WSClient),
		unregister:   make(chan *WSClient),
		broadcast:    make(chan *domain.WSEvent, 256),
		eventService: eventService,
		tableService: tableService,
		roomCache:    roomCache,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.tables[client.tableID]; !ok {
				h.tables[client.tableID] = make(map[*WSClient]bool)
			}
			h.tables[client.tableID][client] = true
			h.mu.Unlock()

			// Добавляем в Redis кэш
			ctx := context.Background()
			h.roomCache.AddUser(ctx, client.tableID, client.userID, client.username)

			// Создаем событие о подключении
			joinEvent := &domain.WSEvent{
				Type:      domain.WSEventTypeJoin,
				TableID:   client.tableID,
				UserID:    client.userID,
				Username:  client.username,
				Content:   map[string]string{"message": "присоединился к столу"},
				Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			}

			// Сохраняем в БД и отправляем всем
			h.handleEvent(joinEvent)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.tables[client.tableID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)

					if len(clients) == 0 {
						delete(h.tables, client.tableID)
					}
				}
			}
			h.mu.Unlock()

			// Удаляем из Redis кэша
			ctx := context.Background()
			h.roomCache.RemoveUser(ctx, client.tableID, client.userID)

			// Создаем событие об отключении
			leaveEvent := &domain.WSEvent{
				Type:      domain.WSEventTypeLeave,
				TableID:   client.tableID,
				UserID:    client.userID,
				Username:  client.username,
				Content:   map[string]string{"message": "покинул стол"},
				Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			}

			h.handleEvent(leaveEvent)

		case event := <-h.broadcast:
			h.mu.RLock()
			clients := h.tables[event.TableID]
			h.mu.RUnlock()

			// Отправляем всем в комнате
			if clients != nil {
				data, _ := json.Marshal(event)
				for client := range clients {
					select {
					case client.send <- data:
					default:
						close(client.send)
						delete(clients, client)
					}
				}
			}
		}
	}
}

// Обработка события
func (h *Hub) handleEvent(event *domain.WSEvent) {
	ctx := context.Background()

	// Сохраняем событие в БД и кэш
	go h.eventService.ProcessEvent(ctx, event)

	// Отправляем всем в комнате
	h.broadcast <- event
}

// Отправка события конкретному клиенту
func (h *Hub) SendToClient(client *WSClient, event *domain.WSEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	select {
	case client.send <- data:
	default:
		return errors.New("client send channel full")
	}

	return nil
}

// Получить количество клиентов в комнате
func (h *Hub) GetClientCount(tableID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.tables[tableID])
}
