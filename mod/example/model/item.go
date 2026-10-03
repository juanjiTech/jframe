package model

import "github.com/juanjiTech/jframe/pkg/stdao"

// Item 演示用业务模型：嵌入 stdao.Model 获得 ULID 主键、时间戳与软删除。
type Item struct {
	stdao.Model
	Name   string `json:"name" gorm:"size:128;not null"`
	Status int    `json:"status" gorm:"not null;default:1"`
}

func (Item) TableName() string { return "example_items" }

// CreateItemReq 创建请求 DTO（无 GORM tag）。
type CreateItemReq struct {
	Name string `json:"name"`
}

// ItemResp 响应 DTO。
type ItemResp struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}
