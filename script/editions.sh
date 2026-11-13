#!/usr/bin/env bash
#
# The corpus: every edition of the five adventures we can read.
#
# It lives on its own because more than one harness walks it, and a list that
# is copied is a list that drifts. Nothing here runs anything; it is sourced.

# Every edition we can read today, as target:input. The target is where under
# work/decomp the source goes; the input is a database or an image holding some.
#
# A database is written into the target directory as it stands. An image is
# opened and every database in it decompiled into a directory of its own
# underneath, which is why entries naming a whole disk name only the platform.
#
# Some of these disks and none of the tapes carry a filesystem, so their
# databases are found by searching the bytes. A disk holds both parts and they
# are numbered in the order they lie; a tape holds one, which is why the tape
# entries name the part themselves.
#
# The two Amstrad PCW disks of Los Templos Sagrados are labelled the other way
# round to what they hold: disk A carries PARTE002 and disk B carries PARTE001.
# The output is named after the database, not the disk, so the pairs further
# down look crossed and are not.
#
# The MSX disks are here as well, and they are the odd ones out: their database
# is nowhere on the disk in one piece. Every part of it loads into the same
# buffer and copies itself down into the RAM the game runs in, so what is read
# is the memory those copies build rather than any file — see msx.go.
#
# Left out because the loader keeps the database in pieces, so only its header
# can be found where it lies:
#   La Aventura Original  commodore64
#   El Jabato             commodore64, amstradcpc, msx, and the Spectrum tapes
#   Chichén Itzá          commodore64, and the Amstrad tapes
#   Cozumel               commodore64, msx
#   Los Templos Sagrados  commodore64
#
# The MSX tapes of all five are left out too: a .CAS is a stream of blocks like
# a Spectrum tape, and nothing reads them yet.
#
# And left out for a different reason: the two Spectrum disks of Los Templos
# Sagrados carry a PART1.DDB that does not read as a database. Its tapes do, so
# that edition is here — off the tape, which is also the only place its drawings
# can be had.
readonly EDITIONS=(
	"ao/pc/part1:work/aventuras/La_Aventura_Original/PC/PART1.DDB"
	"ao/pc/part2:work/aventuras/La_Aventura_Original/PC/PART2.DDB"
	"ao/amiga:work/aventuras/La_Aventura_Original/Amiga/ORIGINAL.ADF"
	"ao/atarist:work/aventuras/La_Aventura_Original/AtariST/ORIGINAL.ST"
	"ao/cpc/dsk:work/aventuras/La_Aventura_Original/AmstradCPC/ORIGINAL.DSK"
	"ao/spectrum/dsk:work/aventuras/La_Aventura_Original/ZXSpectrum/La Aventura Original.dsk"
	"ao/cpc/cdt/part1:work/aventuras/La_Aventura_Original/AmstradCPC/La aventura original (Cara A).cdt"
	"ao/cpc/cdt/part2:work/aventuras/La_Aventura_Original/AmstradCPC/La aventura original (Cara B).cdt"
	"ao/spectrum/tzx/part1:work/aventuras/La_Aventura_Original/ZXSpectrum/La Aventura Original - Part 1.tzx"
	"ao/spectrum/tzx/part2:work/aventuras/La_Aventura_Original/ZXSpectrum/La Aventura Original - Part 2.tzx"
	"jabato/pc-ega/part1:work/aventuras/El_Jabato/PC/Ega/PART1.DDB"
	"jabato/pc-ega/part2:work/aventuras/El_Jabato/PC/Ega/PART2.DDB"
	"jabato/pc-cga/part1:work/aventuras/El_Jabato/PC/CGA/PART1.DDB"
	"jabato/pc-cga/part2:work/aventuras/El_Jabato/PC/CGA/PART2.DDB"
	"jabato/amiga:work/aventuras/El_Jabato/Amiga/JABATO.ADF"
	"jabato/atarist:work/aventuras/El_Jabato/AtariST/JABATO1.ST"
	"jabato/spectrum/dsk:work/aventuras/El_Jabato/ZXSpectrum/Jabato.dsk"
	"chichen/pc/part1:work/aventuras/Chichén_Itzá/PC/PART1.DDB"
	"chichen/pc/part2:work/aventuras/Chichén_Itzá/PC/PART2.DDB"
	"chichen/amiga:work/aventuras/Chichén_Itzá/Amiga/CHICHEN.ADF"
	"chichen/atarist:work/aventuras/Chichén_Itzá/AtariST/chicitza.st"
	"chichen/cpc/dsk:work/aventuras/Chichén_Itzá/AmstradCPC/CHICHEN.DSK"
	"chichen/msx:work/aventuras/Chichén_Itzá/MSX/chichen_itza.dsk"
	"chichen/spectrum/tzx/part1:work/aventuras/Chichén_Itzá/ZXSpectrum/Chichen Itza - Side 1.tzx"
	"chichen/spectrum/tzx/part2:work/aventuras/Chichén_Itzá/ZXSpectrum/Chichen Itza - Side 2.tzx"
	"cozumel/pc/part1:work/aventuras/Cozumel/PC/PART1.DDB"
	"cozumel/pc/part2:work/aventuras/Cozumel/PC/PART2.DDB"
	"cozumel/amiga:work/aventuras/Cozumel/Amiga/COZUMEL.ADF"
	"cozumel/atarist:work/aventuras/Cozumel/AtariST/Cozumel.st"
	"cozumel/cpc/dsk:work/aventuras/Cozumel/AmstradCPC/Cozumel (S) [Original] [Dinamic].dsk"
	"cozumel/cpc/cdt/part1:work/aventuras/Cozumel/AmstradCPC/COZUMEL1.CDT"
	"cozumel/cpc/cdt/part2:work/aventuras/Cozumel/AmstradCPC/COZUMEL2.CDT"
	"cozumel/spectrum/dsk:work/aventuras/Cozumel/ZXSpectrum/Cozumel.dsk"
	"cozumel/spectrum/tzx/part1:work/aventuras/Cozumel/ZXSpectrum/cozumel1_2e.tzx"
	"cozumel/spectrum/tzx/part2:work/aventuras/Cozumel/ZXSpectrum/cozumel2_2e.tzx"
	"cozumel/pcw/a:work/aventuras/Cozumel/AmstradPCW/Cozumel_A.dsk"
	"cozumel/pcw/a-fixed:work/aventuras/Cozumel/AmstradPCW/Cozumel_A_corregida.dsk"
	"cozumel/pcw/b:work/aventuras/Cozumel/AmstradPCW/Cozumel_B.dsk"
	"templos/pc/part1:work/aventuras/Los_templos_sagrados/PC/PART1.DDB"
	"templos/pc/part2:work/aventuras/Los_templos_sagrados/PC/PART2.DDB"
	"templos/amiga:work/aventuras/Los_templos_sagrados/Amiga/Templos Sagrados, Los (1991).adf"
	"templos/atarist:work/aventuras/Los_templos_sagrados/AtariST/TEMPLOS1.ST"
	"templos/atarist:work/aventuras/Los_templos_sagrados/AtariST/TEMPLOS2.ST"
	"templos/cpc/dsk:work/aventuras/Los_templos_sagrados/AmstradCPC/TEMPLOS.DSK"
	"templos/cpc/cdt/part1:work/aventuras/Los_templos_sagrados/AmstradCPC/templos1.cdt"
	"templos/cpc/cdt/part2:work/aventuras/Los_templos_sagrados/AmstradCPC/templos2.cdt"
	"templos/spectrum/tzx/part1:work/aventuras/Los_templos_sagrados/ZXSpectrum/Los Templos Sagrados - Side 1.tzx"
	"templos/spectrum/tzx/part2:work/aventuras/Los_templos_sagrados/ZXSpectrum/Los Templos Sagrados - Side 2.tzx"
	"templos/pcw/a:work/aventuras/Los_templos_sagrados/AmstradPCW/templosa.DSK"
	"templos/pcw/b:work/aventuras/Los_templos_sagrados/AmstradPCW/templosb.dsk"
)
