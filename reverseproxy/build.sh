#!/bin/bash

# Build and start containers in detached mode
echo "Building and starting containers..."
docker-compose up --build -d
