.PHONY: build
build: cmd/kubectl-cost

.PHONY: build
build:
	cd cmd && go build -o kubectl-cost

.PHONY: install
install: build
	chmod +x ./cmd/kubectl-cost
	cp ./cmd/kubectl-cost ~/go/bin/kubectl-cost
