// Package models contains database models (entities) for the application.
//
// Models represent database tables and are used with Bun ORM for queries.
// Each model struct maps to a table, with fields mapping to columns.
//
// Bun struct tags:
//   - bun:"table:name"     - Specifies the table name
//   - bun:"alias:x"        - Short alias for queries (e.g., "u" for users)
//   - bun:"pk"             - Primary key
//   - bun:"autoincrement"  - Auto-incrementing column
//   - bun:"notnull"        - NOT NULL constraint
//   - bun:"default:value"  - Default value (e.g., "default:now()")
//   - bun:"-"              - Ignore field (not mapped to database)
//
// Example model:
//
//	type User struct {
//	    bun.BaseModel `bun:"table:users,alias:u"`
//
//	    ID        int64     `bun:"id,pk,autoincrement"`
//	    Email     string    `bun:"email,notnull,unique"`
//	    Name      string    `bun:"name,notnull"`
//	    CreatedAt time.Time `bun:"created_at,notnull,default:now()"`
//	    UpdatedAt time.Time `bun:"updated_at,notnull,default:now()"`
//	}
//
// After creating a model:
// 1. Create a migration: make migrate-create name=create_users
// 2. Edit the migration files in migrations/
// 3. Run migrations: make migrate-up
// 4. Create a service in internal/services/
package models
