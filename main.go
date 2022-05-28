package main

import (
	"bufio"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"
	"gopkg.in/gookit/color.v1"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var db *sql.DB

type Note struct {
	Id        int
	Title     string
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func createTable() {
	query := "create table if not exists notes\n(\n    id         serial\n        constraint notes_pk\n            primary key,\n    title      text                                not null,\n    note       text                                not null,\n    created_at timestamp default current_timestamp not null,\n    updated_at timestamp default current_timestamp not null\n);\n\ncreate unique index if not exists notes_id_uindex\n    on notes (id);"
	_, err := db.Exec(query)

	if err != nil {
		println("Не удалось создать схему базы данных.")
		os.Exit(-1)
	}
}

func init() {
	connStr := "user=postgres password=postgres dbname=notes sslmode=disable"
	_db, err := sql.Open("postgres", connStr)

	if err != nil {
		println("Не удалось подключиться к базе данных.")
		os.Exit(-1)
	}

	db = _db

	createTable()
}

func getNotes() []Note {
	query := "select id, title, note, created_at, updated_at from notes order by created_at"
	rows, err := db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()
	notes := []Note{}

	for rows.Next() {
		note := Note{}
		err := rows.Scan(&note.Id, &note.Title, &note.Note, &note.CreatedAt, &note.UpdatedAt)
		if err != nil {
			println("Не получилось получить заметку.")
			continue
		}

		notes = append(notes, note)
	}

	return notes
}

func printNotes(notes []Note, showText bool, showTotal bool) {
	for _, note := range notes {
		print("Номер заметки: ")
		color.Green.Println(note.Id)
		print("Название: ")
		color.Green.Println(note.Title)
		print("Дата создания: ")
		color.Green.Println(getFormattedTime(note.CreatedAt))

		if note.CreatedAt != note.UpdatedAt {
			print("Дата последнего изменения: ")
			color.Green.Println(getFormattedTime(note.UpdatedAt))
		}

		if showText {
			println("Текст заметки:")
			println(note.Note)
		}

		if len(notes) != 1 {
			println()
		}
	}

	if showTotal {
		print("Всего заметок: ")
		color.Yellow.Println(len(notes))
	}
}

func createNote() {
	query := "insert into notes (title, note) values ($1, $2);"
	reader := bufio.NewReader(os.Stdin)

	println("Введите название заметки:")
	title, _ := reader.ReadString('\n')
	title = strings.TrimSpace(title)

	println("Введите текст заметки:")
	note, _ := reader.ReadString('\n')
	note = strings.TrimSpace(note)

	_, err := db.Exec(query, title, note)
	if err != nil {
		println(err)
		println("Не получилось создать заметку.")
		os.Exit(-1)
	}
}

func getFormattedTime(ts time.Time) string {
	loc, _ := time.LoadLocation("Asia/Novosibirsk")
	return ts.In(loc).Format("02.01.2006 15:04")
}

func getUpdateQuery(field string) string {
	return "update notes set " + field + " = $1, updated_at = current_timestamp where id = $2"
}

func checkForNote(id int) bool {
	query := "select * from notes where id = $1"
	rows, err := db.Query(query, id)
	if err != nil {
		println("Не удалось найти заметку: не смог выполнить запрос.")
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}

	return count == 1
}

func updateNote(id int) {
	reader := bufio.NewReader(os.Stdin)

	println("Введите новое название заметки (нажмите Enter, чтобы пропустить):")
	title, _ := reader.ReadString('\n')
	title = strings.TrimSpace(title)
	if len(title) != 0 {
		_, err := db.Exec(getUpdateQuery("title"), title, id)
		if err != nil {
			println("Не удалось обновить заголовок заметки.")
		} else {
			println("Название обновлено!")
		}
	}

	println("Введите новый текст заметки (нажмите Enter, чтобы пропустить):")
	note, _ := reader.ReadString('\n')
	note = strings.TrimSpace(note)
	if len(note) != 0 {
		_, err := db.Exec(getUpdateQuery("note"), note, id)
		if err != nil {
			println("Не удалось обновить текст заметки.")
		} else {
			println("Текст обновлен!")
		}
	}
}

func deleteNote(id int) {
	query := "delete from notes where id = $1;"
	_, err := db.Exec(query, id)
	if err != nil {
		println("Не удалось удалить заметку!")
		os.Exit(-1)
	}
}

func getNote(id int) Note {
	query := "select id, title, note, created_at, updated_at from notes where id = $1"
	rows, err := db.Query(query, id)
	if err != nil {
		println("Не удалось получить заметку: запрос выполнен с ошибкой.")
		os.Exit(-1)
	}
	defer rows.Close()
	note := Note{}

	if !rows.Next() {
		println("Заметка не найдена!")
		os.Exit(-1)
	}

	errScan := rows.Scan(&note.Id, &note.Title, &note.Note, &note.CreatedAt, &note.UpdatedAt)
	if errScan != nil {
		println("Не получилось получить заметку.")
		os.Exit(-1)
	}

	return note
}

func main() {
	app := &cli.App{
		Name:  "gonotes",
		Usage: "Консольное приложение для заметок, написанное на Go и использующее PostgreSQL.",
		Action: func(ctx *cli.Context) error {
			notes := getNotes()
			printNotes(notes, false, true)

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "get",
				Aliases: []string{"g"},
				Usage:   "получить заметку",
				Action: func(ctx *cli.Context) error {
					idStr := ctx.Args().First()

					id, err := strconv.Atoi(idStr)
					if err != nil {
						println("Введен некорректный идентификатор заметки.")
						os.Exit(-1)
					}

					if !checkForNote(id) {
						println("Заметка с таким идентификатором не найдена!")
						os.Exit(-1)
					}

					note := getNote(id)
					notes := []Note{}

					notes = append(notes, note)
					printNotes(notes, true, false)

					return nil
				},
			},
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "создать новую заметку",
				Action: func(ctx *cli.Context) error {
					createNote()
					println("Заметка создана!")
					return nil
				},
			},
			{
				Name:    "update",
				Aliases: []string{"u"},
				Usage:   "обновить заметку",
				Action: func(ctx *cli.Context) error {
					idStr := ctx.Args().First()

					id, err := strconv.Atoi(idStr)
					if err != nil {
						println("Введен некорректный идентификатор заметки.")
						os.Exit(-1)
					}

					if !checkForNote(id) {
						println("Заметка с таким идентификатором не найдена!")
						os.Exit(-1)
					}

					updateNote(id)
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "вывести все заметки с их содержанием",
				Action: func(ctx *cli.Context) error {
					notes := getNotes()
					printNotes(notes, true, true)

					return nil
				},
			},
			{
				Name:    "remove",
				Aliases: []string{"r"},
				Usage:   "удалить заметку",
				Action: func(ctx *cli.Context) error {
					idStr := ctx.Args().First()

					id, err := strconv.Atoi(idStr)
					if err != nil {
						println("Введен некорректный идентификатор заметки.")
						os.Exit(-1)
					}

					if !checkForNote(id) {
						println("Заметка с таким идентификатором не найдена!")
						os.Exit(-1)
					}

					deleteNote(id)
					println("Заметка удалена.")
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
