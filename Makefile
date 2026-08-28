.PHONY: test tmp dist lint format decomp-check

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

# Every edition we can read today, as target:input. The target is where under
# tmp/decomp the source goes; the input is a database or an image holding some.
#
# A database is written into the target directory as it stands. An image is
# opened and every database in it decompiled into a directory of its own
# underneath, which is why entries naming a whole disk name only the platform.
#
# Neither the Amstrad and Spectrum disks nor any of the tapes carry a
# filesystem, so their databases are found by searching the bytes. A disk holds
# both parts and they are numbered in the order they lie; a tape holds one,
# which is why the tape entries name the part themselves.
#
# Left out on purpose, all for the same reason: the loader keeps the database in
# pieces, so only its header can be found where it lies.
#   La Aventura Original  commodore64
#   El Jabato             commodore64, amstradcpc, msx, and the Spectrum tapes
EDITIONS := \
	ao/pc/part1:work/aventuras/La_Aventura_Original/PC/PART1.DDB \
	ao/pc/part2:work/aventuras/La_Aventura_Original/PC/PART2.DDB \
	ao/amiga:work/aventuras/La_Aventura_Original/Amiga/ORIGINAL.ADF \
	ao/atarist:work/aventuras/La_Aventura_Original/AtariST/ORIGINAL.ST \
	ao/cpc:work/aventuras/La_Aventura_Original/AmstradCPC/ORIGINAL.DSK \
	ao/spectrum:work/aventuras/La_Aventura_Original/ZXSpectrum/La\ Aventura\ Original.dsk \
	ao/cdt/part1:work/aventuras/La_Aventura_Original/AmstradCPC/La\ aventura\ original\ \(Cara\ A\).cdt \
	ao/cdt/part2:work/aventuras/La_Aventura_Original/AmstradCPC/La\ aventura\ original\ \(Cara\ B\).cdt \
	ao/tzx/part1:work/aventuras/La_Aventura_Original/ZXSpectrum/La\ Aventura\ Original\ -\ Part\ 1.tzx \
	ao/tzx/part2:work/aventuras/La_Aventura_Original/ZXSpectrum/La\ Aventura\ Original\ -\ Part\ 2.tzx \
	jabato/pc-ega/part1:work/aventuras/El_Jabato/PC/Ega/PART1.DDB \
	jabato/pc-ega/part2:work/aventuras/El_Jabato/PC/Ega/PART2.DDB \
	jabato/pc-cga/part1:work/aventuras/El_Jabato/PC/CGA/PART1.DDB \
	jabato/pc-cga/part2:work/aventuras/El_Jabato/PC/CGA/PART2.DDB \
	jabato/amiga:work/aventuras/El_Jabato/Amiga/JABATO.ADF \
	jabato/atarist:work/aventuras/El_Jabato/AtariST/JABATO1.ST \
	jabato/spectrum:work/aventuras/El_Jabato/ZXSpectrum/Jabato.dsk \
	chichen/pc/part1:work/aventuras/Chichén_Itzá/PC/PART1.DDB \
	chichen/pc/part2:work/aventuras/Chichén_Itzá/PC/PART2.DDB \
	chichen/amiga:work/aventuras/Chichén_Itzá/Amiga/CHICHEN.ADF \
	chichen/atarist:work/aventuras/Chichén_Itzá/AtariST/chicitza.st \
	cozumel/pc/part1:work/aventuras/Cozumel/PC/PART1.DDB \
	cozumel/pc/part2:work/aventuras/Cozumel/PC/PART2.DDB \
	cozumel/amiga:work/aventuras/Cozumel/Amiga/COZUMEL.ADF \
	cozumel/atarist:work/aventuras/Cozumel/AtariST/Cozumel.st \
	cozumel/cpc:work/aventuras/Cozumel/AmstradCPC/Cozumel\ \(S\)\ \[Original\]\ \[Dinamic\].dsk \
	cozumel/cdt/part1:work/aventuras/Cozumel/AmstradCPC/COZUMEL1.CDT \
	cozumel/cdt/part2:work/aventuras/Cozumel/AmstradCPC/COZUMEL2.CDT \
	cozumel/spectrum:work/aventuras/Cozumel/ZXSpectrum/Cozumel.dsk \
	cozumel/tzx/part1:work/aventuras/Cozumel/ZXSpectrum/cozumel1_2e.tzx \
	cozumel/tzx/part2:work/aventuras/Cozumel/ZXSpectrum/cozumel2_2e.tzx \
	templos/pc/part1:work/aventuras/Los_templos_sagrados/PC/PART1.DDB \
	templos/pc/part2:work/aventuras/Los_templos_sagrados/PC/PART2.DDB \
	templos/amiga:work/aventuras/Los_templos_sagrados/Amiga/Templos\ Sagrados,\ Los\ \(1991\).adf \
	templos/atarist:work/aventuras/Los_templos_sagrados/AtariST/TEMPLOS1.ST \
	templos/atarist:work/aventuras/Los_templos_sagrados/AtariST/TEMPLOS2.ST

