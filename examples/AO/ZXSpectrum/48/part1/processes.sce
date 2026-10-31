; The process tables: the program of the adventure. Each entry names a
; verb and a noun, and the condacts below run when the player's sentence
; matches them. The null word matches anything.
;
; Skip distances are written as numbers of entries, not as labels.

; TABLE 0 is the game loop, in v3-loop.sce

; TABLE 1 is the game loop, in v3-loop.sce

; TABLE 2 is the game loop, in v3-loop.sce

; ----------------------------------------------------------------------
; TABLE 3 - SUB-PROCESS. No PROCESS condact in this database reaches it: it is
; either dead code or entered some other way.
; ----------------------------------------------------------------------
/PRO 3

> _        _        AT 0
                    NOTZERO 255
                    GOTO 2

> _        _        AT 0
                    PRINTAT 11 21
                    LT 29 128
                    PRINTAT 11 39

> _        _        AT 0
                    MES 18
                    ANYKEY
                    GOTO 1
                    CLS
                    RESTART

> _        _        ATGT 12
                    ATLT 16
                    MES 95

> _        _        AT 1
                    WINDOW 5
                    WINAT 0 5
                    WINSIZE 25 36

> _        _        AT 1
                    WINDOW 0
                    PICTURE 44
                    DISPLAY 0
                    WINDOW 5
                    PRINTAT 5 6
                    MES 77
                    PRINTAT 7 10
                    MES 78
                    PRINTAT 11 9
                    MES 86
                    PRINTAT 13 11
                    MES 85
                    PRINTAT 9 11
                    MES 88
                    PRINTAT 10 8
                    MES 87
                    PRINTAT 12 8
                    MES 89
                    PRINTAT 14 8
                    MES 92
                    PRINTAT 14 16
                    MES 93
                    PRINTAT 17 5
                    MES 94
                    ANYKEY
                    WINDOW 7
                    CLS
                    WINDOW 5

> _        _        AT 1
                    WINSIZE 25 40
                    CLS
                    MES 99
                    ANYKEY
                    WINDOW 1

> _        _        AT 1
                    CLS
                    SET 255
                    GOTO 2

> _        _        AT 2
                    ABILITY 5 255
                    LET 53 64
                    WINAT 14 0

> _        _        AT 2
                    MOUSE 2 0   ; the old MOUSE takes 1, so the last 1 is padding and not in the database
                    GOTO 3
                    CLS
                    RESTART

> _        _        TIME 47 1

> _        _        AT 17
                    LT 103 255
                    MES 71

> _        _        AT 20
                    ZERO 101
                    LET 101 1

> _        _        NOTAT 20
                    EQ 101 1
                    LET 101 2

> _        _        AT 20
                    EQ 101 2
                    MES 62

> _        _        AT 20
                    EQ 101 3
                    MES 67

> _        _        AT 12
                    ZERO 102
                    LET 102 1

> _        _        NOTAT 12
                    EQ 102 1
                    LET 102 2

> _        _        AT 12
                    EQ 102 2
                    LET 102 3

> _        _        NOTAT 12
                    EQ 102 3
                    LET 102 4

> _        _        AT 26
                    WINDOW 0
                    CLS
                    WINDOW 1
                    SYSMESS 23
                    EQ 101 4
                    LT 14 3
                    MESSAGE 134

> S        _        AT 26
                    EQ 101 4
                    EQ 14 3
                    MESSAGE 134

> _        _        AT 26
                    EQ 14 0
                    WINDOW 0
                    PICTURE 46
                    DISPLAY 0
                    WINDOW 1
                    MES 53
                    MES 55
                    MES 54

> _        _        AT 26
                    EQ 14 1
                    WINDOW 0
                    PICTURE 46
                    DISPLAY 0
                    WINDOW 1
                    MES 53
                    MES 56
                    MES 54

> _        _        AT 26
                    EQ 14 2
                    WINDOW 0
                    PICTURE 47
                    DISPLAY 0
                    WINDOW 1
                    MES 53
                    MES 54

