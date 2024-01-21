package model

import (
	"testing"

	"github.com/iotassss/paid-leave-request-form/config"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestCreateTodo(t *testing.T) {
	db, err := gorm.Open(mysql.Open(config.Config.DbName), &gorm.Config{})
	assert.NoError(t, err)
	Db = db

	content := "Test Todo"
	err = CreateTodo(content)
	assert.NoError(t, err)
}

func TestDeleteTodo(t *testing.T) {
	db, err := gorm.Open(mysql.Open(config.Config.DbName), &gorm.Config{})
	assert.NoError(t, err)
	Db = db

	id := 1
	err = DeleteTodo(id)
	assert.NoError(t, err)
}

func TestGetTodo(t *testing.T) {
	db, err := gorm.Open(mysql.Open(config.Config.DbName), &gorm.Config{})
	assert.NoError(t, err)
	Db = db

	id := 1
	todo, err := GetTodo(id)
	assert.NoError(t, err)
	assert.NotNil(t, todo)
}

func TestUpdateTodo(t *testing.T) {
	db, err := gorm.Open(mysql.Open(config.Config.DbName), &gorm.Config{})
	assert.NoError(t, err)
	Db = db

	todo := Todo{
		Model:   gorm.Model{ID: 1},
		Content: "Updated Todo",
	}
	err = UpdateTodo(todo)
	assert.NoError(t, err)
}
