; --------------------------------------------------------------------------
; Decompiled by QUnDAAD from the contents at 0x3ed7 (from La Aventura Original - Part 2.tzx)
; Queru's DAAD decompiler v0.1.0b
; --------------------------------------------------------------------------
; DAAD version   : 1
; Machine        : ZX Spectrum (1)
; Language       : Spanish (1)
; Null word      : _
; Encoding       : UTF-8   (the original sources were ISO-8859-1)
; Header size    : 32 bytes   (deduced, not declared)
; Endianness     : little-endian   (deduced, not declared)
; Declared length: 16199 bytes
; Found at       : 0x3ed7, laid at address 0x8480   (deduced, not declared)
; --------------------------------------------------------------------------
; Objects        : 30
; Locations      : 109
; User messages  : 163
; System messages: 63
; Processes      : 21
; --------------------------------------------------------------------------

; --------------------------------------------------------------------------
; This tree is the +3_Extended one laid out for the Amstrad PCW: 90 columns by
; 32 rows of monochrome where the Spectrum has 42 by 24, which is more than
; twice as wide, so every window number is measured for it.
;
; The illustrations are the PC edition's. La Aventura Original never had a PCW
; release to take them from, and of the three that exist the PC's are the ones
; that convert: CGA mode 4 is four colours on a bitmap, so all that is lost is
; the colour — where the Spectrum's carry one ink per eight-by-eight cell and
; the Amstrad CPC's are vector drawings that version 3 cannot draw. They are
; 512 by 104 of one bit, which is thirteen character rows, and they were made
; by script/pcwpic.
;
; No cover on this one. The second disk of 1989 had none either — neither
; Cozumel's nor Los Templos Sagrados' carries a PARTE000.PIC — because it is
; the disk you put in to carry on, and the cover was on the first. Here it buys
; the room for all fifty-six illustrations. See ../README.md.
; --------------------------------------------------------------------------

; qdaad: charset=gfx/chr/udg.chr
; qdaad: images=gfx/scr

/CTL
_

#include tokens.sce
#include vocabulary.sce
#include sysmess.sce
#include messages.sce
#include object-text.sce
#include location-text.sce
#include connections.sce
#include objects.sce
#include processes.sce
#include v3-loop.sce

/END
