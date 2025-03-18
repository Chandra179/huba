#!/bin/bash

set -e

# Load environment variables
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

function start_redis() {
  echo "Starting Redis container..."
  docker-compose up -d redis
}

function stop_redis() {
  echo "Stopping Redis container..."
  docker-compose stop redis
}

function redis_cli() {
  echo "Opening Redis CLI..."
  local password_arg=""
  if [ -n "$REDIS_PASSWORD" ]; then
    password_arg="-a $REDIS_PASSWORD"
  fi
  docker-compose exec redis redis-cli $password_arg
}

function flush_redis() {
  echo "Flushing Redis database..."
  local password_arg=""
  if [ -n "$REDIS_PASSWORD" ]; then
    password_arg="-a $REDIS_PASSWORD"
  fi
  docker-compose exec redis redis-cli $password_arg FLUSHALL
  echo "Redis database flushed successfully"
}

function redis_logs() {
  echo "Showing Redis logs..."
  docker-compose logs -f redis
}

function redis_info() {
  echo "Redis container info:"
  local password_arg=""
  if [ -n "$REDIS_PASSWORD" ]; then
    password_arg="-a $REDIS_PASSWORD"
  fi
  docker-compose exec redis redis-cli $password_arg INFO
}

case "$1" in
  start)
    start_redis
    ;;
  stop)
    stop_redis
    ;;
  cli)
    redis_cli
    ;;
  flush)
    flush_redis
    ;;
  logs)
    redis_logs
    ;;
  info)
    redis_info
    ;;
  *)
    echo "Usage: $0 {start|stop|cli|flush|logs|info}"
    exit 1
    ;;
esac 