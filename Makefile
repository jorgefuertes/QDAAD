.PHONY: version dist build test qtest doc compiler decompiler packager decomp-all decomp-ao recompile-all compile-all compare-drf compare-drb compare-daadmaker compare-zx0 example-ao example-ao-48 example-ao-p3

# The release this is, which is the tag git last saw and not a number written
# anywhere in the source. A tree with no tags at all still builds, and says so:
# "dev" is the honest answer and the one internal/version defaults to.
#
# It is overridable so that the release workflow can hand over the tag it was
# triggered by, which is more reliable than asking git in a shallow checkout.
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)

# How the version gets inside the binaries. See internal/version.
LDFLAGS := -X github.com/jorgefuertes/QDAAD/internal/version.Version=$(VERSION)

# The four tools a release is made of. qrun used to be missing from dist and
# not from build, so the downloads had three of the four this project is.
TOOLS := qdaad qundaad qpack qrun

# Where a release goes, as a person reading the download page says it rather
# than as Go spells it: OSX and not darwin. The architectures keep Go's own
# names — amd64 and arm64 are what every other project's downloads say, and
# inventing "intel" here would only make them harder to match up.
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

version:
	@echo $(VERSION)

build:
	@rm -rf build
	@mkdir -p build
	@for tool in $(TOOLS); do \
		echo "Building $$tool..."; \
		go build -ldflags "$(LDFLAGS)" -o "build/$$tool" "./cmd/$$tool" || exit 1; \
	done

# A release: the four tools for the six machines, each set in one archive with
# the version in its name, and a file of checksums beside them.
#
# It goes through lint and test first, which is the point of it being this
# target and not a script: a release is the one build nobody gets to redo
# quietly.
#
# Windows gets a .zip and the rest a .tar.gz. A tarball on Windows wants a tool
# that is not in the box, and an archive whose only job is to open should open.
dist: lint test
	@rm -rf dist
	@mkdir -p dist
	@set -e; \
	for platform in $(PLATFORMS); do \
		goos=$${platform%/*}; goarch=$${platform#*/}; \
		case $$goos in \
			darwin)  pretty=OSX ;; \
			linux)   pretty=Linux ;; \
			windows) pretty=Windows ;; \
			*)       pretty=$$goos ;; \
		esac; \
		stem="QDAAD_$(VERSION)_$${pretty}_$${goarch}"; \
		echo "Building $$stem..."; \
		mkdir -p "dist/$$stem"; \
		suffix=""; \
		[ "$$goos" = windows ] && suffix=".exe"; \
		for tool in $(TOOLS); do \
			GOOS=$$goos GOARCH=$$goarch CGO_ENABLED=0 \
				go build -trimpath -ldflags "$(LDFLAGS)" \
				-o "dist/$$stem/$$tool$$suffix" "./cmd/$$tool"; \
		done; \
		cp README.md README_ES.md "dist/$$stem/"; \
		if [ "$$goos" = windows ]; then \
			(cd dist && zip -qr "$$stem.zip" "$$stem"); \
		else \
			tar -czf "dist/$$stem.tar.gz" -C dist "$$stem"; \
		fi; \
		rm -rf "dist/$$stem"; \
	done
	@# One file of checksums for the lot, in the format sha256sum -c reads, so
	@# that somebody who downloaded one archive can check it without reading
	@# instructions. macOS spells the tool differently from Linux and this runs
	@# on both.
	@cd dist && \
		if command -v sha256sum >/dev/null 2>&1; then \
			sha256sum QDAAD_*.tar.gz QDAAD_*.zip > "QDAAD_$(VERSION)_checksums.txt"; \
		else \
			shasum -a 256 QDAAD_*.tar.gz QDAAD_*.zip > "QDAAD_$(VERSION)_checksums.txt"; \
		fi
	@echo
	@ls -1 dist

packager:
	@go build -ldflags "$(LDFLAGS)" -o dist/qpack ./cmd/qpack

decompiler:
	@go build -ldflags "$(LDFLAGS)" -o dist/qundaad ./cmd/qundaad

compiler:
	@go build -ldflags "$(LDFLAGS)" -o dist/qdaad ./cmd/qdaad

# The timeout is what internal/pack needs. ZX0 packs a picture by searching
# every offset at every position — a quarter of a second a screen, which is what
# the format costs done properly and what the reference tool costs too — and one
# test packages a disk that overruns a 128-name directory, so it packages 134 of
# them. The unit tests overrun a directory of twelve instead; see
# zxplus3cache_test.go for why that is the same question.
#
# It is a variable because the number is a measurement of one machine and CI
# runs on another: the same package takes a few seconds here and several times
# that on a hired runner, and a limit tuned to a laptop turns that into a
# failure that says "panic: test timed out" and nothing about the code.
TEST_TIMEOUT ?= 300s

test:
	@go tool executor run -d "vet" -c "go vet ./..."
	@go tool executor run -d "test" -c "go test -failfast ./...  -timeout $(TEST_TIMEOUT)"

test-clean:
	@go tool executor run -d "clean test cache" -c "go clean -testcache"

format:
	@echo "Formatting..."
	@go mod tidy
	@go tool golangci-lint fmt ./...

