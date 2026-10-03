package dao

import (
	"context"

	"github.com/juanjiTech/jframe/mod/example/model"
	"github.com/juanjiTech/jframe/pkg/stdao"
	"gorm.io/gorm"
)

// ItemDao 基于框架内置 pkg/stdao，Init 时自动 AutoMigrate。
type ItemDao struct {
	stdao.Std[*model.Item]
}

func NewItemDao(db *gorm.DB) (*ItemDao, error) {
	d := &ItemDao{}
	if err := d.Init(db); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *ItemDao) GetByID(ctx context.Context, id string) (*model.Item, error) {
	item := &model.Item{}
	err := d.GetTxFromCtx(ctx).WithContext(ctx).Where("id = ?", id).First(item).Error
	return item, err
}

func (d *ItemDao) ListAll(ctx context.Context) ([]*model.Item, error) {
	var list []*model.Item
	err := d.GetTxFromCtx(ctx).WithContext(ctx).Order("created_at desc").Find(&list).Error
	return list, err
}
