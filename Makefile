.PHONY: test tmp dist lint format

dist: lint test
	@rm -rf dist
	@mkdir -p dist
	make compiler

compiler:
	@go build -o dist/qdaad qdaad.go

test:
	@go tool executor run -d "vet" -c "go vet ./..."
	@go tool executor run -d "test" -c "go test -v ./..."

test-clean:
	@go tool executor run -d "clean test cache" -c "go clean -testcache"

format:
	@echo "Formatting..."
	@go mod tidy
	@go tool gofumpt -w .
	@go tool goimports -w .
	@go tool golines -m 120 -t 4 --ignore-generated --chain-split-dots -w .

lint:
	@go tool executor run -d "gofumpt" -c "go tool gofumpt -l -w ."
	@go tool executor run -d "staticcheck" -c "go tool staticcheck ./..."
	@go tool executor run -d "golangci-lint" -c "go tool golangci-lint run ./..."
	@go tool executor run -d "govulncheck" -c "go tool govulncheck ./..."

clean: test-clean
	@rm -Rf dist
	@rm -Rf tmp
	@pushd docs/manual > /dev/null && \
		rm -f *.aux *.log *.out *.toc *.lol *.lot *.lof *.bbl *.blg *.idx *.ilg *.ind \
			*.nlo *.nls *.nlg *.spl *.synctex.gz *.fdb_latexmk *.fls *.listing *.pdf; \
		popd > /dev/null

doc:
	@pushd docs/manual > /dev/null && \
		pdflatex -shell-escape -interaction=nonstopmode \
			-file-line-error manual.tex; \
		popd > /dev/null

go-upgrade-deps:
	@go get -u ./...
