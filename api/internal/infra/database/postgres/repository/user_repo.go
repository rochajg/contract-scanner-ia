package repository

import (
	"context"

	domainuser "contract-scanner/internal/domain/user"
	"contract-scanner/internal/infra/database/postgres/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepoPostgres struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepoPostgres {
	return &UserRepoPostgres{db: db}
}

func (r *UserRepoPostgres) Create(ctx context.Context, u *domainuser.User) error {
	m := models.User{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	u.ID = m.ID
	return nil
}

func (r *UserRepoPostgres) FindByEmail(ctx context.Context, email string) (*domainuser.User, error) {
	var m models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainuser.ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(m), nil
}

func (r *UserRepoPostgres) FindByID(ctx context.Context, id uuid.UUID) (*domainuser.User, error) {
	var m models.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainuser.ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(m), nil
}

func (r *UserRepoPostgres) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepoPostgres) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func toDomainUser(m models.User) *domainuser.User {
	return &domainuser.User{
		ID:           m.ID,
		Username:     m.Username,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
