Here’s an improved version of the README:

---

## Telegram Bot Setup Guide

Follow these steps to set up and run your Telegram bot:

### 1. Create Your Bot on Telegram
- Open [BotFather](https://telegram.me/BotFather) on Telegram.
- Use `/newbot` to create a new bot and get your bot's **API token**.

### 2. Clone the Repository
Clone the repository to your local machine:

```bash
git clone <repository-url>
cd <repository-directory>
```

### 3. Create a `.env` File
In the project directory, create a `.env` file with the following content:

```env
TOKEN=your-bot-token-here
```

Replace `your-bot-token-here` with the API token you received from BotFather.

### 4. Run the Bot
To start the bot, run the following Go command:

```bash
go run main.go
```

### 5. Interact with Your Bot
- Open Telegram and search for your bot.
- Start a chat with your bot by typing `/start`.

### 6. Connect to Your Database
- Type `/connect` and input your database connection string when prompted.

### 7. Query Your Database
- To query your database, use the `/query` command followed by your SQL query. For example:

```bash
/query select * from users where username like 'thinhphoenix'
```

### 8. Enjoy!
You're all set! Enjoy using the bot and querying your database.

---

Let me know if you'd like any further adjustments!
