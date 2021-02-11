.PHONY: build
build:
	cd cmd/kubectl-cost && go build

.PHONY: install
install: build
	chmod +x ./cmd/kubectl-cost/kubectl-cost
	cp ./cmd/kubectl-cost/kubectl-cost ~/go/bin/kubectl-cost
