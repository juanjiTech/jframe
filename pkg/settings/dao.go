package settings

import (
	"context"

	"github.com/juanjiTech/jframe/pkg/stdao"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DAO struct {
	stdao.Std[*Model]
}

func InitDAO(db *gorm.DB) (*DAO, error) {
	dao := &DAO{}
	if err := dao.Init(db); err != nil {
		return nil, errors.WithStack(err)
	}
	return dao, nil
}

func (o *DAO) GetByKey(ctx context.Context, key string) (*Model, error) {
	s := &Model{}
	return s, o.GetTxFromCtx(ctx).WithContext(ctx).Where("key = ?", key).First(s).Error
}

func (o *DAO) UpdateOrCreate(ctx context.Context, s *Model) error {
	return o.GetTxFromCtx(ctx).WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(s).Error
}
