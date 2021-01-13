package checker

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"log"
	"time"
)

type _db struct {
	gormDB *gorm.DB
	sqlitPath string
	sl *log.Logger
}

var privDb _db

func Init(path string, sl *log.Logger) {
	privDb.sqlitPath = path
	privDb.sl = sl

	// 自动创建表
	getDb().AutoMigrate(&SignCommonReq{}, new(TronSignReq))
}

func getDb() *gorm.DB{

	if privDb.gormDB != nil {
		return privDb.gormDB
	}

	if privDb.sqlitPath == "" {
		privDb.sl.Fatalf("sqlite db path is required")
	}

	db, errOpen := gorm.Open("sqlite3", privDb.sqlitPath)
	if errOpen != nil {
		privDb.sl.Fatalf("sqlite3 error %s", errOpen.Error())
	}

	// 统一强制使用UTC时间，避免混乱
	gorm.NowFunc = func() time.Time {
		return time.Now().UTC()
	}

	privDb.gormDB = db
	return privDb.gormDB
}