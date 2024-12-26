#!/bin/bash

# Restart the containers
echo "Restarting containers..."
docker-compose restart go-app nginx-proxy
