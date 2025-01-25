package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

var token = "" //Add BotToken from BotFather here

var (
	currentDriver  string
	currentConnStr string
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

	// Prepare result string
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Columns: %s\n\n", strings.Join(columns, " | ")))
	result.WriteString("---\n")

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

		// Convert row to string
		rowStr := make([]string, len(columns))
		for i, val := range vals {
			if val == nil {
				rowStr[i] = "NULL"
			} else {
				rowStr[i] = fmt.Sprintf("%v", val)
			}
		}
		result.WriteString(fmt.Sprintf("%s\n", strings.Join(rowStr, " | ")))
	}

	return result.String(), nil
}

func main() {
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
					"🤖 SQL Telegram Bot\n\n"+
						"Usage:\n"+
						"/connect [connection_string]\n"+
						"/query [SQL query]\n\n"+
						"Supported databases: PostgreSQL, MySQL, SQL Server")
				bot.Send(helpMsg)

			case "connect":
				connStr := update.Message.CommandArguments()
				if connStr == "" {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
						"❌ Please provide a connection string")
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
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
						"❌ Unsupported database type in connection string")
					bot.Send(errMsg)
					continue
				}

				// Test the connection
				err := testConnection(driver, connStr)
				if err != nil {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("❌ Connection failed: %v", err))
					bot.Send(errMsg)
					continue
				}

				// Store connection details
				currentDriver = driver
				currentConnStr = connStr

				msg := tgbotapi.NewMessage(update.Message.Chat.ID,
					fmt.Sprintf("✅ Connected to %s database successfully!",
						strings.ToUpper(driver)))
				bot.Send(msg)

			case "query":
				query := update.Message.CommandArguments()
				if query == "" {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
						"❌ Please provide a SQL query")
					bot.Send(errMsg)
					continue
				}

				// Execute query
				result, err := executeQuery(query)
				if err != nil {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("❌ Error: %v", err))
					bot.Send(errMsg)
					continue
				}

				// Send result back to Telegram
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, result)
				bot.Send(msg)

			default:
				helpMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
					"❓ Unknown command. Use /start for help.")
				bot.Send(helpMsg)
			}
		}
	}
}
