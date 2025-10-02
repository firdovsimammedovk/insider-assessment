# Insider Messaging Service

This project is an automatic message sending system. It periodically sends out messages from a PostgreSQL database and exposes a REST API to control the sending process and view sent messages. The service uses **Golang**, **PostgreSQL**, **Redis**, and integrates with an external webhook endpoint to simulate sending messages.

## Features
- **Automatic Scheduling**: On startup, the service begins sending unsent messages from the database (2 messages every 2 minutes) without using any cron jobs (implemented with Go's scheduling).
- **REST API**: Provides endpoints to start/stop the automatic sender and to retrieve the list of sent messages.
- **Database**: PostgreSQL is used to store messages (content, recipient, status). Message content is limited to 160 characters.
- **Caching**: After sending a message, the external message ID and send time are cached in Redis (as a bonus feature).

### How can you insert records to db:
```shell
docker compose exec db psql -U postgres -d postgres -c "
INSERT INTO messages (recipient, content) VALUES
('+905521871212','Insider - Project A'),
('+905527128377','Insider - Project B'),
('+905521782372','Insider - Project C');"
```