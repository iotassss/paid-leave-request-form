package todoController

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/iotassss/paid-leave-request-form/internal/model"

	"github.com/gin-gonic/gin"
)

// todo list
func List(c *gin.Context) {
	var todos []model.Todo
	model.Db.Find(&todos)

	c.HTML(http.StatusOK, "list.html", gin.H{
		"title": "Todo",
		"todos": todos,
	})
}

// todo create
func Create(c *gin.Context) {
	model.CreateTodo(c.PostForm("content"))
	c.Redirect(http.StatusMovedPermanently, "/todos/list")
}

// todo edit
func Edit(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		log.Fatalln(err)
	}
	todo, _ := model.GetTodo(id)

	c.HTML(http.StatusOK, "edit.html", gin.H{
		"title": "Todo",
		"todo":  todo,
	})
}

// todo update
func Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.PostForm("id"))
	content := c.PostForm("content")
	todo, _ := model.GetTodo(id)
	todo.Content = content
	model.UpdateTodo(todo)

	c.Redirect(http.StatusMovedPermanently, "/todos/list")
}

// todo delete
func Delete(c *gin.Context) {
	fmt.Println("destroy")
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		log.Fatalln(err)
	}
	model.DeleteTodo(id)

	c.Redirect(http.StatusMovedPermanently, "/todos/list")
}
