MAKEFLAGS += -j2
.PHONY: run install clean server tailwind

# Run server
run: server tailwind

install:
	@npm install -D tailwindcss
	@npm install @tailwindcss/forms && npm install @tailwindcss/typography

# Clean process left by wgo in server command
# This is only needed because the inner process from templ generate does not stop properly
# With `go run .` i works, but with the filewatcher (wgo) it does not kill the inner process
clean:
	@clear; \
	PID=$$(pgrep -f htmxConcept); \
	if [ -n "$$PID" ]; then \
		kill -9 $$PID; \
		echo "Killed process $$PID"; \
	else \
		echo "No programm running with name htmxConcept"; \
	fi;

# Run server with hot reload and go file watcher
# With --proxy="http://localhost:2323" (which opens the browser tab with the website) it seems
# to be much more prone to the error of not quitting processes.
# server: clean
#	templ generate --watch --cmd 'wgo run . -name htmxConcept'
server:
	wgo -file .go -file .templ -xfile _templ.go clear :: templ generate :: go run . -name htmxConcept

# Run tailwind watcher
tailwind:
	@npx @tailwindcss/cli -i ./view/static/styles/index.css -o ./view/static/styles/output.css --watch