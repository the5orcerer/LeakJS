.PHONY: build install clean

build:
	go build -o leakjs ./cmd/leakjs

install: build
	sudo mv leakjs /usr/local/bin/
	@echo "LeakJS installed successfully"

clean:
	rm -f leakjs

uninstall:
	sudo rm -f /usr/local/bin/leakjs
	@echo "LeakJS uninstalled"