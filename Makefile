EXE ?= sitepkg

TARGET ?= $(abspath ./target)

GO ?= go
export GOPATH ?= $(abspath ./go)

build: FORCE
	$(GO) build -v -o $(abspath $(TARGET))/ ./...

run: build
	$(TARGET)/$(EXE)

test:
	$(GO) test $(if $(VERBOSE),-v,) ./...

update:
	$(GO) get -u
	$(GO) mod tidy

SUDO ?=
DOCKER ?= docker
docker:
	$(SUDO) $(DOCKER) build --progress=plain --iidfile=.image .

clean:
	rm -rf $(TARGET)

deepclean: clean
	-chmod +w -R $(GOPATH)
	rm -rf $(GOPATH)

.PHONY: build run test
.PHONY: update
.PHONY: clean deepclean
FORCE:
