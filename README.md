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

4. Initialize .env file

5. Run docker compose

Port: 4500

```bash
docker compose up --build -d
```