> _        _        AT 26
                    EQ 14 3
                    WINDOW 0
                    PICTURE 48
                    DISPLAY 0
                    WINDOW 1
                    MES 57

> _        _        AT 26
                    EQ 101 4
                    EQ 14 3
                    SET 101
                    MESSAGE 67
                    MES 98
                    MES 69
                    PLACE 3 18

> _        _        EQ 101 4
                    PLUS 100 1
                    EQ 100 2
                    CLEAR 100

> _        _        EQ 101 4
                    ZERO 100
                    NOTAT 26
                    NEWLINE
                    MES 98
                    MES 68

> _        _        EQ 101 4
                    EQ 100 1
                    NOTAT 26
                    MES 67

> _        _        AT 12
                    LT 102 255
                    MES 60

> _        _        AT 12
                    EQ 102 255
                    SYSMESS 48

> _        _        AT 30
                    WINDOW 7
                    CLS
                    PRINTAT 8 5

> _        _        AT 30
                    MESSAGE 47
                    PRINTAT 12 5

> _        _        AT 30
                    CARRIED 2
                    CARRIED 5
                    MESSAGE 50

> _        _        AT 30
                    CARRIED 2
                    CARRIED 4
                    MESSAGE 49

> _        _        AT 30
                    CARRIED 0
                    CARRIED 5
                    MESSAGE 48

> _        _        AT 30
                    CARRIED 0
                    CARRIED 4
                    MESSAGE 51

> _        _        AT 30
                    PRINTAT 19 0
                    END

> _        _        NEWLINE
                    ZERO 0
                    ABSENT 0
                    LISTOBJ
                    ZERO 40
                    WINDOW 0
                    PICTURE 27
                    DISPLAY 0
                    WINDOW 1

> _        _        PRESENT 0
                    LISTOBJ
                    ZERO 40
                    WINDOW 0
                    PICTURE 27
                    DISPLAY 0
                    WINDOW 1

> _        _        AT 29
                    CLS
                    PROCESS 21

; ----------------------------------------------------------------------
; TABLE 4 - SUB-PROCESS. No PROCESS condact in this database reaches it: it is
; either dead code or entered some other way.
; ----------------------------------------------------------------------
/PRO 4

> _        _        EQ 5 1
                    MESSAGE 5
                    LET 5 4
                    DONE

> _        _        AT 12
                    EQ 102 4
                    PAUSE 100
                    MESSAGE 3
                    PAUSE 150
                    MESSAGE 65
                    SET 102
                    LET 5 4
                    DONE

; ----------------------------------------------------------------------
; TABLE 5 - SUB-PROCESS. No PROCESS condact in this database reaches it: it is
; either dead code or entered some other way.
; ----------------------------------------------------------------------
/PRO 5

> AD       _        MESSAGE 139
                    DONE

> S        _        AT 28
                    PROCESS 11

> S        _        AT 9
                    PROCESS 11

> S        _        AT 17
                    PROCESS 12

> S        _        AT 23
                    PROCESS 11

> E        _        AT 9
                    PROCESS 11

> E        _        AT 17
                    PROCESS 12

> O        _        AT 17
                    PROCESS 12

> ARRIB    _        AT 11
                    GOTO 10
                    CLEAR 0
                    CLS
                    RESTART

> ABAJO    _        AT 10
                    GOTO 11
                    SET 0
                    CLS
                    RESTART

> ABAJO    _        AT 12
                    PROCESS 10

> ABAJO    _        AT 28
                    PROCESS 11

> ABAJO    _        AT 9
                    PROCESS 11

> ENTRA    _        AT 26
                    ZERO 14
                    MES 53
                    MES 55
                    SYSMESS 48
                    DONE

> ENTRA    _        AT 26
                    EQ 14 1
                    MES 53
                    MES 56
                    SYSMESS 48
                    DONE

> ENTRA    _        AT 26
                    EQ 14 2
                    MES 53
                    SYSMESS 48
                    DONE

> ENTRA    _        AT 26
                    EQ 1 5
                    NOTZERO 11
                    GOTO 30
                    CLS
                    RESTART

