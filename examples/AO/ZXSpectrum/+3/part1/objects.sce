; Objects: where each one starts, what it weighs, whether it is a
; container or can be worn, and the words that name it.
;
; Object 0 is the light source, by convention of the interpreter.
;
; The sixteen columns between the flags and the noun are the extra
; attributes, written highest bit first. A version 1 database has no
; table for them and they all come out unset.

/OBJ
;obj  starts.at    weight  cont  worn   15 14 13 12 11 10  9  8  7  6  5  4  3  2  1  0  noun      adjective
/0    _            12      _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  LAMPA     _
/1    _            11      _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  MONED     _
/2    _            12      _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  LAMPA     _
/3    _            10      _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  TORTI     _
/4    28           28      _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BOTEL     _
/5    _            28      _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BOTEL     _
/6    _            10      _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  LLAVE     _
/7    8            8       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BATER     _
/8    23           23      _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BOTAS     _
