package usecase

import (
	"context"
	"time"

	domainuser "contract-scanner/internal/domain/user"
	"contract-scanner/internal/infra/auth"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type IRegisterUser interface {
	Execute(ctx context.Context, input RegisterUserInput) (*RegisterUserOutput, error)
}

type RegisterUserInput struct {
	Username string
	Email    string
	Password string
}

type RegisterUserOutput struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Token    string    `json:"token"`
}

type RegisterUser struct {
	userRepo domainuser.Repository
	jwtSvc   auth.JWTService
}

func NewRegisterUser(userRepo domainuser.Repository, jwtSvc auth.JWTService) IRegisterUser {
	return &RegisterUser{
		userRepo: userRepo,
		jwtSvc:   jwtSvc,
	}
}

func (uc *RegisterUser) Execute(ctx context.Context, input RegisterUserInput) (*RegisterUserOutput, error) {
	emailExists, err := uc.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if emailExists {
		return nil, domainuser.ErrEmailAlreadyExists
	}

	usernameExists, err := uc.userRepo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		return nil, err
	}
	if usernameExists {
		return nil, domainuser.ErrUsernameAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	u := &domainuser.User{
		ID:           uuid.New(),
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}

	token, err := uc.jwtSvc.Generate(u.ID, u.Username, u.Email)
	if err != nil {
		return nil, err
	}

	return &RegisterUserOutput{
		UserID:   u.ID,
		Username: u.Username,
		Token:    token,
	}, nil
}
