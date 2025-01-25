package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var (
	currentDriver  string
	currentConnStr string
	waitForConnStr bool // Flag to indicate if we're waiting for a connection string
)

func testConnection(driver, connStr string) error {
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return fmt.Errorf("connection error: %v", err)
	}
	defer db.Close()

	return db.Ping()
}

func executeQuery(query string) (string, error) {
	if currentConnStr == "" {
		return "", fmt.Errorf("no database connection established. Use /connect first")
	}

	// Open database connection
	db, err := sql.Open(currentDriver, currentConnStr)
	if err != nil {
		return "", fmt.Errorf("connection error: %v", err)
	}
	defer db.Close()

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return "", fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("columns error: %v", err)
	}

	// Prepare a slice to hold the rows
	var result []map[string]interface{}

	// Scan rows
	for rows.Next() {
		// Create a slice of interface{} to hold row values
		vals := make([]interface{}, len(columns))
		valPtrs := make([]interface{}, len(columns))
		for i := range columns {
			valPtrs[i] = &vals[i]
		}

		// Scan row values
		if err := rows.Scan(valPtrs...); err != nil {
			return "", fmt.Errorf("scan error: %v", err)
		}

		// Create a map to hold column-value pairs
		rowMap := make(map[string]interface{})
		for i, val := range vals {
			if val == nil {
				rowMap[columns[i]] = nil
			} else {
				rowMap[columns[i]] = val
			}
		}
		// Append row map to the result slice
		result = append(result, rowMap)
	}

	// Format the result into the requested output style
	var sb strings.Builder
	for _, row := range result {
		for key, value := range row {
			sb.WriteString(fmt.Sprintf("〔%s〕%v\n", key, value))
		}
		sb.WriteString("──\n")
	}

	return sb.String(), nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	token := os.Getenv("TOKEN")

	// Replace with your Telegram Bot Token
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Start receiving updates
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Check for command
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				helpMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
					"*𝖙𝖊𝖖𝖑* 🫠\n\n"+
						"`Các chức năng:`\n"+
						"*/connect*\n"+
						"*/query* `〔SQL Query〕`")
				helpMsg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(helpMsg)

			case "connect":
				// Ask the user to enter the connection string
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hãy nhập chuỗi cơ sở dữ liệu.")
				bot.Send(msg)

				// Wait for the user's response (connection string)
				update = <-updates // Get the next update (user input)

				connStr := update.Message.Text
				if connStr == "" {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Chuỗi kết nối không hợp lệ. Vui lòng thử lại.")
					bot.Send(errMsg)
					continue
				}

				// Determine database driver based on connection string
				var driver string
				switch {
				case strings.Contains(connStr, "postgresql://") || strings.Contains(connStr, "postgres://"):
					driver = "postgres"
				case strings.Contains(connStr, "@tcp("):
					driver = "mysql"
				case strings.Contains(connStr, "sqlserver://") || strings.Contains(connStr, "server="):
					driver = "sqlserver"
				default:
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Database không được hỗ trợ. Vui lòng thử lại với PostgreSQL, MySQL hoặc SQL Server.")
					bot.Send(errMsg)
					continue
				}

				// Test the connection
				err := testConnection(driver, connStr)
				if err != nil {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Lỗi kết nối: %v", err))
					bot.Send(errMsg)
					continue
				}

				// Store connection details
				currentDriver = driver
				currentConnStr = connStr

				// Confirm connection
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Đã kết nối thành công database: %s", strings.ToUpper(driver)))
				bot.Send(msg)

			case "query":
				query := update.Message.CommandArguments()
				if query == "" {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
						"Hãy nhập lệnh truy vấn.")
					bot.Send(errMsg)
					continue
				}

				// Execute query
				result, err := executeQuery(query)
				if err != nil {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Lỗi xảy ra: %v", err))
					bot.Send(errMsg)
					continue
				}

				// Send result back to Telegram
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, result)
				bot.Send(msg)

			default:
				helpMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
					"Rất tiếc, bạn đã gặp lỗi khi sử dụng chức năng này. Hãy thử nhập /start để xem hướng dẫn sử dụng.")
				bot.Send(helpMsg)
			}
		}

		// Handle the connection string input
		if waitForConnStr && update.Message.Text != "" {
			// Store the provided connection string
			connStr := update.Message.Text
			if connStr == "" {
				errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Chuỗi kết nối không hợp lệ.")
				bot.Send(errMsg)
				continue
			}

			// Determine database driver based on connection string
			var driver string
			switch {
			case strings.Contains(connStr, "postgresql://") || strings.Contains(connStr, "postgres://"):
				driver = "postgres"
			case strings.Contains(connStr, "@tcp("):
				driver = "mysql"
			case strings.Contains(connStr, "sqlserver://") || strings.Contains(connStr, "server="):
				driver = "sqlserver"
			default:
				errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Database không được hỗ trợ")
				bot.Send(errMsg)
				continue
			}

			// Test the connection
			err := testConnection(driver, connStr)
			if err != nil {
				errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Lỗi kết nối: %v", err))
				bot.Send(errMsg)
				continue
			}

			// Store connection details
			currentDriver = driver
			currentConnStr = connStr

			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				fmt.Sprintf("Đã kết nối thành công database: %s", strings.ToUpper(driver)))
			bot.Send(msg)

			// Reset the flag
			waitForConnStr = false
		}
	}
}
