build:
	go build -o bin/exchange

run: build
	./bin/exchange

test:
	go test -v ./...

kill:
	@PID=$$(lsof -ti tcp:3004); \
	if [ -n "$$PID" ]; then \
			echo "Killing process on port 3004 with PID: $$PID"; \
			kill -9 $$PID; \
	else \
			echo "No process running on port 3004"; \
	fi