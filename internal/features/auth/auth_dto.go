package auth

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required" example:"198001012010011001"` // NIP or email
	Password   string `json:"password" binding:"required" example:"s3cretPass!"`
}

// RegisterRequest creates a staff account. Role is ignored for the very first account,
// which always becomes an IT administrator.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,max=150" example:"Dr. Jane Doe"`
	NIP      string `json:"nip" binding:"required,max=30" example:"198001012010011001"`
	Email    string `json:"email" binding:"required,email,max=150" example:"jane.doe@hospital.id"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"s3cretPass!"`
	Role     string `json:"role" example:"doctor"`
}

type UserResponse struct {
	ID        string    `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	Name      string    `json:"name" example:"Dr. Jane Doe"`
	NIP       string    `json:"nip" example:"198001012010011001"`
	Email     string    `json:"email" example:"jane.doe@hospital.id"`
	Role      string    `json:"role" example:"doctor"`
	IsActive  bool      `json:"is_active" example:"true"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type" example:"Bearer"`
	ExpiresIn   int64        `json:"expires_in" example:"28800"` // seconds
	User        UserResponse `json:"user"`
}

func toUserResponse(u *models.User) UserResponse {
	return UserResponse{
		ID:        u.ID.String(),
		Name:      u.Name,
		NIP:       u.NIP,
		Email:     u.Email,
		Role:      u.Role.Name,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
}
