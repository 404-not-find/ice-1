VERSION := $(shell git describe --tags | sed -e 's/^v//g' | awk -F "-" '{print $$1}')
ITERATION := $(shell git describe --tags --long | awk -F "-" '{print $$2}')
GO_VERSION=$(shell gobuild -v)
GO := $(or $(GOROOT),/usr/lib/go)/bin/go
PROCS := $(shell nproc)
cores:
	@echo "cores: $(PROCS)"
test:
	go test -v
bench:
	go test -bench .
bench-record:
	$(GO) test -bench . > "benchmarks/stun-go-$(GO_VERSION).txt"
fuzz-prepare-msg:
	go-fuzz-build -func FuzzMessage -o stun-msg-fuzz.zip github.com/gortc/stun
fuzz-prepare-typ:
	go-fuzz-build -func FuzzType -o stun-typ-fuzz.zip github.com/gortc/stun
fuzz-prepare-setters:
	go-fuzz-build -func FuzzSetters -o stun-setters-fuzz.zip github.com/gortc/stun
fuzz-msg:
	go-fuzz -bin=./stun-msg-fuzz.zip -workdir=examples/stun-msg
fuzz-typ:
	go-fuzz -bin=./stun-typ-fuzz.zip -workdir=examples/stun-typ
fuzz-setters:
	go-fuzz -bin=./stun-setters-fuzz.zip -workdir=examples/stun-setters
fuzz-test:
	go test -tags gofuzz -run TestFuzz -v .
fuzz-reset-setters:
	rm -f -v -r stun-setters-fuzz.zip examples/stun-setters
lint:
	@echo "linting on $(PROCS) cores"
	@gometalinter \
		--enable-all \
		-e "(parse|Equal).+(gocyclo)" \
		-e "_test.go.+(gocyclo|errcheck|dupl)" \
		-e "cmd\/" \
		-e "integration-test\/.+(gocyclo|errcheck|dupl)" \
		--enable="lll" --line-length=100 \
		--disable="gochecknoglobals" \
		--disable="gochecknoinits" \
		--disable="maligned" \
		--deadline=300s \
		-j $(PROCS)
	@echo "ok"
escape:
	@echo "Not escapes, except autogenerated:"
	@go build -gcflags '-m -l' 2>&1 \
	| grep -v "<autogenerated>" \
	| grep escapes
format:
	goimports -w .
install:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install --update
	go get -u github.com/dvyukov/go-fuzz/go-fuzz-build
	go get github.com/dvyukov/go-fuzz/go-fuzz
test-integration:
	@cd integration-test && bash ./test.sh
prepush: test lint test-integration
check-api:
	api -c api/stun1.txt github.com/gortc/ice
