package service

import (
	"context"

	"github.com/juanjiTech/jframe/mod/example/dao"
	"github.com/juanjiTech/jframe/mod/example/e"
	"github.com/juanjiTech/jframe/mod/example/model"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type ItemService struct {
	items *dao.ItemDao
}

func NewItemService(items *dao.ItemDao) *ItemService {
	return &ItemService{items: items}
}

func (s *ItemService) Create(ctx context.Context, name string) (*model.Item, error) {
	if name == "" {
		return nil, e.ErrInvalidName
	}
	item := &model.Item{Name: name, Status: 1}
	if err := s.items.Create(ctx, item); err != nil {
		return nil, errors.Wrap(err, "create item")
	}
	return item, nil
}

func (s *ItemService) Get(ctx context.Context, id string) (*model.Item, error) {
	item, err := s.items.GetByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, e.ErrItemNotFound
	}
	return item, err
}

func (s *ItemService) List(ctx context.Context) ([]*model.Item, error) {
	return s.items.ListAll(ctx)
}
