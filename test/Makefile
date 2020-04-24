PKG_LIST := $(shell go list ./...)
BUILD_HOME := ${PWD}

MODULE_NAME := uangel.com\\/usmsf\\/

PKG_LIST := $(shell go list ./... | grep -vE "sim\/amfnrfcli|test|cmd\/httpif|cmd\/nrfclient|cmd\/usmsfperf")
GO_FILES := $(shell find . -name "*.go" | grep -v /vendor/ | grep -v _test.go)
MAKEIMAGE_SHELL := sh ./cmd/usmsf/jenkins_build.sh


.PHONY: all lint test install_go_junit_report install_go_acc install_gocover_cobertura jenkins_dep sonarqube_unittestcoverage jenkins_coverage coverjenkins jenkins_junittest junittest coverhtml dep image build clean 
all: build

reports:
	@mkdir -p reports
reports/unittest: reports
	@mkdir -p reports/unittest
reports/coverage: reports
	@mkdir -p reports/coverage

test: ## Run unittests
	go test -v ${PKG_LIST}:q


install_go_junit_report: ## install go-junit-report dependency
	@go get github.com/jstemmer/go-junit-report
install_go_acc: ## install go-acc
	@go get github.com/ory/go-acc
install_gocover_cobertura: ## install gocover-cobertura
	@go get github.com/t-yuki/gocover-cobertura
jenkins_dep: install_go_junit_report install_go_acc install_gocover_cobertura ; ## install jenkins need tool

reports/coverage/coverage.out reports/unittest/report.txt: unitest_coverage.temporary ; ## Run unittest and coverage, generate raw data 

.INTERMEDIATE: unitest_coverage.temporary
unitest_coverage.temporary: reports/coverage reports/unittest
	set -o pipefail; go-acc -o reports/coverage/coverage.out ${PKG_LIST} -- -v $(UNITTEST_OPT) | tee reports/unittest/report.txt

reports/unittest/report.xml: reports/unittest/report.txt
	cat reports/unittest/report.txt | go-junit-report > reports/unittest/report.xml

reports/unittest/report.json: reports/unittest/report.txt
	cat reports/unittest/report.txt | go tool test2json > reports/unittest/report.json

reports/coverage/coverage.xml: reports/coverage/coverage.out
	gocover-cobertura < reports/coverage/coverage.out > reports/coverage/coverage.xml && sed -i "s/$(MODULE_NAME)//g" reports/coverage/coverage.xml

reports/coverage/coverage.html: reports/coverage/coverage.out
	go tool cover -html=reports/coverage/coverage.out -o reports/coverage/coverage.html

sonarqube_unittestcoverage: reports/unittest/report.json reports/coverage/coverage.out ; ## generate sonarqube unitest and coverage data

jenkins_coverage : reports/coverage/coverage.xml ; ## generate cobertura data

jenkins_junittest: reports/unittest/report.xml ; ## generate junit report

coverhtml: reports/coverage/coverage.xml ; ## Generate global code coverage report in HTML

dep: ## Get the dependencies
	@go get -v -d ./...

image: ## make docker image
	$(MAKEIMAGE_SHELL)

build: dep ## Build the binary file

clean: ## Remove previous build
	go clean -v ./...
	rm -rf reports


