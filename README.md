# Forum — Web Forum in Go

## Description

Forum is a learning web forum built with Go, supporting posts, comments, categories, likes/dislikes, user roles, moderation, notifications, image uploads, and Docker deployment.

## Features

- Registration and authentication (email, username, password)
- Email validation (using regular expressions), username and password validation (by Unicode character count)
- Posts and comments with protection against empty or whitespace-only content
- Categories and filtering
- Likes and dislikes (only via POST requests)
- User roles: guest, user, moderator, admin
- Moderation and reports
- Action notifications
- Image uploads for posts
- User activity page
- SQLite as the database
- Docker support (Alpine-based, CGO enabled for SQLite)

## Requirements

- Go 1.20+
- Docker

## Quick Start

### 1. Locally (Go)

```sh
cd forum
cp forum.db forum/forum.db # if needed, or it will be created automatically
cd forum/cmd/server
go run main.go
```

### 2. Docker (Recommended)

Build and run the app using the provided script:

```sh
chmod +x build.sh
./build.sh
```

This will:
- Build the Docker image (Alpine-based, CGO enabled for SQLite)
- Stop and remove any existing container named `forum`
- Run the app in a new container, mapping port 8080 and mounting your local `forum.db` for persistence

The app will be available at http://localhost:8080

#### Manual Docker Commands

If you prefer, you can run the commands manually:

```sh
docker build -t forum:latest .
docker run -d -p 8080:8080 --name forum -v $(pwd)/forum.db:/app/forum.db forum:latest
```

## Project Structure

```
forum/
  cmd/server/         # main.go — entry point
  internal/
    db/               # database logic, migrations, tests
    handlers/         # HTTP handlers
    middleware/       # Middleware (authentication, etc.)
    models/           # Data models
    config/           # Configuration
  static/             # HTML, CSS, images
  Dockerfile
  build.sh            # Build and run script for Docker
  README.md
```

## Validation & Security

- **Email:** Checked with a regular expression for valid format.
- **Username:** 3–30 characters, counts Unicode runes (e.g., emojis).
- **Password:** 6–50 characters, counts Unicode runes, leading/trailing spaces are trimmed.
- **Posts & Comments:** Cannot submit empty or whitespace-only text.
- **Delete post/comment:** Only via DELETE requests (secure, cannot delete via link).
- **Textarea:** Resizing is disabled (`resize: none`).
- **Likes/Dislikes:** Only via POST requests.

## Usage Notes

- Deleting a post/comment uses a button inside a form (DELETE), not a link.
- Reporting, creating, and editing posts/comments all use POST forms.
- All fields are strictly validated on the server.

## Tests

```sh
cd forum/internal/db
go test -v
```

## Main Commands

- `go run ./forum/cmd/server` — run the server
- `./build.sh` — build and run with Docker (recommended)
- `docker build -t forum:latest . && docker run -d -p 8080:8080 --name forum -v $(pwd)/forum.db:/app/forum.db forum:latest` — manual Docker run
- `go test ./forum/internal/db/...` — run tests

## Author

- Asset - amussabe

---

**This project was created for educational purposes.**