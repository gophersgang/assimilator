.PHONY: build-go watch-go get-go-deps get-js-deps get-go-tools get-atom-plugins
.PHONY: test-js lint-go lint-js help
.DEFAULT_GOAL := help

build-go:  ## Build Golang project
	go build -o bin/assimilator

watch-go:  ## Live reload Golang code
	gin --bin bin/assimilator-gin --immediate

get-go-deps:  ## Install Golang dependencies
	glide install

get-js-deps:  ## Install NodeJS dependencies
	cd ui && npm install

get-go-tools:  ## Install Golang development tools
	go get github.com/codegangsta/gin
	go get -u golang.org/x/tools/cmd/goimports
	go get -u golang.org/x/tools/cmd/gorename
	go get -u github.com/sqs/goreturns
	go get -u github.com/nsf/gocode
	go get -u github.com/zmb3/gogetdoc
	go get -u github.com/rogpeppe/godef
	go get -u golang.org/x/tools/cmd/guru
	go get -u github.com/derekparker/delve/cmd/dlv
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
	#go get github.com/smartystreets/goconvey
	go get github.com/onsi/ginkgo/ginkgo
	go get github.com/mattn/goveralls

get-atom-plugins:  ## Install plugins for Atom editor
	apm install go-plus hyperclick go-debug go-signature-statusbar

test-go:  ## Run Go tests
	ginkgo -r -cover

test-watch-go:  ## Continuous testing for Go sources
	# goconvey -excludedDirs vendor,ui,.volumes,templates
	ginkgo watch -r -notify

test-js:  ## Run JavaScript tests
	@echo "--> Building static assets"
	# cd ui && SENTRY_EXTRACT_TRANSLATIONS=1 node_modules/.bin/webpack -p
	cd ui && node_modules/.bin/webpack -p
	@echo "--> Running JavaScript tests"
	cd ui && npm run test
	@echo ""

lint-go:  ## Run static code analysis for Go sources
	gometalinter --deadline=45s --vendor ./...

lint-js:  ## Run static code analysis for JavaScript sources
	cd ui && node_modules/.bin/eslint  --config .eslintrc --ext .jsx,.js {tests/js,app}
	@echo

help:  ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
