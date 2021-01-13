package checker

import "github.com/jinzhu/gorm"

type SignCommonReq struct {
	gorm.Model
	FromAddress string `gorm:"type:varchar(255);not null;default:'';"`
	Nonce       uint32 `gorm:"type:int(10);not null;default:0;INDEX"`
	Symbol      string `gorm:"type:varchar(255);not null;default:'';"`
	To          string `gorm:"type:varchar(255);not null;default:'';"`
	Amount      string `gorm:"type:varchar(255);not null;default:'';"`
	Random      uint32 `gorm:"type:int(10);not null;default:0;INDEX"`
	HexData     string `gorm:"type:varchar(255);not null;default:'';"`
}

type TronSignReq struct {
	ID          uint   `gorm:"primary_key"`
	FromAddress string `gorm:"type:varchar(255);not null;default:'';"`
	Symbol      string `gorm:"type:varchar(255);not null;default:'';"`
	To          string `gorm:"type:varchar(255);not null;default:'';"`
	Amount      string `gorm:"type:varchar(255);not null;default:'';"`
	Random      uint32 `gorm:"type:int(10);not null;default:0;INDEX"`
	HexData     string `gorm:"type:varchar(255);not null;default:'';"`
	CreateTime  int64  `gorm:"type:int(10);not null;default:0;INDEX"`
}
