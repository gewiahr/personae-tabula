package service

import (
	"context"
	"time"

	"personae-tabula/internal/domain"
	"personae-tabula/internal/repository/postgres"
	"personae-tabula/internal/repository/redis"
)

type TableService struct {
	tableRepo *postgres.TableRepository
	eventRepo *postgres.EventRepository
	roomCache *redis.RoomCache
}

func NewTableService(
	tableRepo *postgres.TableRepository,
	eventRepo *postgres.EventRepository,
	roomCache *redis.RoomCache,
) *TableService {
	return &TableService{
		tableRepo: tableRepo,
		eventRepo: eventRepo,
		roomCache: roomCache,
	}
}

func (s *TableService) CreateTable(ctx context.Context, name, description string, createdBy int64) (*domain.Table, error) {
	table := &domain.Table{
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
		IsActive:    true,
	}

	if err := s.tableRepo.Create(ctx, table); err != nil {
		return nil, err
	}

	// Создаем системное событие о создании стола
	event := &domain.WSEvent{
		Type:      domain.WSEventTypeSystem,
		TableID:   table.ID,
		Content:   "Стол создан",
		Timestamp: table.CreatedAt.UnixNano() / int64(time.Millisecond),
	}

	s.saveEvent(ctx, event)

	return table, nil
}

func (s *TableService) GetTable(ctx context.Context, id int64) (*domain.Table, error) {
	return s.tableRepo.GetByID(ctx, id)
}

func (s *TableService) GetTableFeed(ctx context.Context, tableID int64, limit int) ([]string, error) {
	// Сначала пробуем получить из кэша
	events, err := s.roomCache.GetRecentEvents(ctx, tableID, limit)
	if err == nil && len(events) > 0 {
		return events, nil
	}

	// Если в кэше нет, грузим из БД
	return s.eventRepo.GetTableFeedText(ctx, tableID, limit)
}

// Внутренний метод для сохранения события
func (s *TableService) saveEvent(ctx context.Context, event *domain.WSEvent) error {
	dbEvent, err := event.ToDBEvent()
	if err != nil {
		return err
	}

	// Сохраняем в PostgreSQL
	if err := s.eventRepo.Save(ctx, dbEvent); err != nil {
		return err
	}

	// Сохраняем в Redis кэш
	return s.roomCache.PushEvent(ctx, event.TableID, event)
}

func (s *TableService) ListTables(ctx context.Context, limit, offset int) ([]domain.Table, error) {
	tables, err := s.tableRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return tables, nil
}
