package usecase

import (
	"context"

	domainuser "contract-scanner/internal/domain/user"
	"contract-scanner/internal/infra/auth"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ILoginUser interface {
	Execute(ctx context.Context, input LoginUserInput) (*LoginUserOutput, error)
}

type LoginUserInput struct {
	Email    string
	Password string
}

type LoginUserOutput struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Token    string    `json:"token"`
}

type LoginUser struct {
	userRepo domainuser.Repository
	jwtSvc   auth.JWTService
}

func NewLoginUser(userRepo domainuser.Repository, jwtSvc auth.JWTService) ILoginUser {
	return &LoginUser{
		userRepo: userRepo,
		jwtSvc:   jwtSvc,
	}
}

func (uc *LoginUser) Execute(ctx context.Context, input LoginUserInput) (*LoginUserOutput, error) {
	u, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if err == domainuser.ErrUserNotFound {
			return nil, domainuser.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domainuser.ErrInvalidCredentials
	}

	token, err := uc.jwtSvc.Generate(u.ID, u.Username, u.Email)
	if err != nil {
		return nil, err
	}

	return &LoginUserOutput{
		UserID:   u.ID,
		Username: u.Username,
		Token:    token,
	}, nil
}
