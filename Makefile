.PHONY: dist build test doc decomp-check decomp-ao

VERSION=$$(git describe --tags --abbrev=0)

version:
	@echo $(VERSION)

build:
	@rm -rf build
	@mkdir -p build
	@echo "Building qdaad compiler..."
	@go build -o build/qdaad cmd/qdaad/qdaad.go
	@echo "Building qundaad decompiler..."
	@go build -o build/qundaad cmd/qundaad/qundaad.go

dist: lint test
	@rm -rf dist
	@mkdir -p dist
	@echo "Building qdaad compiler..."
	@make compiler
	@echo "Building qundaad decompiler..."
	@make decompiler


decompiler:
	@go build -o dist/qundaad cmd/qundaad/qundaad.go

compiler:
	@go build -o dist/qdaad cmd/qdaad/qdaad.go

test:
	@go tool executor run -d "vet" -c "go vet ./..."
	@go tool executor run -d "test" -c "go test -failfast ./...  -timeout 30s"

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

dead:
	@go tool deadcode ./...

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

# Decompiles every edition and holds them against one another. The lists of
# editions and of the pairs they have to match live in the script.
decomp-check:
	@script/decomp-all.sh

AO_PC := work/aventuras/La_Aventura_Original/PC
AO_OUT := work/decomp/ao/pc
decomp-ao:
	@rm -rf $(AO_OUT)
	@go run cmd/qundaad/qundaad.go decompile --input $(AO_PC)/PART1.DDB --output $(AO_OUT)/part1
	@go run cmd/qundaad/qundaad.go decompile --input $(AO_PC)/PART2.DDB --output $(AO_OUT)/part2
