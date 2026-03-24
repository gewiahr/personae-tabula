package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"personae-tabula/internal/domain"

	"github.com/go-redis/redis/v8"
)

type RoomCache struct {
	client *redis.Client
}

func NewRoomCache(client *redis.Client) *RoomCache {
	return &RoomCache{client: client}
}

// Ключи для Redis
func roomStateKey(tableID int64) string {
	return fmt.Sprintf("room:state:%d", tableID)
}

func roomUsersKey(tableID int64) string {
	return fmt.Sprintf("room:users:%d", tableID)
}

func roomFeedKey(tableID int64) string {
	return fmt.Sprintf("room:feed:%d", tableID)
}

// Сохранить состояние комнаты
// func (c *RoomCache) SaveRoomState(ctx context.Context, state *domain.RoomState) error {
// 	data, err := json.Marshal(state)
// 	if err != nil {
// 		return err
// 	}
// 	return c.client.Set(ctx, roomStateKey(state.TableID), data, time.Hour).Err()
// }

// Получить состояние комнаты
// func (c *RoomCache) GetRoomState(ctx context.Context, tableID int64) (*domain.RoomState, error) {
// 	data, err := c.client.Get(ctx, roomStateKey(tableID)).Bytes()
// 	if err == redis.Nil {
// 		return nil, nil
// 	}
// 	if err != nil {
// 		return nil, err
// 	}

// 	var state domain.RoomState
// 	if err := json.Unmarshal(data, &state); err != nil {
// 		return nil, err
// 	}
// 	return &state, nil
// }

// Добавить пользователя в комнату
func (c *RoomCache) AddUser(ctx context.Context, tableID, userID int64) error {
	pipe := c.client.TxPipeline()
	pipe.SAdd(ctx, roomUsersKey(tableID), userID)
	pipe.HSet(ctx, fmt.Sprintf("user:%d", userID), "room", tableID)
	_, err := pipe.Exec(ctx)
	return err
}

// Удалить пользователя из комнаты
func (c *RoomCache) RemoveUser(ctx context.Context, tableID, userID int64) error {
	pipe := c.client.TxPipeline()
	pipe.SRem(ctx, roomUsersKey(tableID), userID)
	pipe.HDel(ctx, fmt.Sprintf("user:%d", userID), "room")
	_, err := pipe.Exec(ctx)
	return err
}

// Получить список пользователей в комнате
func (c *RoomCache) GetUsers(ctx context.Context, tableID int64) ([]string, error) {
	return c.client.SMembers(ctx, roomUsersKey(tableID)).Result()
}

// Добавить событие в ленту комнаты (Redis список, для быстрого доступа)
func (c *RoomCache) PushEvent(ctx context.Context, tableID int64, event *domain.WSEvent[any]) error {
	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	pipe := c.client.TxPipeline()

	// Store JSON string instead of the struct
	pipe.LPush(ctx, roomFeedKey(tableID), eventJSON)
	pipe.LTrim(ctx, roomFeedKey(tableID), 0, 99)

	stateKey := roomStateKey(tableID)
	pipe.HIncrBy(ctx, stateKey, "event_count", 1)
	pipe.HSet(ctx, stateKey,
		"last_event", eventJSON,
		"updated_at", time.Now().Unix(),
	)

	_, err = pipe.Exec(ctx)
	return err
}

// Получить последние события из ленты
func (c *RoomCache) GetRecentEvents(ctx context.Context, tableID int64, limit int) ([]string, error) {
	return c.client.LRange(ctx, roomFeedKey(tableID), 0, int64(limit-1)).Result()
}

// Очистить состояние комнаты при закрытии
func (c *RoomCache) ClearRoom(ctx context.Context, tableID int64) error {
	pipe := c.client.TxPipeline()
	pipe.Del(ctx, roomStateKey(tableID))
	pipe.Del(ctx, roomUsersKey(tableID))
	pipe.Del(ctx, roomFeedKey(tableID))
	_, err := pipe.Exec(ctx)
	return err
}
