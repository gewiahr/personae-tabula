package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"personae-tabula/internal/domain"

	"github.com/go-redis/redis/v8"
)

type UserCache struct {
	client *redis.Client
}

func NewUserCache(client *redis.Client) *UserCache {
	return &UserCache{client: client}
}

// Ключи для Redis
func userStateKey(userID int64) string {
	return fmt.Sprintf("user:state:%d", userID)
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
func (c *UserCache) AddUser(ctx context.Context, user *domain.UserState) error {
	pipe := c.client.TxPipeline()
	pipe.SAdd(ctx, userStateKey(user.UserID), user)
	_, err := pipe.Exec(ctx)
	return err
}

// Удалить пользователя из комнаты
func (c *UserCache) GetUser(ctx context.Context, userID int64) (*domain.UserState, error) {
	data, err := c.client.Get(ctx, userStateKey(userID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var state domain.UserState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (c *UserCache) RemoveUser(ctx context.Context, userID int64) error {
	pipe := c.client.TxPipeline()
	pipe.Del(ctx, userStateKey(userID))
	_, err := pipe.Exec(ctx)
	return err
}

// Получить список пользователей в комнате
// func (c *RoomCache) GetUsers(ctx context.Context, tableID int64) ([]string, error) {
// 	return c.client.SMembers(ctx, roomUsersKey(tableID)).Result()
// }

// // Добавить событие в ленту комнаты (Redis список, для быстрого доступа)
// func (c *RoomCache) PushEvent(ctx context.Context, tableID int64, event *domain.WSEvent) error {
// 	// Форматируем событие в текст
// 	text := event.FormatText()

// 	// Добавляем в начало списка (LPush для reversed order)
// 	// Используем транзакцию для атомарности
// 	pipe := c.client.TxPipeline()

// 	// Добавляем событие в список
// 	pipe.LPush(ctx, roomFeedKey(tableID), text)

// 	// Обрезаем список до последних 100 событий
// 	pipe.LTrim(ctx, roomFeedKey(tableID), 0, 99)

// 	// Обновляем состояние комнаты
// 	stateKey := roomStateKey(tableID)
// 	pipe.HIncrBy(ctx, stateKey, "event_count", 1)
// 	pipe.HSet(ctx, stateKey, "last_event", text, "updated_at", time.Now().Unix())

// 	_, err := pipe.Exec(ctx)
// 	return err
// }
