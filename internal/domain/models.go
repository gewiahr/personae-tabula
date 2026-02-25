package domain

import (
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID        int64     `bun:"id,pk,autoincrement" json:"id"`
	Username  string    `bun:"username,notnull,unique" json:"username"`
	Email     string    `bun:"email,notnull,unique" json:"email"`
	CreatedAt time.Time `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:now()" json:"updated_at"`
}

type Table struct {
	bun.BaseModel `bun:"table:tables,alias:t"`

	ID          int64     `bun:"id,pk,autoincrement" json:"id"`
	Name        string    `bun:"name,notnull" json:"name"`
	Description string    `bun:"description" json:"description"`
	CreatedBy   int64     `bun:"created_by,notnull" json:"created_by"`
	IsActive    bool      `bun:"is_active,notnull,default:true" json:"is_active"`
	Feed        string    `bun:"feed" json:"feed"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:now()" json:"updated_at"`

	// Relations
	Creator *User `bun:"rel:belongs-to,join:created_by=id" json:"creator,omitempty"`
}

// Для хранения событий комнаты (текстовый фид)
type TableEvent struct {
	bun.BaseModel `bun:"table:table_events,alias:te"`

	ID        int64     `bun:"id,pk,autoincrement" json:"id"`
	TableID   int64     `bun:"table_id,notnull" json:"table_id"`
	EventType string    `bun:"event_type,notnull" json:"event_type"`     // join, leave, roll, chat, system
	UserID    int64     `bun:"user_id" json:"user_id"`                   // может быть null для системных событий
	Content   string    `bun:"content,notnull,type:text" json:"content"` // текстовое содержимое события
	Metadata  string    `bun:"metadata,type:jsonb" json:"metadata"`      // дополнительные данные в JSON
	CreatedAt time.Time `bun:"created_at,notnull,default:now()" json:"created_at"`

	// Relations
	Table *Table `bun:"rel:belongs-to,join:table_id=id" json:"table,omitempty"`
	User  *User  `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

// Для Redis кэша - текущее состояние комнаты
type RoomState struct {
	TableID    int64    `json:"table_id"`
	TableName  string   `json:"table_name"`
	Users      []string `json:"users"`      // список ID пользователей
	UserNames  []string `json:"user_names"` // имена для отображения
	EventCount int      `json:"event_count"`
	LastEvent  string   `json:"last_event"`
	UpdatedAt  int64    `json:"updated_at"` // timestamp
}
