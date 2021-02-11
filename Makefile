# build config
BUILD_DIR 		?= $(abspath build)
GET_GOARCH 		 = $(word 2,$(subst -, ,$1))
GET_GOOS   		 = $(word 1,$(subst -, ,$1))
GOBUILD   		?= $(shell go env GOOS)-$(shell go env GOARCH)
GOFILES_NOCACHE  = $(shell find . -type f -name '*.go' -not -path "./cache/*")
VETARGS? 		 =-all

$(BUILD_DIR):
	mkdir -p $@

.PHONY: build
build:
	go install

.PHONY: fmt
fmt:
	@echo "=> Running go fmt" ;
	@if [ -n "`go fmt ${GOFILES_NOCACHE}`" ]; then \
		echo "[ERR] go fmt updated formatting. Please commit formatted code first."; \
		exit 1; \
	fi

.PHONY: vet
vet: fmt
	@echo "=> Running go vet $(VETARGS) ${GOFILES_NOCACHE}"
	@go vet $(VETARGS) ${GOFILES_NOCACHE} ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "[LINT] Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
	fi

BINARIES = $(addprefix $(BUILD_DIR)/docker-mirror-, $(GOBUILD))
$(BINARIES): $(BUILD_DIR)/docker-mirror-%: $(BUILD_DIR)
	@echo "=> building $@ ..."
	GOOS=$(call GET_GOOS,$*) GOARCH=$(call GET_GOARCH,$*) CGO_ENABLED=0 go build -o $@

.PHONY: dist
dist: fmt vet
	@echo "=> building ..."
	$(MAKE) -j $(BINARIES)