# Pairs that have to decompile to exactly the same source, bar game.sce, which
# names the file it was read from and so differs by definition.
#
# The tape against the disk is the strongest of these: two containers with
# nothing whatever in common — a stream of blocks written for the ear, against a
# table of sectors — arriving at the same text, byte for byte.
SAME_SOURCE := \
	ao/amiga/part1:ao/atarist/part1 \
	ao/amiga/part2:ao/atarist/part2 \
	ao/cpc/part1:ao/cdt/part1 \
	ao/cpc/part2:ao/cdt/part2 \
	ao/spectrum/part1:ao/tzx/part1 \
	ao/spectrum/part2:ao/tzx/part2 \
	jabato/amiga/part1:jabato/atarist/part1 \
	jabato/amiga/part2:jabato/atarist/part2 \
	jabato/pc-ega/part2:jabato/pc-cga/part2 \
	cozumel/cpc/part1:cozumel/cdt/part1 \
	cozumel/cpc/part2:cozumel/cdt/part2 \
	cozumel/spectrum/part1:cozumel/tzx/part1

# Pairs that have to agree on the text and the data, but not on everything.
# Machines that differ in endianness, in size and in every offset reaching one
# text is what makes this worth checking: break the endianness deduction, the
# token table or the pointer arithmetic and one of these stops matching.
SAME_TEXT := \
	ao/pc/part1:ao/amiga/part1 \
	ao/pc/part2:ao/amiga/part2 \
	ao/pc/part1:ao/atarist/part1 \
	ao/pc/part2:ao/atarist/part2 \
	ao/cpc/part1:ao/spectrum/part1 \
	jabato/pc-ega/part1:jabato/amiga/part1 \
	jabato/pc-ega/part2:jabato/amiga/part2 \
	jabato/pc-ega/part1:jabato/pc-cga/part1 \
	chichen/pc/part1:chichen/amiga/part1 \
	chichen/pc/part2:chichen/amiga/part2 \
	chichen/amiga/part1:chichen/atarist/part1 \
	chichen/amiga/part2:chichen/atarist/part2 \
	cozumel/pc/part1:cozumel/amiga/part1 \
	templos/pc/part1:templos/amiga/part1 \
	templos/pc/part2:templos/amiga/part2 \
	templos/amiga/part1:templos/atarist/part1 \
	templos/amiga/part2:templos/atarist/part2

# The sections a SAME_TEXT pair is held to. Left out:
#   processes.sce  editions genuinely differ here: the 68000 build of La
#                  Aventura Original draws an illustration the PC one does not
#   tokens.sce     the CGA build of El Jabato was compiled without a compression
#                  table at all, and its prose still has to come out the same as
#                  the EGA one, which says more about token expansion than
#                  comparing two identical tables ever could. Where the tables
#                  do matter they are compared whole, by SAME_SOURCE.
SHARED_SECTIONS := vocabulary.sce sysmess.sce messages.sce \
	object-text.sce location-text.sce connections.sce objects.sce

decomp-check:
	@rm -rf tmp/decomp
	@for edition in $(EDITIONS); do \
		target=$${edition%%:*}; \
		input=$${edition#*:}; \
		go run cmd/qundaad/qundaad.go decompile \
			--input "$$input" --output "tmp/decomp/$$target" || exit 1; \
	done
	@echo "Decompiled into tmp/decomp:"
	@find tmp/decomp -name game.sce | sort | sed 's|/game.sce||;s|^tmp/decomp/|  |'
	@echo "Checking the editions against each other..."
	@failed=0; \
	for pair in $(SAME_SOURCE); do \
		a=$${pair%%:*}; \
		b=$${pair#*:}; \
		diff -r -q -x game.sce "tmp/decomp/$$a" "tmp/decomp/$$b" > /dev/null \
			|| { echo "  FAIL $$a and $$b are not the same source"; failed=1; }; \
	done; \
	for pair in $(SAME_TEXT); do \
		a=$${pair%%:*}; \
		b=$${pair#*:}; \
		for section in $(SHARED_SECTIONS); do \
			diff -q "tmp/decomp/$$a/$$section" "tmp/decomp/$$b/$$section" > /dev/null \
				|| { echo "  FAIL $$a and $$b disagree on $$section"; failed=1; }; \
		done; \
	done; \
	test $$failed -eq 0 || exit 1
	@echo "  $(words $(SAME_SOURCE)) pairs give the same source whole"
	@echo "  $(words $(SAME_TEXT)) pairs give the same text and data"
