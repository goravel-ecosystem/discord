# Goravel Discord Bot

## Features

- [x] goravel.dev Heartbeat
- [x] Pull Requests Notifications

## Run

1. Add webhook link in Github

Get webhook secret: https://github.com/organizations/goravel/settings/hooks

2. Create a discord bot

Get bot token: https://discord.com/developers/applications

3. Create two channels to receive warning and pull request notifications

Get channel ids

4. Create the .env file before `docker compose up` (it is bind-mounted into the container):
   ```bash
   cp .env.example .env
   # then set APP_KEY, DISCORD_BOT_TOKEN, DISCORD_*_CHANNEL_ID, GITHUB_WEBHOOK_SECRET
   ```

   > The image no longer contains a `.env`. Running it without Compose requires supplying configuration explicitly, e.g. `docker run --env-file .env ...`; otherwise the container exits with `Please initialize APP_KEY first.`

5. Run docker compose

Port: 4500

```bash
docker compose up --build -d
```
