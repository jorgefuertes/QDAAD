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
/0    _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  LAMPA     _
/1    CARRIED      1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  MONED     _
/2    _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  LAMPA     _
/3    CARRIED      1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  TORTI     _
/4    _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BOTEL     _
/5    _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BOTEL     _
/6    CARRIED      1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  LLAVE     _
/7    _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BATER     _
/8    4            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BARRO     _
/9    5            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  VARIT     _
/10   _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BARRO     _
/11   _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BOTEL     _
/12   54           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  HUESO     _
/13   68           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  ALMOH     _
/14   _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  HACHA     _
/15   4            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  CANTO     _
/16   _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  PEPIT     _
/17   11           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  DIAMA     _
/18   39           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  BARRA     _
/19   _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  JOYA      _
/20   64           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  ESPEC     _
/21   _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  ALFOM     _
/22   90           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  TRIDE     _
/23   _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  PERLA     _
/24   78           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  HUEVO     _
/25   _            1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  CADEN     _
/26   77           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  PIRAM     _
/27   72           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  ESMER     _
/28   70           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  JARRO     _
/29   19           1       _     _      _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  _  GANCH     _
