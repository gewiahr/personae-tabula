package apiHttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"personae-tabula/internal/service"

	"github.com/google/uuid"
)

type APIHandlerFunc func(http.ResponseWriter, *http.Request)
type APIEndpointFunc func(context.Context, *http.Request) *APIResponse

type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message,omitzero"`
}

type APIResponse struct {
	Payload any
	Error   *APIError
	Code    int
}

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

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email,omitzero"`
	Password string `json:"password"`
}

type CreateUserResponse struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type CreateTableRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedBy   string `json:"createdBy"`
}

// ==================== User Handlers ====================

// POST /user/register
func (h *APIHandler) RegisterUser(ctx context.Context, r *http.Request) *APIResponse {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return h.newErrorResponse(err.Error(), "Invalid request body", http.StatusBadRequest)
	}

	if err := h.validateCreateUser(req); err != nil {
		return h.newErrorResponse(err.Error(), err.Error(), http.StatusBadRequest)
	}

	user, err := h.userService.CreateUser(ctx, req.Username, req.Password, req.Email)
	if err != nil {
		return h.newErrorResponse(err.Error(), "", http.StatusBadRequest)
	}

	response := CreateUserResponse{
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}

	return h.newApiResponse(response, http.StatusCreated)
}

// GET /user/{id}
func (h *APIHandler) GetUser(ctx context.Context, r *http.Request) *APIResponse {
	userIDStr := r.PathValue("id")
	if userIDStr == "" {
		return h.newErrorResponse("User ID is required", "", http.StatusBadRequest)
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return h.newErrorResponse("Invalid user ID format", "", http.StatusBadRequest)
	}

	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		return h.newErrorResponse("Failed to get user", "", http.StatusInternalServerError)
	}
	if user == nil {
		return h.newErrorResponse("User not found", "", http.StatusNotFound)
	}

	return h.newApiResponse(user, http.StatusOK)
}

// ==================== Table Handlers ====================

// POST /table
func (h *APIHandler) CreateTable(ctx context.Context, r *http.Request) *APIResponse {
	var req CreateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return h.newErrorResponse("Invalid request body", "", http.StatusBadRequest)
	}

	if err := h.validateCreateTable(req); err != nil {
		return h.newErrorResponse(err.Error(), "", http.StatusBadRequest)
	}

	user, err := h.userService.GetUserByUsername(ctx, req.CreatedBy)
	if err != nil {
		return h.newErrorResponse(err.Error(), "Error getting user", http.StatusInternalServerError)
	}
	if user == nil {
		return h.newErrorResponse("", "User not found", http.StatusNotFound)
	}

	table, err := h.tableService.CreateTable(ctx, user.ID, req.Name, req.Description)
	if err != nil {
		return h.newErrorResponse(err.Error(), "", http.StatusBadRequest)
	}

	return h.newApiResponse(table, http.StatusCreated)
}

// GET /tables
func (h *APIHandler) ListTables(ctx context.Context, r *http.Request) *APIResponse {
	limit := h.getQueryInt(r, "limit", 20)
	offset := h.getQueryInt(r, "offset", 0)

	// Ограничиваем максимальный лимит
	if limit > 100 {
		limit = 100
	}

	tables, err := h.tableService.ListTables(r.Context(), limit, offset)
	if err != nil {
		return h.newErrorResponse(err.Error(), "Failed to list tables", http.StatusInternalServerError)
	}

	response := map[string]any{
		"tables": tables,
		"pagination": map[string]int{
			"limit":  limit,
			"offset": offset,
			"total":  len(tables),
		},
	}

	return h.newApiResponse(response, http.StatusOK)
}

// GET /table/{id}
func (h *APIHandler) GetTable(ctx context.Context, r *http.Request) *APIResponse {
	tableIDstr := r.PathValue("id")
	if tableIDstr == "" {
		return h.newErrorResponse("Table ID is required", "", http.StatusBadRequest)
	}

	tableID, err := strconv.ParseInt(tableIDstr, 10, 64)
	if err != nil {
		return h.newErrorResponse(err.Error(), "Invalid table ID format", http.StatusBadRequest)
	}

	table, err := h.tableService.GetTable(r.Context(), tableID)
	if err != nil {
		return h.newErrorResponse(err.Error(), "Failed to get table", http.StatusInternalServerError)
	}
	if table == nil {
		return h.newErrorResponse("Table not found", "", http.StatusNotFound)
	}

	return h.newApiResponse(table, http.StatusOK)
}

