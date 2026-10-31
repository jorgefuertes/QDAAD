.PHONY: dist build test doc packager decomp-all decomp-ao recompile-all compile-all compare-drf compare-drb compare-daadmaker example-ao example-ao-48 example-ao-p3

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
	@echo "Building qpack packager..."
	@go build -o build/qpack ./cmd/qpack
	@echo "Building qrun interpreter..."
	@go build -o build/qrun ./cmd/qrun

dist: lint test
	@rm -rf dist
	@mkdir -p dist
	@echo "Building qdaad compiler..."
	@make compiler
	@echo "Building qundaad decompiler..."
	@make decompiler
	@echo "Building qpack packager..."
	@make packager


packager:
	@go build -o dist/qpack ./cmd/qpack

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

# Decompiles every edition we can read and holds them against one another. The
# lists of editions and of the pairs they have to match live in the script.
decomp-all:
	@script/decomp-all.sh

# Hands every decompilation back to the reference compiler.
# Needs decomp-all to have run first.
recompile-all:
	@script/recompile-all.sh

# Decompiles every edition we can read, converts it to version 3, and compiles
# it back into a database for the machine it came off — each edition to its own
# machine and to no other. Stops at the database: the illustrations and the
# character sets are separate blocks and belong to the packaging step.
# Needs nothing but the repository: both tools are ours.
compile-all:
	@script/compile-all.sh

# Reads every source with both front ends and compares what each understood.
# Needs fpc and iconv; says so and stops if either is missing.
compare-drf:
	@script/compare-drf.sh

# Links every source with both back ends and compares the databases byte for
# byte, over three targets so that the padding and the byte order are covered.
# Needs php and iconv; says so and stops if either is missing.
compare-drb:
	@script/compare-drb.sh

# Packs every source into a 48K Spectrum tape with both packagers and compares
# the tapes byte for byte.
# Needs fpc, php and iconv; says so and stops if any is missing.
compare-daadmaker:
	@script/compare-daadmaker.sh

AO_PC := work/aventuras/La_Aventura_Original/PC
AO_OUT := work/decomp/ao/pc
decomp-ao:
	@rm -rf $(AO_OUT)
	@go run cmd/qundaad/qundaad.go decompile --input $(AO_PC)/PART1.DDB --output $(AO_OUT)/part1
	@go run cmd/qundaad/qundaad.go decompile --input $(AO_PC)/PART2.DDB --output $(AO_OUT)/part2

# The ported La Aventura Original, built for each of its two machines. The
# sources are in the repository and need nothing from work/, so these run on a
# fresh checkout. See examples/AO/README.md for what is hand-corrected in them.
AO_EX := examples/AO/ZXSpectrum
example-ao-48:
	@go run cmd/qdaad/qdaad.go compile -i $(AO_EX)/48/game.sce -o tmp/example-ao-48 -f build -m zx -s 48k -v 3
	@go run cmd/qpack/qpack.go -f tap -o tmp/AO48 --name AO tmp/example-ao-48

example-ao-p3:
	@go run cmd/qdaad/qdaad.go compile -i $(AO_EX)/+3/game.sce -o tmp/example-ao-p3 -f build -m zx -s plus3 -v 3
	@go run cmd/qpack/qpack.go -f dsk -o tmp/AOP3 --name AO tmp/example-ao-p3

example-ao: example-ao-48 example-ao-p3
