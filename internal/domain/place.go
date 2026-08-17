package domain

import (
	"context"
	"regexp"
	"time"
)

// PlaceType represents the high-level classification of a venue.
type PlaceType string

const (
	PlaceTypeLibrary   PlaceType = "library"
	PlaceTypeExhibit   PlaceType = "exhibit"
	PlaceTypeMuseum    PlaceType = "museum"
	PlaceTypeCommunity PlaceType = "community"
	PlaceTypeOther     PlaceType = "other"
)

// Place represents a physical venue that hosts facilities.
type Place struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      PlaceType `json:"type"`
	Address   string    `json:"address"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`
}

// PlaceInput is the validated input for creating or updating a place.
type PlaceInput struct {
	Name    string    `json:"name"`
	Type    PlaceType `json:"type"`
	Address string    `json:"address"`
	Code    string    `json:"code"`
}

var placeCodePattern = regexp.MustCompile(`^[A-Z0-9-]{3,32}$`)

// Validate enforces field-level invariants for PlaceInput.
func (i PlaceInput) Validate() error {
	if len(i.Name) < 2 || len(i.Name) > 128 {
		return ErrInvalid("场所名称长度需在 2-128 之间", "name", nil)
	}
	switch i.Type {
	case PlaceTypeLibrary, PlaceTypeExhibit, PlaceTypeMuseum, PlaceTypeCommunity, PlaceTypeOther:
	default:
		return ErrInvalid("场所类型不在白名单内", "type", nil)
	}
	if i.Address == "" {
		return ErrInvalid("场所地址不能为空", "address", nil)
	}
	if !placeCodePattern.MatchString(i.Code) {
		return ErrInvalid("场所编码需为 3-32 位大写字母数字或短横线", "code", nil)
	}
	return nil
}

// PlaceRepository is the persistence contract for places.
type PlaceRepository interface {
	Create(ctx context.Context, p *Place) error
	Update(ctx context.Context, p *Place) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*Place, error)
	List(ctx context.Context, q PageQuery) (*PageResult[*Place], error)
	GetByCode(ctx context.Context, code string) (*Place, error)
}