> ENTRA    _        AT 26
                    MESSAGE 73
                    DONE

> N        _        AT 12
                    PROCESS 10

> N        _        AT 17
                    PROCESS 12

> ESCUC    _        AT 19
                    MESSAGE 130
                    DONE

> ESCUC    _        MESSAGE 129
                    DONE

> AYUDA    _        PROCESS 7
                    DONE

> MATAR    _        MESSAGE 28
                    NOTDONE

> GRABA    RAM      RAMSAVE
                    CLS
                    RESTART

> GRABA    _        SAVE 19

> CARGA    _        NOTEQ 34 99
                    LOAD 0   ; the old LOAD takes 0, so the last 1 is padding and not in the database

> CARGA    RAM      RAMLOAD 248

> CARGA    _        NOTEQ 34 99
                    DONE

> GRABA    _        WINDOW 7
                    CLS
                    CLS
                    RESTART

> ABRIR    ENANO    AT 12
                    LT 102 255
                    SET 102
                    MESSAGE 128
                    PAUSE 100
                    MESSAGE 25
                    LET 5 4
                    DONE

> ABRIR    PUERT    AT 26
                    EQ 14 2
                    LET 14 3
                    CLS
                    RESTART

> ABRIR    PUERT    AT 3
                    OK

> ABRIR    BARRO    AT 26
                    CARRIED 6
                    EQ 14 0
                    LET 14 1
                    CLS
                    RESTART

> CERRA    PUERT    AT 26
                    EQ 14 3
                    LET 14 2
                    CLS
                    RESTART

> CERRA    BARRO    AT 26
                    EQ 14 1
                    LET 14 0
                    CLS
                    RESTART

> ASUST    ENANO    AT 12
                    LT 102 255
                    PROCESS 15
                    DONE

> AGARR    TODO     DOALL 255

> AGARR    MONED    AT 11
                    PRESENT 1
                    PRESENT 0
                    GET 1
                    SET 12
                    DONE

> AGARR    MONED    AT 11
                    MES 1
                    PAUSE 100
                    MESSAGE 12
                    DONE

> AGARR    LLAVE    AT 17
                    EQ 103 1
                    MESSAGE 127
                    DONE

> AGARR    ENANO    AT 12
                    LT 102 255
                    PROCESS 15
                    DONE

> AGARR    BARRO    AT 26
                    EQ 14 1
                    LET 14 2
                    MESSAGE 58
                    DONE

> AGARR    _        AUTOG
                    DONE

> DEJA     TODO     AT 18
                    ZERO 1
                    MESSAGE 100
                    NEWLINE
                    DONE

> DEJA     TODO     AT 18
                    GOTO 10
                    DROPALL
                    GOTO 18
                    MESSAGE 52
                    DONE

> DEJA     TODO     DOALL 254

> DEJA     _        AT 18
                    WHATO
                    LT 51 255
                    EQ 54 254
                    COPYFF 55 38
                    AUTOD
                    PAUSE 50
                    MESSAGE 52
                    GOTO 18
                    DONE

> DEJA     _        AT 17
                    WHATO
                    LT 51 255
                    EQ 54 254
                    MESSAGE 24
                    PUTO 16
                    DONE

> DEJA     _        AUTOD
                    DONE

> SALTA    _        AT 9
                    PROCESS 11

> SALTA    _        AT 12
                    PROCESS 10

> SALTA    _        AT 17
                    PROCESS 12

> SALTA    _        AT 28
                    PROCESS 11

> SALTA    _        OK

> ENCEN    LAMPA    CARRIED 2
                    NOTZERO 11
                    AT 11
                    SWAP 2 0
                    CLEAR 0
                    CLS
                    RESTART

> ENCEN    LAMPA    CARRIED 2
                    NOTZERO 11
                    SWAP 2 0
                    OK

> ENCEN    LAMPA    CARRIED 2
                    ZERO 11
                    MESSAGE 136
                    DONE

> APAGA    LAMPA    CARRIED 0
                    AT 11
                    SWAP 0 2
                    SET 0
                    CLS
                    RESTART

