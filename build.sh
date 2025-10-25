#!/bin/sh

# Build the Docker image
docker image build -t forum:latest .

# Stop and remove any existing container named 'forum'
docker container stop forum 2>/dev/null || true
docker container rm forum 2>/dev/null || true

# Run the container in detached mode, map port 8080, and mount the database file
docker container run -d -p 8080:8080 --name forum -v "$(pwd)/forum.db:/app/forum.db" forum:latest 