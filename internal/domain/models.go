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
	Password  string    `bun:"password,notnull" json:"-"`
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
	JoinCode    string    `bun:"join_code,notnull" json:"join_code"`

	// Relations
	Creator *User `bun:"rel:belongs-to,join:created_by=id" json:"creator,omitempty"`
}

type TableEvent struct {
	bun.BaseModel `bun:"table:table_events,alias:te"`

	ID        int64     `bun:"id,pk,autoincrement" json:"id"`
	EventType string    `bun:"event_type,notnull" json:"type"` // join, leave, roll, chat, system
	TableID   int64     `bun:"table_id,notnull" json:"tableId"`
	UserID    int64     `bun:"user_id" json:"userId"`                    // может быть null для системных событий
	Content   string    `bun:"content,notnull,type:text" json:"content"` // текстовое содержимое события
	CreatedAt time.Time `bun:"created_at,notnull,default:now()" json:"created"`
	//Metadata  string    `bun:"metadata,type:jsonb" json:"metadata"`      // дополнительные данные в JSON

	// Relations
	Table *Table `bun:"rel:belongs-to,join:table_id=id" json:"table,omitempty"`
	User  *User  `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

type RoomState struct {
	TableID    int64    `json:"tableId"`
	TableName  string   `json:"tableName"`
	Users      []string `json:"users"`
	EventCount int      `json:"eventCount"`
	LastEvent  string   `json:"lastEvent"`
	UpdatedAt  int64    `json:"updated"`
}

type UserState struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
}