> APAGA    LAMPA    CARRIED 0
                    SWAP 0 2
                    OK

> ESPER    _        MESSAGE 36
                    DONE

> LEE      CARTE    PROCESS 8
                    DONE

> EX       _        PROCESS 8
                    DONE

> VACIA    BOTEL    CARRIED 5
                    SWAP 5 4
                    OK

> LLENA    BOTEL    AT 3
                    CARRIED 4
                    SWAP 4 5
                    OK

> LLENA    BOTEL    AT 19
                    CARRIED 4
                    SWAP 4 5
                    OK

> LLENA    BOTEL    AT 24
                    CARRIED 4
                    SWAP 4 5
                    OK

> LLENA    BOTEL    AT 25
                    CARRIED 4
                    SWAP 4 5
                    OK

> COMER    TORTI    CARRIED 3
                    DESTROY 3
                    MESSAGE 4
                    LET 5 4
                    DONE

> BEBER    AGUA     CARRIED 5
                    SWAP 5 4
                    OK

> AGITA    ARBOL    AT 17
                    LT 103 2
                    LET 51 6
                    MESSAGE 24
                    PLACE 6 16
                    SET 103
                    DONE

> AGITA    ARBOL    AT 17
                    MESSAGE 138
                    DONE

> INSER    BATER    CARRIED 7
                    CARRIED 2
                    DESTROY 7
                    SET 11
                    MESSAGE 2
                    DONE

> QUITA    BATER    CARRIED 2
                    NOTZERO 11
                    GT 1 4
                    SYSMESS 27
                    DONE

> QUITA    BATER    CARRIED 2
                    NOTZERO 11
                    CLEAR 11
                    CREATE 7
                    GET 7
                    OK

> QUITA    BATER    CARRIED 0
                    NOTZERO 11
                    GT 1 4
                    SYSMESS 27
                    DONE

> QUITA    BATER    CARRIED 0
                    AT 11
                    SWAP 0 2
                    CREATE 7
                    GET 7
                    CLEAR 11
                    CLS
                    RESTART

> QUITA    BATER    CARRIED 0
                    NOTZERO 11
                    SWAP 0 2
                    CREATE 7
                    GET 7
                    CLEAR 11
                    OK

> QUITA    BARRO    AT 26
                    EQ 14 1
                    LET 14 2
                    MESSAGE 58
                    ANYKEY
                    CLS
                    RESTART

> D        _        CLS
                    RESTART

> SENTA    SILLA    AT 18
                    MESSAGE 35
                    PAUSE 100
                    MESSAGE 36
                    DONE

> ROMPE    BOTEL    CARRIED 4
                    DESTROY 4
                    LET 5 4
                    OK

> ROMPE    BOTEL    CARRIED 5
                    DESTROY 5
                    LET 5 4
                    OK

> SALTA    _        AT 12
                    PROCESS 10

> SALTA    _        AT 9
                    PROCESS 11

> SALTA    _        AT 28
                    PROCESS 11

> SALTA    _        AT 23
                    PROCESS 11

> CABRO    _        NOTZERO 254
                    CLEAR 254
                    MESSAGE 46
                    PAUSE 200
                    PROCESS 9

> CABRO    _        PROCESS 9

> CAPUL    _        NOTZERO 254
                    CLEAR 254
                    MESSAGE 46
                    PAUSE 200
                    PROCESS 9

> CAPUL    _        DROPALL
                    SET 254
                    MESSAGE 28
                    PAUSE 200
                    GOTO 3
                    CLS
                    RESTART

> ABAND    _        QUIT
                    PROCESS 21
                    END

> ABAND    _        OK

> SALUD    _        AT 12
                    LT 102 255
                    MESSAGE 125
                    DONE

> DECIR    ENANO    AT 12
                    LT 102 255
                    PROCESS 14
                    DONE

> SALUD    _        AT 20
                    GT 101 2
                    LT 101 5
                    MESSAGE 45
                    DONE

> SALUD    _        EQ 101 4
                    MESSAGE 45
                    DONE

