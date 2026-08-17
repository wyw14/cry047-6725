package domain

import (
	"context"
	"net/mail"
	"regexp"
	"time"
)

// ResponsiblePerson is the human accountable for one or more facilities.
type ResponsiblePerson struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	Department string    `json:"department"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Version    int       `json:"version"`
}

// ResponsiblePersonInput is the validated input for creating or updating a
// responsible person.
type ResponsiblePersonInput struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Department string `json:"department"`
}

var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$|^\d{3,4}-\d{6,8}$`)

// Validate enforces field-level invariants for ResponsiblePersonInput.
func (i ResponsiblePersonInput) Validate() error {
	if len(i.Name) < 2 || len(i.Name) > 64 {
		return ErrInvalid("责任人名称长度需在 2-64 之间", "name", nil)
	}
	if _, err := mail.ParseAddress(i.Email); err != nil {
		return ErrInvalid("邮箱格式不正确", "email", err)
	}
	if !phonePattern.MatchString(i.Phone) {
		return ErrInvalid("电话格式不正确（手机或区号-座机）", "phone", nil)
	}
	if len(i.Department) > 64 {
		return ErrInvalid("部门名称长度不能超过 64", "department", nil)
	}
	return nil
}

// ResponsiblePersonRepository is the persistence contract.
type ResponsiblePersonRepository interface {
	Create(ctx context.Context, p *ResponsiblePerson) error
	Update(ctx context.Context, p *ResponsiblePerson) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*ResponsiblePerson, error)
	List(ctx context.Context, q PageQuery) (*PageResult[*ResponsiblePerson], error)
}
