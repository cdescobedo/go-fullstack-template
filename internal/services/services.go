// Package services contains business logic for the application.
//
// Services are the middle layer between handlers and the database.
// They encapsulate business rules and data validation, keeping handlers thin
// and focused on HTTP concerns.
//
// Architecture:
//
//	Handler → Service → Database
//	         ↑
//	   Business Logic
//	   Validation
//	   Error Handling
//
// Services should:
//   - Validate input data
//   - Apply business rules
//   - Return domain-specific errors
//   - Be testable in isolation (mock the database)
//
// Example service:
//
//	var ErrUserNotFound = errors.New("user not found")
//	var ErrInvalidEmail = errors.New("invalid email address")
//
//	type UserService struct {
//	    db *bun.DB
//	}
//
//	func NewUserService(db *bun.DB) *UserService {
//	    return &UserService{db: db}
//	}
//
//	func (s *UserService) Create(ctx context.Context, email, name string) (*models.User, error) {
//	    if !isValidEmail(email) {
//	        return nil, ErrInvalidEmail
//	    }
//
//	    user := &models.User{Email: email, Name: name}
//	    _, err := s.db.NewInsert().Model(user).Exec(ctx)
//	    return user, err
//	}
//
//	func (s *UserService) GetByID(ctx context.Context, id int64) (*models.User, error) {
//	    user := new(models.User)
//	    err := s.db.NewSelect().Model(user).Where("id = ?", id).Scan(ctx)
//	    if err != nil {
//	        return nil, ErrUserNotFound
//	    }
//	    return user, nil
//	}
//
// After creating a service:
// 1. Add it to handlers.Handlers struct in internal/handlers/handlers.go
// 2. Initialize it in handlers.New()
// 3. Use it in your handlers
package services