lint:
	@go tool executor run -d "format" -c "go tool golangci-lint fmt ./..."
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

# "go get -u ./..." cannot work here: golangci-lint's graph carries three
# renamed modules and two pseudo-versions above their latest tag, and -u only
# ever picks release tags. Direct requires go by name; tools go by @latest,
# which lets MVS honour what each tool pins.
go-upgrade-deps:
	@go get -u github.com/charmbracelet/lipgloss github.com/common-nighthawk/go-figure \
		github.com/muesli/termenv github.com/stretchr/testify github.com/urfave/cli/v3
	@go get github.com/golangci/golangci-lint/v2@latest
	@go get github.com/jorgefuertes/executor@latest
	@go get golang.org/x/tools@latest
	@go get golang.org/x/vuln@latest
	@go mod tidy

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

# Compresses every screen the examples carry with Einar Saukas' own ZX0 coder
# and requires our decoder to read what it wrote. Byte equality is not asked
# for: his coder runs a shortest path and ours is greedy.
# Needs a C compiler; says so and stops if it is missing.
compare-zx0:
	@script/compare-zx0.sh

AO_PC := work/aventuras/La_Aventura_Original/PC
AO_OUT := work/decomp/ao/pc
decomp-ao:
	@rm -rf $(AO_OUT)
	@go run cmd/qundaad/qundaad.go decompile --input $(AO_PC)/PART1.DDB --output $(AO_OUT)/part1
	@go run cmd/qundaad/qundaad.go decompile --input $(AO_PC)/PART2.DDB --output $(AO_OUT)/part2

# The ported La Aventura Original, built for each of its three trees and both
# of its parts. The sources are in the repository and need nothing from work/,
# so these run on a fresh checkout. See examples/AO/README.md for what is
# hand-corrected in them, and the README beside each tree for the commands
# without make in the way.
AO_EX := examples/AO/ZXSpectrum
example-ao-48:
	@for part in 1 2; do \
		go run cmd/qdaad/qdaad.go compile -i $(AO_EX)/48/part$$part/game.sce \
			-o tmp/ao-48-part$$part -f build -m zx -s 48k -v 3 --part $$part; \
		go run cmd/qpack/qpack.go -f tap -o tmp/AO$$part --name AO$$part tmp/ao-48-part$$part; \
		go run cmd/qpack/qpack.go -f tzx -o tmp/AO$$part --name AO$$part tmp/ao-48-part$$part; \
	done

example-ao-p3:
	@for part in 1 2; do \
		go run cmd/qdaad/qdaad.go compile -i "$(AO_EX)/+3/part$$part/game.sce" \
			-o tmp/ao-p3-part$$part -f build -m zx -s plus3 -v 3 --part $$part; \
		go run cmd/qpack/qpack.go -f dsk -o tmp/AOP3-$$part --name AO$$part tmp/ao-p3-part$$part; \
	done

# The same disks with the texts of the 16-bit editions in them.
example-ao-p3x:
	@for part in 1 2; do \
		go run cmd/qdaad/qdaad.go compile -i "$(AO_EX)/+3_Extended/part$$part/game.sce" \
			-o tmp/ao-p3x-part$$part -f build -m zx -s plus3 -v 3 --part $$part; \
		go run cmd/qpack/qpack.go -f dsk -o tmp/AOP3X-$$part --name AO$$part tmp/ao-p3x-part$$part; \
	done

example-ao: example-ao-48 example-ao-p3 example-ao-p3x

# Plays the walkthroughs in examples/AO/test, which is the first part of La
# Aventura Original won and the second part entered, in the test language qrun
# reads. The databases are built here rather than taken from anywhere: a
# prebuilt one left lying about goes on passing against a week-old build.
#
# The suite plays these too — internal/qtest, TestTheExampleScriptsPlay — so
# this target is for watching it happen, and for running one script on its own.
qtest:
	@mkdir -p tmp
	@for part in 1 2; do \
		go run cmd/qdaad/qdaad.go compile -i "$(AO_EX)/+3/part$$part/game.sce" \
			-o tmp/qtest-part$$part -f ddb -m pc -s vga -v 3 --part $$part; \
		go run ./cmd/qrun -i tmp/qtest-part$$part.ddb \
			-T examples/AO/test/part$$part.qtest; \
		go run cmd/qdaad/qdaad.go compile -i "$(AO_EX)/+3_Extended/part$$part/game.sce" \
			-o tmp/qtest-x-part$$part -f ddb -m pc -s vga -v 3 --part $$part; \
		go run ./cmd/qrun -i tmp/qtest-x-part$$part.ddb \
			-T examples/AO/test/part$$part-extended.qtest; \
	done
	@# The xmessages, which are the one thing no edition of 1989 exercises: the
	@# texts live outside the database and qrun picks the .XMB up beside it.
	@go run cmd/qdaad/qdaad.go compile -i script/sources/xmessages.sce \
		-o tmp/qtest-xmessages -f ddb -m pc -s vga256 -v 3
	@go run ./cmd/qrun -i tmp/qtest-xmessages.ddb -T script/sources/xmessages.qtest
