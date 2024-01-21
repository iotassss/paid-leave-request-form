package model

import (
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type Todo struct {
	gorm.Model
	Content string
}

func CreateTodo(content string) (err error) {

	todo := Todo{
		Content: content,
	}
	Db.Create(&todo)

	return err
}

func DeleteTodo(id int) (err error) {

	Db.Delete(&Todo{}, id)

	return err
}

func GetTodo(id int) (todo Todo, err error) {

	Db.Find(&todo, id)

	return todo, err
}

func UpdateTodo(t Todo) (err error) {

	Db.Save(&t)

	return err
}