> DECIR    ELFIT    AT 20
                    GT 101 2
                    LT 101 5
                    PROCESS 13
                    DONE

> DECIR    _        EQ 101 4
                    PROCESS 13
                    DONE

> DECIR    _        AT 20
                    EQ 101 3
                    PROCESS 13
                    DONE

> DECIR    _        AT 12
                    PROCESS 14
                    DONE

> ASUST    _        EQ 101 4
                    LET 101 3
                    MESSAGE 70
                    DONE

> ASUST    _        OK

> INSER    BOTAS    CARRIED 8
                    MESSAGE 135
                    DONE

> DAR      BOTAS    AT 12
                    CARRIED 8
                    LT 102 255
                    DESTROY 8
                    SET 102
                    CREATE 2
                    MESSAGE 61
                    DONE

> SALID    _        PROCESS 17
                    DONE

> _        I        NOTZERO 1
                    SYSMESS 9
                    LISTAT 254
                    DONE

> _        I        MESSAGE 100

> _        LLAVE    AT 26
                    EQ 33 255
                    ZERO 14
                    MESSAGE 120
                    DONE

> _        ELFIT    AT 20
                    GT 101 2
                    LT 101 5
                    PROCESS 13
                    DONE

> _        ELFIT    EQ 101 4
                    PROCESS 13
                    DONE

> _        ELFIT    AT 20
                    EQ 101 3
                    PROCESS 13
                    DONE

> _        _        LT 33 14
                    MOVE 38
                    CLS
                    RESTART

> _        _        LT 33 14
                    SYSMESS 7
                    EQ 33 13
                    SYSMESS 34

> _        _        LT 33 9
                    SYSMESS 34

> _        _        LT 33 14
                    COPYFF 33 200
                    PLUS 200 100
                    MES @200
                    SYSMESS 11
                    PROCESS 17

; TABLE 6 is the game loop, in v3-loop.sce

; ----------------------------------------------------------------------
; TABLE 7 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 7

> AYUDA    _        NOTZERO 5
                    MESSAGE 96
                    DONE

> AYUDA    _        AT 11
                    ABSENT 0
                    MESSAGE 8
                    DONE

> AYUDA    _        ATGT 12
                    ATLT 16
                    MESSAGE 9
                    DONE

> AYUDA    _        AT 26
                    EQ 14 0
                    MESSAGE 10
                    DONE

> AYUDA    _        AT 26
                    EQ 14 3
                    MESSAGE 119
                    DONE

> AYUDA    _        AT 23
                    MESSAGE 36
                    DONE

> AYUDA    _        AT 20
                    NOTCARR 0
                    MESSAGE 8
                    DONE

> _        _        CHANCE 10
                    MESSAGE 36

> _        _        MESSAGE 11

; ----------------------------------------------------------------------
; TABLE 8 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 8

> EX       ESTRU    AT 22
                    MESSAGE 132
                    DONE

> EX       CAMIN    AT 21
                    MESSAGE 131
                    DONE

> EX       LAMPA    CARRIED 2
                    MESSAGE 16
                    DONE

> EX       LAMPA    CARRIED 0
                    MESSAGE 15
                    DONE

> EX       MONED    CARRIED 1
                    MESSAGE 17
                    DONE

> EX       TORTI    CARRIED 3
                    MESSAGE 19
                    DONE

> EX       BOTAS    CARRIED 8
                    MESSAGE 133
                    DONE

> EX       BOTEL    CARRIED 4
                    MESSAGE 20
                    DONE

> EX       BOTEL    CARRIED 5
                    MESSAGE 124
                    DONE

> EX       LLAVE    CARRIED 6
                    MESSAGE 21
                    DONE

> EX       BATER    CARRIED 7
                    MESSAGE 22
                    DONE

> EX       CABAÑ    AT 3
                    MESSAGE 23
                    DONE

> EX       CABAÑ    AT 26
                    MESSAGE 123
                    DONE

> EX       BOCA     AT 10
                    MESSAGE 122
                    DONE

> EX       AGUA     AT 11
                    PRESENT 0
                    MES 6
                    PAUSE 150
                    MESSAGE 12
                    DONE