// GET /table/{id}/feed
func (h *APIHandler) GetTableFeed(ctx context.Context, r *http.Request) *APIResponse {
	tableIDStr := r.PathValue("id")
	if tableIDStr == "" {
		return h.newErrorResponse("Table ID is required", "", http.StatusBadRequest)
	}

	tableID, err := strconv.ParseInt(tableIDStr, 10, 64)
	if err != nil {
		return h.newErrorResponse("Invalid table ID format", "", http.StatusBadRequest)
	}

	// // Параметры запросаevent
	// limit := h.getQueryInt(r, "limit", 50)
	// offset := h.getQueryInt(r, "offset", 0)
	// format := r.URL.Query().Get("format")

	// // Ограничиваем лимит
	// if limit > 200 {
	// 	limit = 200
	// }

	var result any

	result, err = h.tableService.GetTableFeed(r.Context(), tableID, 50)

	// switch format {
	// case "text":
	// 	result, err = h.tableService.GetTableFeed(r.Context(), tableID, limit)
	// case "json", "":
	// 	result, err = h.eventService.GetHistory(r.Context(), tableID, limit, offset)
	// default:
	// 	return h.newErrorResponse("Invalid format parameter. Use 'text' or 'json'", "", http.StatusBadRequest)
	// }

	// if err != nil {
	// 	return h.newErrorResponse(err.Error(), "Failed to get feed", http.StatusInternalServerError)
	// }

	// response := map[string]any{
	// 	"table_id": tableID,
	// 	"format":   format,
	// 	"feed":     result,
	// 	"pagination": map[string]int{
	// 		"limit":  limit,
	// 		"offset": offset,
	// 	},
	// }

	return h.newApiResponse(result, http.StatusOK) //(response, http.StatusOK)
}

// POST /table/{id}/events (для тестирования)
// func (h *APIHandler) CreateEvent(ctx context.Context, r *http.Request) *APIResponse {
// 	tableIDStr := r.PathValue("id")
// 	if tableIDStr == "" {
// 		return h.newErrorResponse("Table ID is required", "", http.StatusBadRequest)
// 	}

// 	tableID, err := strconv.ParseInt(tableIDStr, 10, 64)
// 	if err != nil {
// 		return h.newErrorResponse(err.Error(), "Invalid table ID format", http.StatusBadRequest)
// 	}

// 	var eventData struct {
// 		Type    string `json:"type"`
// 		UserID  int64  `json:"user_id"`
// 		Content any    `json:"content"`
// 	}

// 	if err := json.NewDecoder(r.Body).Decode(&eventData); err != nil {
// 		return h.newErrorResponse(err.Error(), "Invalid request body", http.StatusBadRequest)
// 	}

// 	// Валидация
// 	if eventData.Type == "" {
// 		return h.newErrorResponse("Event type is required", "", http.StatusBadRequest)
// 	}
// 	if eventData.UserID == 0 {
// 		return h.newErrorResponse("User ID is required", "", http.StatusBadRequest)
// 	}

// 	// Получаем пользователя
// 	user, err := h.userService.GetUser(r.Context(), eventData.UserID)
// 	if err != nil {
// 		return h.newErrorResponse(err.Error(), "Failed to get user", http.StatusInternalServerError)
// 	}
// 	if user == nil {
// 		return h.newErrorResponse("User not found", "", http.StatusNotFound)
// 	}

// 	// Создаем событие
// 	event := &domain.WSEvent{
// 		Type:      domain.WSEventType(eventData.Type),
// 		TableID:   tableID,
// 		UserID:    eventData.UserID,
// 		Content:   eventData.Content,
// 		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
// 	}

// 	if _, err := h.eventService.ProcessEvent(r.Context(), event); err != nil {
// 		return h.newErrorResponse(err.Error(), "Failed to save event", http.StatusInternalServerError)
// 	}

// 	return h.newApiResponse(event, http.StatusCreated)
// }

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

// GET /health
func (h *APIHandler) HealthCheck(ctx context.Context, r *http.Request) *APIResponse {
	return h.newApiResponse(map[string]any{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	}, http.StatusOK)
}

// ==================== Валидация ====================

func (h *APIHandler) validateCreateUser(req CreateUserRequest) error {
	const usernameMinLength = 3
	const usernameMaxLength = 30
	const passwordMinLength = 3
	const passwordMaxLength = 30

	if req.Username == "" {
		return fmt.Errorf("username is required")
	}
	if len(req.Username) < usernameMinLength {
		return fmt.Errorf("username must be at least %d characters", usernameMinLength)
	}
	if len(req.Username) > usernameMaxLength {
		return fmt.Errorf("username must be less than %d characters", usernameMaxLength)
	}

	if req.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(req.Password) < passwordMinLength {
		return fmt.Errorf("password must be at least %d characters", passwordMinLength)
	}
	if len(req.Password) > passwordMaxLength {
		return fmt.Errorf("password must be less than %d characters", passwordMaxLength)
	}

	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !h.isValidEmail(req.Email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func (h *APIHandler) validateCreateTable(req CreateTableRequest) error {
	const tablenameMinLength = 3
	const tablenameMaxLength = 50

	if req.Name == "" {
		return fmt.Errorf("table name is required")
	}
	if len(req.Name) < tablenameMinLength {
		return fmt.Errorf("table name must be at least %d characters", tablenameMinLength)
	}
	if len(req.Name) > tablenameMaxLength {
		return fmt.Errorf("table name must be less than %d characters", tablenameMaxLength)
	}

	return nil
}

func (h *APIHandler) isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func (h *APIHandler) isValidEmail(email string) bool {
	return len(email) > 3 && len(email) < 100 && email != "" && email != "@"
}
