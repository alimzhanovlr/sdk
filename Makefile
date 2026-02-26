.PHONY: install build test lint clean

# Установка CLI
install:
	go install ./cmd/microkit-cli

# Сборка CLI
build:
	mkdir -p bin
	go build -o bin/microkit ./cmd/microkit-cli

# Тесты
test:
	go test -v -race -cover ./...

# Линтер
lint:
	golangci-lint run

# Очистка
clean:
	rm -rf bin/
	
# Обновление зависимостей
deps:
	go mod tidy
	go mod download

# Запуск примера
example:
	cd examples/user-service && go run cmd/api/main.go