.PHONY: install

install:
	GOBIN="$$(go env GOPATH)/bin" go install ./cmd/modrel
	@echo "Installed modrel to $$(go env GOPATH)/bin/modrel"
