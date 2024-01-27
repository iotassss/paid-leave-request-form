package model

import (
	"fmt"

	"github.com/iotassss/paid-leave-request-form/config"

	_ "github.com/go-sql-driver/mysql"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Db *gorm.DB

var err error

const (
	tableNameTodo = "todos"
)

func init() {

	Db, err = gorm.Open(mysql.Open(config.Config.DbName))

	if err != nil {
		fmt.Errorf("error:%v", err)
	}
	// Db.AutoMigrate(&Todo{})

}
