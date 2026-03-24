package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"personae-tabula/internal/domain"
	"personae-tabula/internal/repository/redis"
	"personae-tabula/internal/service"
)

type Hub struct {
	tables          map[int64]map[*WSClient]bool // tableID -> clients
	register        chan *WSClient
	unregister      chan *WSClient
	broadcast       chan *domain.WSEvent[any]
	broadcastSystem chan *domain.WSEvent[*domain.SystemWSEvent[any]]
	broadcastTable  chan *domain.WSEvent[*domain.TableWSEvent[any]]
	mu              sync.RWMutex

	// Сервисы
	eventService *service.EventService
	tableService *service.TableService
	userCache    *redis.UserCache
	roomCache    *redis.RoomCache
}

func NewHub(
	eventService *service.EventService,
	tableService *service.TableService,
	roomCache *redis.RoomCache,
	userCache *redis.UserCache,
) *Hub {
	return &Hub{
		tables:          make(map[int64]map[*WSClient]bool),
		register:        make(chan *WSClient),
		unregister:      make(chan *WSClient),
		broadcast:       make(chan *domain.WSEvent[any], 512),
		broadcastSystem: make(chan *domain.WSEvent[*domain.SystemWSEvent[any]]),
		broadcastTable:  make(chan *domain.WSEvent[*domain.TableWSEvent[any]]),
		eventService:    eventService,
		tableService:    tableService,
		userCache:       userCache,
		roomCache:       roomCache,
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
			h.roomCache.AddUser(ctx, client.tableID, client.userID)
			//h.userCache.AddUser(ctx)

			// Создаем событие о подключении
			joinEvent := &domain.WSEvent[*domain.SystemWSEvent[any]]{
				Type:    domain.WSEventTypeSystem,
				TableID: client.tableID,
				UserID:  client.userID,
				Event: &domain.SystemWSEvent[any]{
					Type:    domain.SystemWSEventTypeLeave,
					Content: nil,
				},
				Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			}

			// Сохраняем в БД и отправляем всем
			h.handleSystemEvent(joinEvent)

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
			leaveEvent := &domain.WSEvent[*domain.SystemWSEvent[any]]{
				Type:    domain.WSEventTypeSystem,
				TableID: client.tableID,
				UserID:  client.userID,
				Event: &domain.SystemWSEvent[any]{
					Type:    domain.SystemWSEventTypeJoin,
					Content: nil,
				},
				Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			}

			h.handleSystemEvent(leaveEvent)

		case event := <-h.broadcast:
			h.mu.RLock()
			clients := h.tables[event.TableID]
			h.mu.RUnlock()

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

			// case event := <-h.broadcastSystem: //<-h.broadcast:
			// 	h.mu.RLock()
			// 	clients := h.tables[event.TableID]
			// 	h.mu.RUnlock()

			// 	if clients != nil {
			// 		data, _ := json.Marshal(event)
			// 		for client := range clients {
			// 			select {
			// 			case client.send <- data:
			// 			default:
			// 				close(client.send)
			// 				delete(clients, client)
			// 			}
			// 		}
			// 	}

			// case event := <-h.broadcastTable: //<-h.broadcast:
			// 	h.mu.RLock()
			// 	clients := h.tables[event.TableID]
			// 	h.mu.RUnlock()

			// 	if clients != nil {
			// 		data, _ := json.Marshal(event)
			// 		for client := range clients {
			// 			select {
			// 			case client.send <- data:
			// 			default:
			// 				close(client.send)
			// 				delete(clients, client)
			// 			}
			// 		}
			// 	}
		}
	}
}

func (h *Hub) handleSystemEvent(event *domain.WSEvent[*domain.SystemWSEvent[any]]) {
	ctx := context.Background()

	go h.eventService.ProcessSystemEvent(ctx, event.Event)
	//h.broadcastSystem <- event

	eventToBroadcast := &domain.WSEvent[any]{
		Type:      event.Type,
		Event:     event.Event,
		UserID:    event.UserID,
		TableID:   event.TableID,
		Timestamp: event.Timestamp,
	}
	h.broadcast <- eventToBroadcast

	//rollResultEvent, _ := h.eventService.ProcessEvent(ctx, event)
	// if rollResultEvent != nil {
	// 	h.broadcast <- rollResultEvent
	// }
}

func (h *Hub) handleTableEvent(ctx context.Context, event *domain.WSEvent[*domain.TableWSEvent[any]]) {
	tableEvent, err := h.eventService.ProcessTableEvent(ctx, event.UserID, event.TableID, event.Event, event.Timestamp)
	if err != nil {
		log.Printf("error processing event: %v", err)
		return
	}

	resultEvent := &domain.WSEvent[any]{
		Type:      domain.WSEventTypeTable,
		Event:     tableEvent,
		UserID:    event.UserID,
		TableID:   event.TableID,
		Timestamp: event.Timestamp,
	}

	if err := h.roomCache.PushEvent(ctx, event.TableID, resultEvent); err != nil {
		log.Printf("error push to redis: %d", err)
	}

	h.broadcast <- resultEvent
}

// func (h *Hub) SendToClient(client *WSClient, event *domain.WSEvent[any]) error {
// 	data, err := json.Marshal(event)
// 	if err != nil {
// 		return err
// 	}

// 	select {
// 	case client.send <- data:
// 	default:
// 		return errors.New("client send channel full")
// 	}

// 	return nil
// }

// Получить количество клиентов в комнате
// func (h *Hub) GetClientCount(tableID int64) int {
// 	h.mu.RLock()
// 	defer h.mu.RUnlock()
// 	return len(h.tables[tableID])
// }
