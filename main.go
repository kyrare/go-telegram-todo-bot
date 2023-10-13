package main

import (
	"errors"
	"fmt"
	botApi "github.com/kyrare/go-telegram-bot-api"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"strconv"
)

func main() {
	db := initDB()

	secret := os.Getenv("BOT_SECRET")

	bot, err := botApi.New(secret)

	if err != nil {
		panic(err)
	}

	initBotCommands(&bot, db)

	bot.Run()
}

func initDB() *gorm.DB {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(mysql:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("MYSQL_USER"),
		os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("MYSQL_DATABASE"),
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	return db
}

func initBotCommands(bot *botApi.Bot, db *gorm.DB) {
	bot.Command("ping", func(message botApi.Message) {
		bot.SendMessage(message.Chat.Id, "pong")
	}).Command("start", func(message botApi.Message) {
		user, err := checkUser(message.From, db)

		if err == nil {
			bot.SendMessage(message.Chat.Id, fmt.Sprintf("Добро пожаловать в Список дел, %s!\n\nДоступные команды:\nping - ping/pong\ntodo - show todo list\nlist - show all list\nadd  - add new todo item\ncheck - check item\ndelete - delete item ", user.FirstName))
		} else {
			bot.SendMessage(message.Chat.Id, "В данный момент бот не работает, попробуйте позже")
		}
	}).Command("add", func(message botApi.Message) {
		err := addTodo(message, db)

		sendMessageWithError(bot, message.Chat.Id, "Дело добавлено", err, "Дело НЕ добавлено, попробуйте еще раз")
	}).Command("delete", func(message botApi.Message) {
		err := deleteTodo(message, db)

		sendMessageWithError(bot, message.Chat.Id, "Дело удалено", err, "Дело НЕ удалено, попробуйте еще раз")
	}).Command("check", func(message botApi.Message) {
		err := checkedTodo(message, db)

		sendMessageWithError(bot, message.Chat.Id, "Дело отмечено как завершенное", err, "Дело НЕ отмечено как завершенное, попробуйте еще раз")
	}).Command("todo", func(message botApi.Message) {
		todos, err := getTodos(message, true, db)

		if err != nil {
			bot.SendMessage(message.Chat.Id, "В данный момент бот не работает, попробуйте позже")
		}

		bot.SendMessage(message.Chat.Id, todosToStr(todos))
	}).Command("list", func(message botApi.Message) {
		todos, err := getTodos(message, false, db)

		if err != nil {
			bot.SendMessage(message.Chat.Id, "В данный момент бот не работает, попробуйте позже")
		}

		bot.SendMessage(message.Chat.Id, todosToStr(todos))
	})
}

// не уверен в пральности реализации такиз методов в Go, возможно можно как-то проще, пока так
func sendMessageWithError(bot *botApi.Bot, chatId int, message string, err error, defaultError string) {
	if err == nil {
		bot.SendMessage(chatId, message)
	} else {
		if _, ok := err.(*ValidationError); ok {
			bot.SendMessage(chatId, err.Error())
		} else {
			bot.SendMessage(chatId, defaultError)
		}
	}
}

func checkUser(from botApi.User, db *gorm.DB) (User, error) {
	var user User

	tx := db.Where(User{TelegramId: from.Id}).Assign(User{FirstName: from.FirstName, UserName: from.UserName}).FirstOrCreate(&user)

	return user, tx.Error
}

func addTodo(message botApi.Message, db *gorm.DB) error {
	user, err := checkUser(message.From, db)

	if err != nil {
		return err
	}

	text := message.BotCommandArgument()

	if len(text) == 0 {
		return errors.New("не передан текст для дела")
	}

	todo := Todo{UserId: user.ID, Text: message.BotCommandArgument()}

	tx := db.Create(&todo)

	return tx.Error
}

func deleteTodo(message botApi.Message, db *gorm.DB) error {
	todoIdStr := message.BotCommandArgument()

	if len(todoIdStr) == 0 {
		return &ValidationError{text: "для удаления дела необходимо передать его ID"}
	}

	todoId, err := strconv.Atoi(todoIdStr)

	if err != nil || todoId <= 0 {
		return &ValidationError{text: "для удаления дела необходимо передать его ID", Err: err}
	}

	user, err := checkUser(message.From, db)

	if err != nil {
		return err
	}

	var todo Todo
	tx := db.Where("user_id = ?", user.ID).Where("id = ?", todoId).First(&todo)

	if tx.Error != nil {
		return tx.Error
	}

	tx = db.Delete(&todo)

	return tx.Error
}

func checkedTodo(message botApi.Message, db *gorm.DB) error {
	todoIdStr := message.BotCommandArgument()

	if len(todoIdStr) == 0 {
		return &ValidationError{text: "для завершения дела необходимо передать его ID"}
	}

	todoId, err := strconv.Atoi(todoIdStr)

	if err != nil || todoId <= 0 {
		return &ValidationError{text: "для завершения дела необходимо передать его ID", Err: err}
	}

	user, err := checkUser(message.From, db)

	if err != nil {
		return err
	}

	var todo Todo
	tx := db.Where("user_id = ?", user.ID).Where("id = ?", todoId).First(&todo)

	if tx.Error != nil {
		return tx.Error
	}

	tx = db.Model(&todo).Update("checked", 1)

	return tx.Error
}

func getTodos(message botApi.Message, onlyUnchecked bool, db *gorm.DB) ([]Todo, error) {
	user, err := checkUser(message.From, db)

	var users []Todo

	if err != nil {
		return users, err
	}

	tx := db.Where("user_id = ?", user.ID)

	if onlyUnchecked {
		tx.Where("checked is null or checked = 0")
	}

	// todo pagination
	tx.Order("created_at ASC, id").Limit(20).Find(&users)

	return users, tx.Error
}

func todosToStr(todos []Todo) string {
	if len(todos) == 0 {
		return "Список дел пуст, используйте команду /add для добавления дела"
	}

	result := "Список дел:\n"

	for i, todo := range todos {
		var checked string
		if todo.Checked {
			checked = "x"
		} else {
			checked = "  "
		}

		result += fmt.Sprintf("%d) [%s] %s (%d)\n", i+1, checked, todo.Text, todo.ID)
	}

	return result
}
