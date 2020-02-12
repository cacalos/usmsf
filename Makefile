PKG_LIST := $(shell go list ./... | grep -vE "test|cli\/cmd")
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
.PHONY: all dep build clean test coverage coverhtml lint
all: build

make_report_dir:
	mkdir -p reports
	mkdir -p reports/unittest
	mkdir -p reports/coverage

lint: ## Lint the files
	@golint -set_exit_status ${PKG_LIST}

test: ## Run unittests
	@go test -short ${PKG_LIST}

install_junittest_dep: ## install junit test dependency
	@go get github.com/jstemmer/go-junit-report

junittest: install_junittest_dep make_report_dir ## Run unittests and generate junit reports
	@go test -v ${PKG_LIST} 2>&1 | go-junit-report > reports/unittest/report.xml

race: dep ## Run data race detector
	@go test -race -short ${PKG_LIST}

msan: dep ## Run memory sanitizer
	@go test -msan -short ${PKG_LIST}

install_dep_cover: ## coverage dependency
	@go get github.com/ory/go-acc

coverage: install_dep_cover ## Generate global code coverage report
	@go-acc ${PKG_LIST}

coverhtml: coverage ## Generate global code coverage report in HTML
	@go tool cover -html=coverage.txt -o reports/coverage/coverage.html

dep_coverjenkins: ## converting tool for cobertura
	@go get github.com/t-yuki/gocover-cobertura

coverjenkins: coverage dep_coverjenkins ## converting coverage data to cobertura format
	gocover-cobertura < coverage.txt > reports/coverage/coverage.xml

dep: ## Get the dependencies
	@go get -v -d ./...

build: dep ## Build the binary file
	@go build -o ./cmds/hello -v ./cmds

clean: ## Remove previous build
	@go clean -v ./...

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
