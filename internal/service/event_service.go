package service

import (
	"context"
	"time"

	"personae-tabula/internal/domain"
	"personae-tabula/internal/repository/postgres"
	"personae-tabula/internal/repository/redis"
)

type EventService struct {
	eventRepo *postgres.EventRepository
	roomCache *redis.RoomCache
}

func NewEventService(
	eventRepo *postgres.EventRepository,
	roomCache *redis.RoomCache,
) *EventService {
	return &EventService{
		eventRepo: eventRepo,
		roomCache: roomCache,
	}
}

// Обработка и сохранение события
func (s *EventService) ProcessEvent(ctx context.Context, event *domain.WSEvent) error {
	// Сохраняем событие
	dbEvent, err := event.ToDBEvent()
	if err != nil {
		return err
	}

	// В PostgreSQL
	if err := s.eventRepo.Save(ctx, dbEvent); err != nil {
		return err
	}

	// В Redis кэш
	if err := s.roomCache.PushEvent(ctx, event.TableID, event); err != nil {
		// Логируем ошибку, но не прерываем выполнение
		// return err
	}

	return nil
}

// Получить историю событий
func (s *EventService) GetHistory(ctx context.Context, tableID int64, limit, offset int) ([]domain.WSEvent, error) {
	dbEvents, err := s.eventRepo.GetTableFeed(ctx, tableID, limit, offset)
	if err != nil {
		return nil, err
	}

	var events []domain.WSEvent
	for _, dbEvent := range dbEvents {
		event, err := domain.EventFromDBEvent(&dbEvent)
		if err != nil {
			continue
		}
		events = append(events, *event)
	}

	return events, nil
}

// Обработка броска кубов
func (s *EventService) ProcessDiceRoll(ctx context.Context, tableID, userID int64, username, diceString string) (*domain.WSEvent, error) {
	// Здесь логика броска кубов
	result := calculateDiceRoll(diceString)

	event := &domain.WSEvent{
		Type:      domain.WSEventTypeRoll,
		TableID:   tableID,
		UserID:    userID,
		Username:  username,
		Content:   result,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}

	if err := s.ProcessEvent(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

func calculateDiceRoll(diceString string) map[string]any {
	// Простая реализация для примера
	// В реальности нужно парсить "2d6+3" и т.д.
	return map[string]any{
		"dice":     diceString,
		"result":   10,
		"details":  []int{4, 3},
		"modifier": 3,
	}
}