> EX       AGUA     AT 11
                    PRESENT 0
                    ZERO 12
                    MES 6
                    PAUSE 150
                    MESSAGE 7
                    CREATE 1
                    SET 12
                    DONE

> EX       AGUA     AT 3
                    MESSAGE 121
                    DONE

> EX       AGUA     AT 24
                    MESSAGE 121
                    DONE

> EX       AGUA     AT 25
                    MESSAGE 121
                    DONE

> EX       ARBOL    AT 3
                    MESSAGE 26
                    DONE

> EX       ARBOL    AT 6
                    MESSAGE 26
                    DONE

> EX       ARBOL    AT 8
                    MESSAGE 26
                    DONE

> EX       ARBOL    AT 17
                    LT 103 2
                    MESSAGE 72
                    LET 103 1
                    DONE

> EX       ARBOL    AT 16
                    MESSAGE 126
                    DONE

> EX       ARBOL    ATGT 12
                    ATLT 24
                    MESSAGE 26
                    DONE

> EX       VOLCA    ATGT 5
                    ATLT 10
                    MESSAGE 27
                    DONE

> EX       LAVA     AT 6
                    MESSAGE 28
                    DONE

> EX       PRECI    AT 9
                    MESSAGE 28
                    DONE

> EX       ENANO    AT 12
                    LT 102 255
                    MESSAGE 79
                    DONE

> EX       HABIT    AT 10
                    CLS
                    RESTART

> EX       VENTA    AT 10
                    MESSAGE 30
                    DONE

> EX       ESCAL    AT 10
                    MESSAGE 31
                    DONE

> EX       ESCAL    AT 11
                    PRESENT 0
                    MESSAGE 32
                    DONE

> EX       CHARC    AT 11
                    PRESENT 0
                    ZERO 12
                    MES 6
                    PAUSE 100
                    MESSAGE 7
                    CREATE 1
                    DONE

> EX       CHARC    AT 11
                    PRESENT 0
                    MES 6
                    PAUSE 100
                    MESSAGE 12
                    DONE

> EX       ESPIR    AT 23
                    MESSAGE 33
                    DONE

> EX       MESA     AT 18
                    MESSAGE 34
                    DONE

> EX       SILLA    AT 18
                    MESSAGE 34
                    DONE

> EX       GRIET    AT 25
                    MESSAGE 37
                    DONE

> EX       PUERT    AT 26
                    LT 14 2
                    MESSAGE 38
                    DONE

> EX       BARRO    AT 26
                    LT 14 2
                    MESSAGE 38
                    DONE

> EX       CASCA    AT 28
                    MESSAGE 39
                    DONE

> _        CARTE    AT 18
                    MESSAGE 13
                    DONE

> EX       ELFIT    AT 20
                    CARRIED 0
                    EQ 101 2
                    MESSAGE 63
                    LET 101 3
                    DONE

> EX       ELFIT    AT 20
                    EQ 101 2
                    MESSAGE 64
                    DONE

> EX       BRILL    AT 17
                    LT 103 2
                    MESSAGE 72
                    LET 103 1
                    DONE

> EX       _        MESSAGE 14
                    DONE

; ----------------------------------------------------------------------
; TABLE 9 - SUB-PROCESS, called from table 5, 10, 11, 12.
; ----------------------------------------------------------------------
/PRO 9

> _        _        PAUSE 150
                    PAUSE 150
                    GOTO 29
                    CLS
                    RESTART

; ----------------------------------------------------------------------
; TABLE 10 - SUB-PROCESS, called from table 5, 15.
; ----------------------------------------------------------------------
/PRO 10

> _        _        MESSAGE 29
                    PAUSE 150
                    MESSAGE 43
                    PAUSE 150
                    MESSAGE 40
                    PROCESS 9

; ----------------------------------------------------------------------
; TABLE 11 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 11

> _        _        MESSAGE 41
                    PAUSE 150
                    MESSAGE 42
                    PAUSE 150
                    MESSAGE 43
                    PROCESS 9

