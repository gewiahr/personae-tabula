package websocket

import (
	"log"
	"net/http"
	"strconv"

	"personae-tabula/internal/service"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // В продакшене проверять
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebSocketHandler struct {
	hub          *Hub
	userService  *service.UserService
	tableService *service.TableService
}

func NewWebSocketHandler(
	hub *Hub,
	userService *service.UserService,
	tableService *service.TableService,
) *WebSocketHandler {
	return &WebSocketHandler{
		hub:          hub,
		userService:  userService,
		tableService: tableService,
	}
}

func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры
	tableIDStr := r.URL.Query().Get("table_id")
	userIDStr := r.URL.Query().Get("user_id")

	if tableIDStr == "" || userIDStr == "" {
		http.Error(w, "table_id and user_id are required", http.StatusBadRequest)
		return
	}

	tableID, err := strconv.ParseInt(tableIDStr, 10, 64)
	if err != nil {
		http.Error(w, "table_id is invalid", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "user_id is invalid", http.StatusBadRequest)
		return
	}

	// Проверяем существование стола
	ctx := r.Context()
	table, err := h.tableService.GetTable(ctx, tableID)
	if err != nil || table == nil {
		http.Error(w, "table not found", http.StatusNotFound)
		return
	}

	// Проверяем пользователя
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "cannot establish websocket connection", http.StatusInternalServerError)
		log.Println("WebSocket upgrade failed:", err)
		return
	}

	// Создаем клиента
	client := &WSClient{
		hub:     h.hub,
		conn:    conn,
		send:    make(chan []byte, 256),
		tableID: tableID,
		userID:  userID,
	}

	// Регистрируем в хабе
	client.hub.register <- client

	// Запускаем горутины
	go client.writePump()
	go client.readPump()
}
