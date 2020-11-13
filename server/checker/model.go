package checker

import "github.com/jinzhu/gorm"

type SignCommonReq struct {
	gorm.Model
	Symbol  string `gorm:"type:varchar(255);not null;default:'';"`
	To      string `gorm:"type:varchar(255);not null;default:'';"`
	Amount  string `gorm:"type:varchar(255);not null;default:'';"`
	Random  uint32 `gorm:"type:int(10);not null;default:0;INDEX"`
	HexData string `gorm:"type:varchar(255);not null;default:'';"`
}