; ----------------------------------------------------------------------
; TABLE 12 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 12

> _        _        MESSAGE 41
                    PAUSE 150
                    MESSAGE 44
                    PAUSE 150
                    MESSAGE 43
                    PROCESS 9

; ----------------------------------------------------------------------
; TABLE 13 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 13

> _        _        PARSE 1
                    NOTEQ 101 3
                    MESSAGE 97
                    DONE

> _        _        EQ 101 3
                    ZERO 104
                    MESSAGE 67
                    SET 104
                    DONE

> AYUDA    _        MESSAGE 76
                    DONE

> CAPUL    _        LET 33 96

> CABRO    _        LET 101 3
                    NOTAT 20
                    MESSAGE 70
                    DONE

> CABRO    _        MESSAGE 59
                    DONE

> _        ESTAS    MESSAGE 67
                    DONE

> _        CABAÑ    MESSAGE 75
                    DONE

> _        SIGUE    LET 101 4
                    OK

> _        LLAMA    MESSAGE 74
                    DONE

> _        HOLA     MESSAGE 137
                    DONE

> _        _        MESSAGE 66
                    DONE

; ----------------------------------------------------------------------
; TABLE 14 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 14

> _        _        PARSE 1
                    MESSAGE 97
                    DONE

> AYUDA    _        MESSAGE 66
                    DONE

> CAPUL    _        PROCESS 15
                    DONE

> CABRO    _        PROCESS 15
                    DONE

> _        ESTAS    MESSAGE 81
                    DONE

> _        LLAMA    MESSAGE 80
                    DONE

> _        HOLA     MESSAGE 137
                    DONE

> _        _        MESSAGE 59
                    DONE

; ----------------------------------------------------------------------
; TABLE 15 - SUB-PROCESS, called from table 5, 14.
; ----------------------------------------------------------------------
/PRO 15

> _        _        SET 102
                    MESSAGE 82
                    PAUSE 100
                    MES 83
                    PAUSE 200
                    MESSAGE 84
                    PAUSE 200
                    PROCESS 10
                    OK

; ----------------------------------------------------------------------
; TABLE 16 - SUB-PROCESS. No PROCESS condact in this database reaches it: it is
; either dead code or entered some other way.
; ----------------------------------------------------------------------
/PRO 16

; ----------------------------------------------------------------------
; TABLE 17 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 17

> _        _        CLEAR 201
                    LET 200 101
                    PROCESS 20
                    PROCESS 18
                    GT 201 1
                    MES 117

> _        _        EQ 201 1
                    MES 118

> _        _        MES @200
                    MESSAGE 115

; ----------------------------------------------------------------------
; TABLE 18 - SUB-PROCESS, called from table 17.
; ----------------------------------------------------------------------
/PRO 18

> _        _        COPYFF 38 202
                    MOVE 202
                    PROCESS 19

> _        _        PLUS 33 1
                    LT 33 13
                    REDO

; ----------------------------------------------------------------------
; TABLE 19 - SUB-PROCESS, called from table 18, 20.
; ----------------------------------------------------------------------
/PRO 19

> _        _        EQ 201 1
                    MES 114

> _        _        GT 201 1
                    MES 116

> _        _        NOTZERO 201
                    MES @200

> _        _        COPYFF 33 200
                    PLUS 200 100
                    PLUS 201 1

; ----------------------------------------------------------------------
; TABLE 20 - SUB-PROCESS, called from table 17.
; ----------------------------------------------------------------------
/PRO 20

> _        _        LET 33 13
                    COPYFF 38 202
                    MOVE 202
                    PROCESS 19

> _        _        LET 33 2

; ----------------------------------------------------------------------
; TABLE 21 - SUB-PROCESS, called from table 3, 5.
; ----------------------------------------------------------------------
/PRO 21

> _        _        SYSMESS 17
                    DPRINT 31

> _        _        EQ 32 0
                    EQ 31 1
                    SYSMESS 18
                    SYSMESS 20
                    SKIP 1

> _        _        SYSMESS 18
                    SYSMESS 19
                    SYSMESS 20

