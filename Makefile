.PHONY: test tmp dist lint format decomp-check

dist: lint test
	@rm -rf dist
	@mkdir -p dist
	make compiler

compiler:
	@go build -o dist/qdaad qdaad.go

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

# Every edition of La Aventura Original we can read today, as target:input.
#
# The input is either a database or a disk image. A database is written into the
# target directory as it stands; an image is walked and every database inside it
# gets a directory of its own underneath, which is why the two 68000 entries
# name only the platform.
#
# Left out on purpose, until the readers exist:
#   commodore64  the databases live inside the programs, not on the disk
#   amstradcpc   .DSK and .CDT, not read yet
#   zxspectrum   .DSK and .TZX, not read yet
AO_EDITIONS := \
	pc/part1:work/AO/PC/PART1.DDB \
	pc/part2:work/AO/PC/PART2.DDB \
	amiga:work/AO/Amiga/ORIGINAL.ADF \
	atarist:work/AO/AtariST/ORIGINAL.ST

# The sections every machine has to agree on. Three binaries that differ in
# endianness, in size and in every offset have to come out as one source, which
# is what makes this worth checking: break the endianness deduction, the token
# table or the pointer arithmetic and one of these stops matching.
#
# Left out, both for reasons and not for convenience:
#   game.sce       names the file it was read from, so it differs by definition
#   processes.sce  the 68000 editions genuinely differ from the PC one: an extra
#                  illustration in location 29 and one line less of window
AO_SHARED := tokens.sce vocabulary.sce sysmess.sce messages.sce \
	object-text.sce location-text.sce connections.sce objects.sce

decomp-check:
	@rm -rf tmp/decomp/ao
	@for edition in $(AO_EDITIONS); do \
		target=$${edition%%:*}; \
		input=$${edition#*:}; \
		go run cmd/qundaad/qundaad.go decompile \
			--input "$$input" --output "tmp/decomp/ao/$$target" || exit 1; \
	done
	@echo "Decompiled into tmp/decomp/ao:"
	@find tmp/decomp/ao -name game.sce | sort | sed 's|/game.sce||;s|^|  |'
	@echo "Checking the editions against each other..."
	@failed=0; \
	for part in part1 part2; do \
		for other in amiga atarist; do \
			for section in $(AO_SHARED); do \
				diff -q "tmp/decomp/ao/pc/$$part/$$section" \
					"tmp/decomp/ao/$$other/$$part/$$section" > /dev/null \
					|| { echo "  FAIL $$part/$$section: pc and $$other disagree"; failed=1; }; \
			done; \
		done; \
		diff -r -q -x game.sce "tmp/decomp/ao/amiga/$$part" \
			"tmp/decomp/ao/atarist/$$part" > /dev/null \
			|| { echo "  FAIL $$part: amiga and atarist are not the same source"; failed=1; }; \
	done; \
	test $$failed -eq 0 || exit 1
	@echo "  pc, amiga and atarist yield the same source, bar the process table"
	@echo "  amiga and atarist match whole: the Amiga shipped the Atari database"
