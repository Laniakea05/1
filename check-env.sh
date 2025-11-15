#!/bin/bash
echo "=== Checking Docker Compose Environment ==="

# Проверим запущены ли контейнеры
echo "1. Checking if containers are running..."
docker-compose ps

echo ""
echo "2. Checking backend environment variables..."
docker-compose exec backend env | grep DB_

echo ""
echo "3. Checking PostgreSQL environment variables..."
docker-compose exec postgres env | grep POSTGRES_

echo ""
echo "4. Testing database connection from backend..."
docker-compose exec backend sh -c 'psql "host=postgres port=5432 user=postgres password=\$DB_PASSWORD dbname=psycho_test_system" -c "SELECT version();"'

echo ""
echo "=== Environment Check Complete ==="
