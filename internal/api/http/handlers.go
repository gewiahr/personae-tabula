package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"personae-tabula/internal/domain"
	"personae-tabula/internal/service"

	"github.com/google/uuid"
)

type APIHandler struct {
	userService  *service.UserService
	tableService *service.TableService
	eventService *service.EventService
}

func NewAPIHandler(
	userService *service.UserService,
	tableService *service.TableService,
	eventService *service.EventService,
) *APIHandler {
	return &APIHandler{
		userService:  userService,
		tableService: tableService,
		eventService: eventService,
	}
}

// Request/Response структуры
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type CreateUserResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type CreateTableRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedBy   int64  `json:"created_by"`
}

type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// ==================== User Handlers ====================

// POST /api/users
func (h *APIHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validateCreateUser(req); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.userService.CreateUser(r.Context(), req.Username, req.Email)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := CreateUserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}

	h.sendJSON(w, response, http.StatusCreated)
}

// GET /api/users/{id}
func (h *APIHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	if userIDStr == "" {
		h.sendError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.sendError(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	user, err := h.userService.GetUser(r.Context(), userID)
	if err != nil {
		h.sendError(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		h.sendError(w, "User not found", http.StatusNotFound)
		return
	}

	h.sendJSON(w, user, http.StatusOK)
}

// ==================== Table Handlers ====================

// POST /api/tables
func (h *APIHandler) CreateTable(w http.ResponseWriter, r *http.Request) {
	var req CreateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validateCreateTable(req); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	table, err := h.tableService.CreateTable(r.Context(), req.Name, req.Description, req.CreatedBy)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.sendJSON(w, table, http.StatusCreated)
}

// GET /api/tables
func (h *APIHandler) ListTables(w http.ResponseWriter, r *http.Request) {
	limit := h.getQueryInt(r, "limit", 20)
	offset := h.getQueryInt(r, "offset", 0)

	// Ограничиваем максимальный лимит
	if limit > 100 {
		limit = 100
	}

	tables, err := h.tableService.ListTables(r.Context(), limit, offset)
	if err != nil {
		h.sendError(w, "Failed to list tables", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"tables": tables,
		"pagination": map[string]int{
			"limit":  limit,
			"offset": offset,
			"total":  len(tables),
		},
	}

	h.sendJSON(w, response, http.StatusOK)
}

// GET /api/tables/{id}
func (h *APIHandler) GetTable(w http.ResponseWriter, r *http.Request) {
	tableIDstr := r.PathValue("id")
	if tableIDstr == "" {
		h.sendError(w, "Table ID is required", http.StatusBadRequest)
		return
	}

	tableID, err := strconv.ParseInt(tableIDstr, 10, 64)
	if err != nil {
		h.sendError(w, "Invalid table ID format", http.StatusBadRequest)
		return
	}

	table, err := h.tableService.GetTable(r.Context(), tableID)
	if err != nil {
		h.sendError(w, "Failed to get table", http.StatusInternalServerError)
		return
	}
	if table == nil {
		h.sendError(w, "Table not found", http.StatusNotFound)
		return
	}

	h.sendJSON(w, table, http.StatusOK)
}

// GET /api/tables/{id}/feed
func (h *APIHandler) GetTableFeed(w http.ResponseWriter, r *http.Request) {
	tableIDStr := r.PathValue("id")
	if tableIDStr == "" {
		h.sendError(w, "Table ID is required", http.StatusBadRequest)
		return
	}

	tableID, err := strconv.ParseInt(tableIDStr, 10, 64)
	if err != nil {
		h.sendError(w, "Invalid table ID format", http.StatusBadRequest)
		return
	}

	// Параметры запроса
	limit := h.getQueryInt(r, "limit", 50)
	offset := h.getQueryInt(r, "offset", 0)
	format := r.URL.Query().Get("format")

	// Ограничиваем лимит
	if limit > 200 {
		limit = 200
	}

	var result any

	switch format {
	case "text":
		result, err = h.tableService.GetTableFeed(r.Context(), tableID, limit)
	case "json", "":
		result, err = h.eventService.GetHistory(r.Context(), tableID, limit, offset)
	default:
		h.sendError(w, "Invalid format parameter. Use 'text' or 'json'", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.sendError(w, "Failed to get feed", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"table_id": tableID,
		"format":   format,
		"feed":     result,
		"pagination": map[string]int{
			"limit":  limit,
			"offset": offset,
		},
	}

	h.sendJSON(w, response, http.StatusOK)
}

// POST /api/tables/{id}/events (для тестирования)
func (h *APIHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	tableIDStr := r.PathValue("id")
	if tableIDStr == "" {
		h.sendError(w, "Table ID is required", http.StatusBadRequest)
		return
	}

	tableID, err := strconv.ParseInt(tableIDStr, 10, 64)
	if err != nil {
		h.sendError(w, "Invalid table ID format", http.StatusBadRequest)
		return
	}

	var eventData struct {
		Type    string `json:"type"`
		UserID  int64  `json:"user_id"`
		Content any    `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&eventData); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if eventData.Type == "" {
		h.sendError(w, "Event type is required", http.StatusBadRequest)
		return
	}
	if eventData.UserID == 0 {
		h.sendError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Получаем пользователя
	user, err := h.userService.GetUser(r.Context(), eventData.UserID)
	if err != nil {
		h.sendError(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		h.sendError(w, "User not found", http.StatusNotFound)
		return
	}

	// Создаем событие
	event := &domain.WSEvent{
		Type:      domain.WSEventType(eventData.Type),
		TableID:   tableID,
		UserID:    eventData.UserID,
		Username:  user.Username,
		Content:   eventData.Content,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}

	if err := h.eventService.ProcessEvent(r.Context(), event); err != nil {
		h.sendError(w, "Failed to save event", http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, event, http.StatusCreated)
}

// // DELETE /api/tables/{id} (опционально)
// func (h *APIHandler) DeleteTable(w http.ResponseWriter, r *http.Request) {
// 	tableID := r.PathValue("id")
// 	if tableID == "" {
// 		h.sendError(w, "Table ID is required", http.StatusBadRequest)
// 		return
// 	}

// 	if !h.isValidUUID(tableID) {
// 		h.sendError(w, "Invalid table ID format", http.StatusBadRequest)
// 		return
// 	}

// 	// Здесь можно добавить проверку прав (например, только создатель может удалять)

// 	err := h.tableService.DeleteTable(r.Context(), tableID)
// 	if err != nil {
// 		h.sendError(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	h.sendJSON(w, map[string]string{"status": "deleted"}, http.StatusOK)
// }

// ==================== Health ====================

// GET /api/health
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.sendJSON(w, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	}, http.StatusOK)
}

// ==================== Валидация ====================

func (h *APIHandler) validateCreateUser(req CreateUserRequest) error {
	if req.Username == "" {
		return errors.New("username is required")
	}
	if len(req.Username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if len(req.Username) > 50 {
		return errors.New("username must be less than 50 characters")
	}

	if req.Email == "" {
		return errors.New("email is required")
	}
	if !h.isValidEmail(req.Email) {
		return errors.New("invalid email format")
	}

	return nil
}

func (h *APIHandler) validateCreateTable(req CreateTableRequest) error {
	if req.Name == "" {
		return errors.New("table name is required")
	}
	if len(req.Name) < 3 {
		return errors.New("table name must be at least 3 characters")
	}
	if len(req.Name) > 100 {
		return errors.New("table name must be less than 100 characters")
	}

	if req.CreatedBy == 0 {
		return errors.New("created_by (user ID) is required")
	}

	return nil
}

func (h *APIHandler) isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func (h *APIHandler) isValidEmail(email string) bool {
	// Простая проверка email
	// В продакшене лучше использовать нормальную библиотеку валидации
	return len(email) > 3 && len(email) < 100 && email != "" && email != "@"
}

// ==================== Утилиты ====================

func (h *APIHandler) sendJSON(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode response: %v", err)
		// Не можем отправить ошибку, так как статус уже отправлен
	}
}

func (h *APIHandler) sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := ErrorResponse{
		Error: message,
		Code:  status,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode error response: %v", err)
		// Пробуем отправить plain text
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(message))
	}
}

func (h *APIHandler) getQueryInt(r *http.Request, key string, defaultValue int) int {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	if value < 0 {
		return defaultValue
	}

	return value
}
