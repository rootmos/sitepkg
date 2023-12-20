EXE ?= sitepkg

TARGET ?= $(abspath ./target)

GO ?= go
export GOPATH ?= $(abspath ./go)

build: FORCE
	$(GO) build -v -o $(abspath $(TARGET))/ .

run: build
	$(TARGET)/$(EXE)

clean:
	rm -rf $(TARGET)

deepclean: clean
	-chmod +w -R $(GOPATH)
	rm -rf $(GOPATH)

.PHONY: build run
.PHONY: clean deepclean
FORCE:
