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
                    WINDOW 5
                    WINAT 0 5

> _        _        AT 0
                    SET 115
                    LET 8 200
                    MOUSE 2 0   ; the old MOUSE takes 1, so the last 1 is padding and not in the database
                    LET 53 64
                    ABILITY 7 255
                    NOTZERO 255
                    COPYFF 254 33
                    PROCESS 7
                    CLS
                    RESTART

> _        _        AT 0
                    WINDOW 0
                    SET 255
                    PICTURE 1
                    DISPLAY 0

> _        _        AT 0
                    WINDOW 5
                    PRINTAT 5 6
                    MES 0
                    PRINTAT 11 9
                    MES 107
                    PRINTAT 13 11
                    MES 61
                    PRINTAT 7 10
                    MES 106
                    PRINTAT 12 8
                    MES 2
                    PRINTAT 10 8
                    MES 6
                    PRINTAT 14 8
                    MES 77
                    PRINTAT 15 9
                    PRINTAT 17 5
                    MES 83
                    PRINTAT 9 11
                    MES 140
                    GOTO 1
                    INPUT 6 0
                    WINDOW 6
                    MODE 2
                    WINAT 21 0
                    DONE

> _        _        AT 92
                    PROCESS 25

> _        _        AT 91
                    PROCESS 19
                    LT 150 4
                    GT 150 1
                    MES 136
                    PRINT 150
                    MESSAGE 131

> _        _        AT 91
                    EQ 150 1
                    MESSAGE 137

> _        _        AT 3
                    ZERO 152
                    NOTZERO 151
                    WINDOW 0
                    PICTURE 107
                    DISPLAY 0
                    WINDOW 1
                    MESSAGE 146
                    SET 152
                    ANYKEY
                    CLS
                    RESTART

> _        _        PROCESS 22

> _        _        AT 89
                    CLEAR 110

> _        _        AT 3
                    ZERO 152
                    WINDOW 0
                    PICTURE 107
                    DISPLAY 0
                    WINDOW 1
                    MES 147

> _        _        AT 3
                    NOTZERO 152
                    WINDOW 0
                    PICTURE 106
                    DISPLAY 0
                    WINDOW 1
                    MES 148

> _        _        ATGT 2
                    NEWLINE
                    NOTAT 91
                    ZERO 0
                    LISTOBJ
                    ZERO 40
                    WINDOW 0
                    PICTURE 95
                    DISPLAY 0
                    WINDOW 1

> _        _        NOTAT 91
                    ATGT 2
                    NOTZERO 0
                    PRESENT 0
                    LISTOBJ
                    ZERO 40
                    WINDOW 0
                    PICTURE 95
                    DISPLAY 0
                    WINDOW 1

> _        _        NOTAT 91
                    ATGT 2
                    NOTZERO 0
                    ABSENT 0
                    WINDOW 0
                    CLS
                    WINDOW 1

; ----------------------------------------------------------------------
; TABLE 4 - SUB-PROCESS. No PROCESS condact in this database reaches it: it is
; either dead code or entered some other way.
; ----------------------------------------------------------------------
/PRO 4

> _        _        AT 2
                    WINDOW 7
                    CLS
                    SET 100
                    TIME 30 1
                    INPUT 0 4
                    WINDOW 1
                    WINAT 14 0

> _        _        AT 2
                    WINDOW 0
                    CLS
                    PICTURE 106
                    DISPLAY 0
                    WINDOW 1
                    MESSAGE 150
                    ANYKEY
                    GOTO 3
                    CLS
                    RESTART

> _        _        AT 1
                    PAUSE 100
                    CLS
                    MES 1
                    DONE

> _        _        NOTZERO 0
                    ABSENT 0
                    ZERO 9
                    SET 9

> _        _        PRESENT 0
                    CLEAR 9

> _        _        ZERO 0
                    CLEAR 9

> _        _        ISAT 0 252
                    NOTZERO 8
                    PLUS 8 1

> _        _        EQ 8 1
                    LET 8 2
                    PRESENT 0
                    SWAP 2 0
                    CLS
                    RESTART

> _        _        LT 118 2
                    SET 7

> _        _        EQ 7 250
                    PROCESS 12
                    MESSAGE 89
                    PROCESS 21
                    CLEAR 0
                    GOTO 92
                    ANYKEY
                    CLS
                    RESTART

> _        _        PLUS 104 1
                    NOTAT 38
                    LT 104 9
                    CLEAR 104

> _        _        EQ 104 9
                    MESSAGE 8

> _        _        EQ 104 16
                    MESSAGE 9
                    CLEAR 0
                    GOTO 92
                    ANYKEY
                    CLS
                    RESTART

> _        _        PRESENT 0
                    EQ 8 60
                    MESSAGE 74
                    MINUS 8 1

> _        _        PRESENT 0
                    NOTZERO 8
                    LT 8 50
                    MESSAGE 75

> _        _        LT 9 250
                    NOTZERO 9
                    MESSAGE 76
                    ANYKEY
                    CLEAR 0
                    GOTO 92
                    CLS
                    RESTART

> _        _        AT 54
                    ZERO 107
                    LET 107 4

> _        _        AT 54
                    LT 107 255
                    GT 107 1
                    MINUS 107 1

> _        _        PROCESS 23

> _        _        LT 118 2
                    LT 120 5
                    CHANCE 9
                    ATGT 43
                    ATLT 68
                    NOTAT 47
                    NOTAT 52
                    NOTAT 53
                    NOTAT 54
                    NOTAT 55
                    NOTAT 56
                    NOTAT 58
                    NOTAT 59
                    NOTAT 91
                    MESSAGE 85
                    PLUS 118 1
                    ANYKEY
                    MESSAGE 22
                    ANYKEY
                    NEWTEXT
                    EQ 118 1
                    PROCESS 12
                    PROCESS 21
                    MESSAGE 109
                    MESSAGE 87
                    PLACE 14 255
                    ANYKEY
                    CLS
                    RESTART

> _        _        NOTAT 3
                    NOTAT 38
                    NOTAT 91
                    NOTAT 89
                    NOTAT 54
                    NOTAT 19
                    PROCESS 20

> _        _        AT 91
                    ZERO 151
                    CLEAR 121
                    DOALL 91
                    WHATO
                    GT 51 15
                    PLUS 121 1
                    EQ 121 14
                    PAUSE 100
                    CLS
                    WINDOW 5
                    PRINTAT 10 0
                    MESSAGE 48
                    ANYKEY
                    CLS
                    PRINTAT 8 0
                    MESSAGE 93
                    ANYKEY
                    CLS
                    LET 151 11
                    CLS
                    RESTART

> _        _        NOTZERO 151
                    MINUS 151 1
                    ZERO 151
                    CLS
                    MESSAGE 51
                    ANYKEY
                    CLS
                    CLEAR 0
                    GOTO 92
                    CLS
                    RESTART

; ----------------------------------------------------------------------
; TABLE 5 - SUB-PROCESS. No PROCESS condact in this database reaches it: it is
; either dead code or entered some other way.
; ----------------------------------------------------------------------
/PRO 5

> _        _        ZERO 100
                    PROCESS 7
                    DONE

> AD       _        MESSAGE 162
                    DONE

> SALID    _        PROCESS 14
                    DONE

> _        FIGUR    AT 41
                    MESSAGE 138
                    DONE

> ABRIR    LAMPA    LET 33 107
                    LET 34 55

> BAÑAR    JARRO    CARRIED 28
                    DESTROY 28
                    MESSAGE 32
                    DONE

> DEJA     PAJAR    LET 33 18
                    LET 34 56

> S        _        AT 75
                    LET 33 104

> S        _        AT 77
                    CLEAR 0
                    GOTO 73
                    CLS
                    RESTART

> E        _        AT 72
                    EQ 1 1
                    CARRIED 27
                    CLEAR 0
                    GOTO 73
                    CLS
                    RESTART

> E        _        AT 72
                    ZERO 1
                    CLEAR 0
                    GOTO 73
                    CLS
                    RESTART

> E        _        AT 72
                    MESSAGE 33
                    DONE

> E        _        AT 4
                    CLEAR 0
                    GOTO 3
                    CLS
                    RESTART

> O        _        AT 9
                    PRESENT 16
                    NOTCARR 16
                    DESTROY 16
                    CLEAR 230
                    CLEAR 102

> O        _        AT 9
                    ZERO 103
                    CLEAR 160
                    GOTO 10
                    CLS
                    RESTART

> O        _        AT 10
                    ZERO 160
                    PROCESS 13

> E        _        AT 10
                    NOTZERO 160
                    PROCESS 13

> E        _        AT 12
                    ZERO 103
                    SET 160
                    GOTO 10
                    CLS
                    RESTART

> SALGO    _        AT 3
                    NOTZERO 152
                    WINDOW 7
                    CLS
                    WINDOW 5
                    PRINTAT 8 0
                    MESSAGE 145
                    END

> SALGO    _        AT 3
                    MESSAGE 33
                    DONE

> SALGO    _        AT 77
                    CLEAR 0
                    GOTO 73
                    CLS
                    RESTART

> O        _        AT 73
                    EQ 1 1
                    CARRIED 27
                    SET 0
                    GOTO 72
                    CLS
                    RESTART

> O        _        AT 73
                    ZERO 1
                    SET 0
                    GOTO 72
                    CLS
                    RESTART

> O        _        AT 73
                    MESSAGE 33
                    DONE

> O        _        AT 3
                    SET 0
                    GOTO 4
                    CLS
                    RESTART

> NE       _        AT 74
                    LET 33 104
                    LET 34 105

> SO       _        AT 38
                    GT 104 8
                    GOTO 53
                    CLS
                    RESTART

> SO       _        AT 38
                    MESSAGE 8
                    LET 104 9
                    DONE

> SO       _        AT 75
                    LET 33 104
                    LET 34 105

> ARRIB    _        AT 9
                    PRESENT 16
                    NOTCARR 16
                    DESTROY 16
                    CLEAR 230
                    CLEAR 102
                    NOTDONE

> ABAJO    _        AT 9
                    PRESENT 16
                    NOTCARR 16
                    DESTROY 16
                    CLEAR 230
                    CLEAR 102
                    NOTDONE

> ARRIB    BROTE    AT 64
                    GT 109 1
                    GOTO 78
                    CLS
                    RESTART

> ARRIB    BROTE    AT 64
                    MESSAGE 30
                    CLEAR 109
                    DONE

> ENTRA    _        AT 73
                    SET 0
                    GOTO 77
                    CLS
                    RESTART

> ENTRA    _        AT 78
                    EQ 114 255
                    GOTO 90
                    CLS
                    RESTART

> ENTRA    _        AT 78
                    MESSAGE 33
                    DONE

> ENTRA    _        AT 88
                    NOTZERO 113
                    GOTO 89
                    CLS
                    RESTART

> ENTRA    _        AT 88
                    MESSAGE 33
                    DONE

> N        _        AT 38
                    GT 104 8
                    GOTO 39
                    CLS
                    RESTART

> N        _        AT 38
                    MESSAGE 8
                    LET 104 9
                    DONE

> N        _        AT 54
                    EQ 107 255
                    GOTO 55
                    CLS
                    RESTART

> N        _        AT 54
                    MESSAGE 33
                    DONE

> N        _        AT 73
                    SET 0
                    GOTO 77
                    CLS
                    RESTART

> N        _        AT 74
                    LET 33 104
                    LET 34 105

> I        _        PROCESS 10

> AYUDA    _        PROCESS 9
                    DONE

> DORMI    _        MESSAGE 101
                    DONE

> EMPUJ    _        AT 3
                    MESSAGE 33
                    DONE

> DAR      LAMPA    PRESENT 0
                    PROCESS 24
                    ABSENT 0
                    ANYKEY
                    CLS
                    RESTART

> DAR      LAMPA    PRESENT 0
                    DONE

> DAR      _        PROCESS 24
                    DONE

> ABRIR    BARRO    NOTAT 38
                    CARRIED 10
                    SWAP 10 8
                    MESSAGE 11
                    CLEAR 122
                    DONE

> ABRIR    BARRO    LT 104 9
                    CARRIED 10
                    SWAP 10 8
                    MESSAGE 11
                    CLEAR 122
                    MESSAGE 10
                    SET 104
                    CREATE 19
                    DONE

> ABRIR    BARRO    CARRIED 10
                    MESSAGE 9
                    ANYKEY
                    CLEAR 0
                    GOTO 92
                    CLS
                    RESTART

> ABRIR    BARRO    AT 88
                    CARRIED 6
                    MESSAGE 57
                    SET 113
                    DONE

> ABRIR    BARRO    AT 88
                    MESSAGE 56
                    DONE

> ABRIR    BARRO    CARRIED 8
                    MESSAGE 105
                    DONE

> ABRIR    ALMEJ    AT 49
                    CARRIED 22
                    ZERO 106
                    MESSAGE 15
                    PLACE 23 51
                    SET 106
                    DONE

> ABRIR    ALMEJ    AT 49
                    CARRIED 22
                    MESSAGE 16
                    DONE

> ABRIR    ALMEJ    AT 49
                    MESSAGE 17
                    DONE

> ABRIR    PUERT    AT 78
                    EQ 114 1
                    SET 114
                    MESSAGE 60
                    DONE

> ABRIR    _        AT 3
                    MESSAGE 33
                    DONE

> CERRA    BARRO    CARRIED 10
                    MESSAGE 105
                    DONE

> AGITA    VARIT    ATGT 9
                    ATLT 12
                    CARRIED 9
                    PLUS 103 1
                    GOTO 11
                    EQ 103 2
                    CLEAR 103
                    GOTO 10

> AGITA    VARIT    CARRIED 9
                    ATGT 9
                    ATLT 12
                    CLS
                    RESTART

> AGITA    VARIT    CARRIED 9
                    PRESENT 10
                    PROCESS 11

> AGITA    VARIT    CARRIED 9
                    AT 7
                    ZERO 122
                    PROCESS 11

> AGITA    VARIT    CARRIED 9
                    OK

> AGITA    VARIT    LET 33 20

> SACAR    BATER    NOUN2 MAQUI
                    AT 25
                    LET 33 20

> BUSCA    _        AT 9
                    ZERO 102
                    MESSAGE 143
                    DONE

> BUSCA    PEPIT    AT 9
                    ISAT 16 9
                    SYSMESS 49
                    DONE

> AGARR    _        AT 38
                    LT 104 9
                    LET 104 9
                    MESSAGE 8
                    DONE

> AGARR    TODO     DOALL 255

> AGARR    ALMOH    ISAT 13 255
                    ISAT 28 255
                    NOTAT 68
                    GET 13
                    LET 51 28
                    MESSAGE 32
                    DESTROY 28
                    DONE

> AGARR    PAJAR    AT 7
                    NOTCARR 9
                    CARRIED 8
                    ZERO 122
                    SWAP 8 10
                    SET 122
                    MESSAGE 94
                    DONE

> AGARR    PAJAR    AT 7
                    MESSAGE 11
                    DONE

> AGARR    BROTE    AT 64
                    MESSAGE 29
                    DONE

> AGARR    PEPIT    AT 9
                    ZERO 102
                    MESSAGE 143
                    DONE

> AGARR    HACHA    ABSENT 14
                    AT 78
                    NOTDONE

> AGARR    _        AUTOG
                    DONE

> COJER    _        MESSAGE 114
                    DONE

> COGEL    _        AT 70
                    MESSAGE 113
                    DONE

> DEJA     TODO     DOALL 254

> DEJA     JARRO    ISNOTAT 13 255
                    CARRIED 28
                    NOTAT 68
                    WHATO
                    MESSAGE 32
                    DESTROY 28
                    DONE

> DEJA     OSO      AT 89
                    EQ 111 2
                    CARRIED 6
                    SET 111
                    MESSAGE 43
                    DONE

> DEJA     OSO      AT 89
                    CARRIED 6
                    ZERO 111
                    MESSAGE 54
                    ANYKEY
                    CLEAR 0
                    GOTO 92
                    CLS
                    RESTART

> DEJA     OSO      EQ 111 255
                    ATGT 73
                    ATLT 76
                    ZERO 110
                    MES 40
                    MESSAGE 41
                    SET 110
                    CREATE 25
                    LET 111 1
                    DONE

> DEJA     OSO      EQ 111 255
                    MESSAGE 40
                    CREATE 25
                    LET 111 1
                    DONE

> DEJA     _        AUTOD
                    DONE

> ENCEN    LAMPA    EQ 8 2
                    MESSAGE 74
                    NOTDONE

> ENCEN    LAMPA    CARRIED 2
                    NOTZERO 115
                    SWAP 2 0
                    CLS
                    RESTART

> APAGA    LAMPA    CARRIED 0
                    SWAP 0 2
                    CLS
                    RESTART

> APAGA    LAMPA    CARRIED 2
                    OK

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

> PALPA    PARED    PROCESS 8
                    DONE

> PALPA    _        MESSAGE 55
                    DONE

> ALUMB    _        PROCESS 8
                    DONE

> ESCUC    _        AT 40
                    MESSAGE 12
                    DONE

> ECHAR    _        CARRIED 5
                    EQ 34 94
                    LET 34 53

> ECHAR    BOTEL    CARRIED 5
                    NOTAT 64
                    SWAP 4 5
                    OK

> ECHAR    BOTEL    CARRIED 11
                    NOTAT 64
                    NOTAT 78
                    SWAP 11 4
                    OK

> ECHAR    BOTEL    AT 64
                    NOTCARR 5
                    NOTCARR 11
                    SYSMESS 8
                    DONE

> ECHAR    BOTEL    AT 64
                    LET 33 89
                    LET 34 90

> ECHAR    BOTEL    AT 78
                    LET 33 45
                    LET 34 93

> LLENA    BOTEL    AT 90
                    CARRIED 4
                    SWAP 4 5
                    OK

> LLENA    BOTEL    CARRIED 4
                    AT 41
                    SWAP 4 5
                    OK

> LLENA    BOTEL    CARRIED 4
                    AT 56
                    SWAP 4 5
                    OK

> LLENA    BOTEL    CARRIED 4
                    AT 65
                    SWAP 4 11
                    OK

> LEER     _        LET 34 77
                    PROCESS 8
                    DONE

> COMER    HUESO    CARRIED 12
                    MESSAGE 3
                    ANYKEY
                    CLEAR 0
                    GOTO 92
                    CLS
                    RESTART

> COMER    TORTI    CARRIED 3
                    DESTROY 3
                    OK

> BEBER    _        CARRIED 5
                    SWAP 4 5
                    OK

> BEBER    _        AT 90
                    OK

> BEBER    _        AT 41
                    OK

> BEBER    _        AT 56
                    OK

> ACEIT    PUERT    AT 78
                    CARRIED 11
                    ZERO 114
                    SWAP 11 4
                    MESSAGE 59
                    LET 114 1
                    DONE

> ACEIT    _        NOTDONE

> APART    NIEBL    AT 9
                    MESSAGE 4
                    DONE

> SOPLA    _        AT 9
                    ZERO 102
                    SET 102

> SOPLA    _        ZERO 230
                    SET 230
                    PLACE 16 9

> SOPLA    _        OK

> ASUST    _        AT 38
                    LT 104 9
                    MESSAGE 8
                    ANYKEY
                    MESSAGE 9
                    ANYKEY
                    CLEAR 0
                    GOTO 92
                    CLS
                    RESTART

> ASUST    _        AT 54
                    LT 107 255
                    SET 107
                    MESSAGE 23
                    PLACE 21 54
                    DONE

> ASUST    _        EQ 118 2
                    MESSAGE 88
                    DONE

> ASUST    _        ATGT 73
                    ATLT 76
                    ZERO 110
                    MESSAGE 45
                    PROCESS 13

> REGAR    BROTE    AT 64
                    CARRIED 5
                    PLUS 109 1
                    SWAP 4 5
                    CLS
                    RESTART

> REGAR    BROTE    AT 64
                    CARRIED 11
                    MESSAGE 31
                    CLEAR 109
                    SWAP 11 4
                    DONE

> CAPUL    _        PLUS 33 1
                    ZERO 251
                    SET 251
                    MESSAGE 14
                    DONE

> CABRO    _        DESTROY 0
                    DESTROY 2
                    MESSAGE 14
                    ANYKEY
                    CLS
                    RESTART

> INTRO    MONED    AT 25
                    NOTZERO 117
                    CARRIED 1
                    DESTROY 1
                    CLEAR 8
                    MESSAGE 84
                    CREATE 7
                    DONE

> INTRO    MONED    AT 25
                    CARRIED 1
                    DESTROY 1
                    SET 117
                    OK

> INTRO    BATER    NOUN2 MAQUI
                    AT 25
                    NOTZERO 117
                    CARRIED 7
                    CLEAR 8
                    PLACE 7 25
                    MESSAGE 84
                    DONE

> INTRO    BATER    NOUN2 MAQUI
                    AT 25
                    CARRIED 7
                    DESTROY 7
                    SET 117
                    OK

> INTRO    BATER    CARRIED 7
                    CARRIED 2
                    DESTROY 7
                    SET 115
                    OK

> INTRO    BATER    NOTCARR 7
                    DROP 7
                    DONE

> BAÑAR    BOTEL    CARRIED 4
                    DESTROY 4
                    MESSAGE 32
                    DONE

> BAÑAR    BOTEL    CARRIED 5
                    DESTROY 5
                    MESSAGE 32
                    DONE

> BAÑAR    BOTEL    CARRIED 11
                    DESTROY 11
                    MESSAGE 32
                    DONE

> BAÑAR    HACHA    CARRIED 14
                    ZERO 119
                    EQ 118 2
                    CHANCE 50
                    MESSAGE 90
                    PLACE 14 255
                    SET 119
                    DONE

> BAÑAR    HACHA    CARRIED 14
                    EQ 118 2
                    MESSAGE 91
                    DESTROY 14
                    CLEAR 118
                    CLEAR 119
                    PLUS 120 1
                    DONE

> BAÑAR    _        AT 90
                    MESSAGE 3
                    CLEAR 0
                    GOTO 92
                    ANYKEY
                    CLS
                    RESTART

> BAÑAR    _        AT 56
                    MESSAGE 3
                    CLEAR 0
                    GOTO 92
                    ANYKEY
                    CLS
                    RESTART

> BAÑAR    _        WHATO
                    EQ 54 254
                    PUTO 255
                    OK

> BAÑAR    _        SYSMESS 28
                    DONE

> ABAND    _        QUIT
                    PROCESS 25
                    END

> ABAND    _        CLS
                    RESTART

> CRUZA    _        ATGT 73
                    ATLT 76
                    ZERO 110
                    MESSAGE 36
                    DONE

> CRUZA    _        ATGT 73
                    ATLT 76
                    EQ 111 255
                    MESSAGE 44
                    CLEAR 0
                    GOTO 92
                    ANYKEY
                    CLS
                    RESTART

> CRUZA    _        AT 74
                    GOTO 75
                    CLS
                    RESTART

> CRUZA    _        AT 75
                    GOTO 74
                    CLS
                    RESTART

> CRUZA    _        AT 11
                    GOTO 12
                    CLS
                    RESTART

> CRUZA    _        AT 12
                    GOTO 11
                    CLS
                    RESTART

> SACAR    BATER    CARRIED 0
                    CLEAR 115
                    PLACE 7 254
                    SWAP 0 2
                    CLS
                    RESTART

> SACAR    BATER    CARRIED 2
                    NOTZERO 115
                    PLACE 7 254
                    CLEAR 115
                    OK

> SALTA    _        AT 68
                    MESSAGE 112
                    DONE

> SALTA    _        AT 10
                    PROCESS 13

> SALTA    _        ATGT 73
                    ATLT 76
                    PROCESS 13

> D        _        CLS
                    RESTART

> FEE      FIE      AT 78
                    ISAT 24 78
                    MESSAGE 49
                    DONE

> FEE      FIE      AT 78
                    CARRIED 24
                    PLACE 24 78
                    MES 46
                    MESSAGE 49
                    DONE

> FEE      FIE      AT 78
                    MESSAGE 50
                    CREATE 24
                    DONE

> FEE      FIE      MESSAGE 115
                    DONE

> _        RECOL    AT 40
                    CLEAR 0
                    GOTO 91
                    CLS
                    RESTART

> _        RECOL    AT 91
                    SET 0
                    GOTO 40
                    CLS
                    CLS
                    RESTART

> _        RECOL    MESSAGE 115
                    DONE

> _        MAGIC    AT 40
                    CLEAR 0
                    GOTO 73
                    CLS
                    RESTART

> _        MAGIC    AT 73
                    CARRIED 27
                    PLACE 27 73

> _        MAGIC    AT 73
                    SET 0
                    GOTO 40
                    CLS
                    RESTART

> _        MAGIC    MESSAGE 115
                    DONE

> _        XYZZY    AT 5
                    CLEAR 0
                    GOTO 91
                    CLS
                    RESTART

> _        XYZZY    AT 91
                    CLS
                    SET 0
                    GOTO 5
                    CLS
                    RESTART

> _        XYZZY    MESSAGE 115
                    DONE

> _        PLOVE    MESSAGE 115
                    DONE

> DECIR    _        MESSAGE 14
                    DONE

> _        _        AT 48
                    CLEAR 105
                    SYSMESS 7
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
                    NEWTEXT
                    COPYFF 33 200
                    PLUS 200 116
                    MES @200
                    SYSMESS 14
                    PROCESS 14

; TABLE 6 is the game loop, in v3-loop.sce

; ----------------------------------------------------------------------
; TABLE 7 - SUB-PROCESS, called from table 3, 5.
; ----------------------------------------------------------------------
/PRO 7

> _        _        COPYFF 33 254

> BATRA    _        PLACE 2 254
                    PLACE 5 254
                    GOTO 2
                    DONE

> WHALK    _        PLACE 2 254
                    PLACE 4 254
                    GOTO 2
                    DONE

> TIMAC    _        PLACE 0 254
                    PLACE 5 254
                    GOTO 2
                    DONE

> GUACH    _        PLACE 0 254
                    PLACE 4 254
                    GOTO 2
                    DONE

> _        _        CLEAR 254

> CABRO    _        MES 14
                    DONE

> ABAND    _        CLS
                    CLEAR 255
                    END

> AYUDA    _        MES 105
                    DONE

> NO       _        MES 25
                    DONE

> _        _        MESSAGE 64

; ----------------------------------------------------------------------
; TABLE 8 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 8

> PALPA    PARED    AT 40
                    MESSAGE 100
                    DONE

> _        _        NOTZERO 0
                    ABSENT 0
                    MESSAGE 99
                    DONE

> _        BROTE    AT 64
                    MESSAGE 24
                    DONE

> _        ESPEJ    AT 55
                    MESSAGE 139
                    DONE

> _        ALMEJ    AT 49
                    MESSAGE 157
                    DONE

> _        LLAVE    CARRIED 6
                    MESSAGE 158
                    DONE

> _        CACA     AT 54
                    MESSAGE 159
                    DONE

> _        PUENT    ATGT 73
                    ATLT 76
                    MESSAGE 153
                    DONE

> _        PUENT    AT 11
                    MESSAGE 154
                    DONE

> _        HACHA    AT 78
                    MESSAGE 155
                    DONE

> _        TORTI    PRESENT 3
                    MESSAGE 156
                    DONE

> _        LAMPA    NOTZERO 115
                    LET 34 55

> _        BATER    NOTZERO 115
                    ABSENT 0
                    ABSENT 2
                    SYSMESS 28
                    DONE

> _        BATER    ZERO 115
                    ABSENT 7
                    SYSMESS 28
                    DONE

> _        BATER    MES 98
                    NOTZERO 8
                    PRINT 8
                    NEWLINE
                    DONE

> _        BATER    MESSAGE 102
                    DONE

> _        BARRO    AT 3
                    MESSAGE 17
                    DONE

> _        BARRO    AT 88
                    ZERO 113
                    MESSAGE 17
                    DONE

> _        BARRO    PRESENT 8
                    MESSAGE 141
                    DONE

> _        BARRO    CARRIED 10
                    MESSAGE 94
                    DONE

> _        PAJAR    CARRIED 10
                    MESSAGE 94
                    DONE

> _        PAJAR    AT 7
                    MESSAGE 142
                    DONE

> _        CARTE    AT 9
                    MESSAGE 34
                    DONE

> _        CARTE    AT 5
                    MESSAGE 92
                    DONE

> _        CARTE    AT 40
                    MESSAGE 108
                    DONE

> _        CARTE    AT 77
                    MESSAGE 149
                    DONE

> _        SERPI    AT 38
                    LT 104 9
                    MESSAGE 116
                    DONE

> _        FISUR    ATGT 9
                    ATLT 12
                    MESSAGE 144
                    DONE

> _        _        WHATO

> _        _        PRESENT @51
                    GT 51 15
                    MESSAGE 151
                    DONE

> _        _        PRESENT @51
                    LT 51 16
                    MESSAGE 152
                    DONE

> _        PARED    AT 5
                    MESSAGE 100
                    DONE

> _        PARED    AT 40
                    MESSAGE 100
                    DONE

> _        _        MESSAGE 55
                    DONE

; ----------------------------------------------------------------------
; TABLE 9 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 9

> _        _        ATGT 13
                    ATLT 26
                    MESSAGE 103
                    DONE

> _        _        AT 73
                    MESSAGE 110
                    DONE

> _        _        AT 5
                    MESSAGE 110
                    DONE

> _        _        AT 40
                    MESSAGE 161
                    DONE

> _        _        AT 10
                    MESSAGE 110
                    DONE

> _        _        NOTCARR 0
                    MESSAGE 35
                    DONE

> _        _        AT 9
                    MESSAGE 5
                    DONE

> _        _        MESSAGE 25
                    DONE

; ----------------------------------------------------------------------
; TABLE 10 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 10

> I        _        NOTZERO 0
                    ABSENT 0
                    MESSAGE 99
                    DONE

> I        _        NOTZERO 1
                    SYSMESS 9
                    LISTAT 254
                    DONE

> I        _        SYSMESS 11

; ----------------------------------------------------------------------
; TABLE 11 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 11

> _        _        WINDOW 7
                    CLS
                    WINDOW 2
                    PICTURE 82
                    DISPLAY 0
                    ANYKEY
                    WINDOW 6
                    MODE 2
                    WINAT 15 16
                    WINSIZE 6 15
                    CLS
                    MES 96
                    ANYKEY
                    WINDOW 7
                    CLS
                    GOTO 92
                    CLEAR 0
                    CLS
                    RESTART

; ----------------------------------------------------------------------
; TABLE 12 - SUB-PROCESS, called from table 4.
; ----------------------------------------------------------------------
/PRO 12

> _        _        WINDOW 7
                    CLS
                    SET 7
                    WINDOW 2
                    PICTURE 43
                    DISPLAY 0
                    WINDOW 6
                    WINAT 21 5
                    WINSIZE 4 20
                    MES 86
                    ANYKEY
                    WINDOW 7
                    CLS
                    WINDOW 1
                    CLS

; ----------------------------------------------------------------------
; TABLE 13 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 13

> _        _        MESSAGE 111
                    ANYKEY
                    GOTO 92
                    CLEAR 0
                    CLS
                    RESTART

; ----------------------------------------------------------------------
; TABLE 14 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 14

> _        _        CLEAR 201
                    LET 200 117
                    PROCESS 17
                    PROCESS 15
                    GT 201 1
                    MES 133

> _        _        EQ 201 1
                    MES 134

> _        _        MES @200
                    MESSAGE 131

; ----------------------------------------------------------------------
; TABLE 15 - SUB-PROCESS, called from table 14.
; ----------------------------------------------------------------------
/PRO 15

> _        _        COPYFF 38 202
                    MOVE 202
                    PROCESS 16

> _        _        PLUS 33 1
                    LT 33 13
                    REDO

; ----------------------------------------------------------------------
; TABLE 16 - SUB-PROCESS, called from table 15, 17.
; ----------------------------------------------------------------------
/PRO 16

> _        _        EQ 201 1
                    MES 130

> _        _        GT 201 1
                    MES 132

> _        _        NOTZERO 201
                    MES @200

> _        _        COPYFF 33 200
                    PLUS 200 116
                    PLUS 201 1

; ----------------------------------------------------------------------
; TABLE 17 - SUB-PROCESS, called from table 14.
; ----------------------------------------------------------------------
/PRO 17

> _        _        LET 33 13
                    COPYFF 38 202
                    MOVE 202
                    PROCESS 16

> _        _        LET 33 2

; ----------------------------------------------------------------------
; TABLE 18 - SUB-PROCESS. No PROCESS condact in this database reaches it: it is
; either dead code or entered some other way.
; ----------------------------------------------------------------------
/PRO 18

; ----------------------------------------------------------------------
; TABLE 19 - SUB-PROCESS, called from table 3.
; ----------------------------------------------------------------------
/PRO 19

> _        _        WINDOW 7
                    CLS
                    WINDOW 4
                    WINSIZE 24 127
                    PRINTAT 0 7

> _        _        MESSAGE 135
                    LET 150 14
                    DOALL 255
                    WHATO
                    GT 51 15
                    MESSAGE 63
                    MINUS 150 1

> _        _        LT 51 16
                    SYSMESS 10
                    NEWLINE

; ----------------------------------------------------------------------
; TABLE 20 - SUB-PROCESS, called from table 4.
; ----------------------------------------------------------------------
/PRO 20

> _        _        ISNOTAT 29 19
                    DONE

> _        _        WHATO
                    LT 51 16
                    COPYFF 46 34
                    WHATO
                    LT 51 16
                    DONE

> _        _        NOTEQ 54 254
                    NOTSAME 54 38
                    DONE

> _        _        CHANCE 80
                    DONE

> _        _        PUTO 19

> _        _        MESSAGE 97
                    ANYKEY
                    MESSAGE 87

; ----------------------------------------------------------------------
; TABLE 21 - SUB-PROCESS, called from table 4.
; ----------------------------------------------------------------------
/PRO 21

> _        _        NOTZERO 0
                    ABSENT 0
                    DONE

> _        _        WINDOW 0
                    PICTURE @38
                    DISPLAY 0
                    ZERO 40
                    PICTURE 95
                    DISPLAY 0

> _        _        WINDOW 1

; ----------------------------------------------------------------------
; TABLE 22 - SUB-PROCESS, called from table 3.
; ----------------------------------------------------------------------
/PRO 22

> _        _        NOTZERO 0
                    ABSENT 0
                    DONE

> _        _        AT 54
                    LT 107 255
                    WINDOW 7
                    CLS
                    WINDOW 2
                    PICTURE 82
                    DISPLAY 0
                    ANYKEY
                    WINDOW 7
                    CLS

> _        _        AT 54
                    WINDOW 0
                    PICTURE 95
                    DISPLAY 0
                    WINDOW 1

> _        _        AT 54
                    MES 18
                    LT 107 255
                    MES 19

> _        _        ATGT 73
                    ATLT 76
                    ZERO 110
                    MES 36

> _        _        AT 38
                    LT 104 9
                    MES 7

> _        _        AT 7
                    ZERO 122
                    MES 95

> _        _        AT 78
                    ZERO 114
                    MES 58

> _        _        AT 78
                    EQ 114 1
                    MES 59

> _        _        AT 78
                    GT 114 1
                    MES 60

> _        _        ATGT 19
                    ATLT 25
                    MES 65
                    AT 20
                    MES 66
                    MES 78
                    MES 79
                    MES 80

> _        _        AT 21
                    MES 81
                    MES 66
                    MES 79
                    MES 78

> _        _        AT 22
                    MES 66
                    MES 80
                    MES 79
                    MES 78

> _        _        AT 23
                    MES 81
                    MES 66
                    MES 78
                    MES 79

> _        _        AT 24
                    MES 66
                    MES 79
                    MES 78
                    MES 80

> _        _        ATGT 19
                    ATLT 25
                    SYSMESS 14

> _        _        AT 64
                    ZERO 109
                    MES 26

> _        _        AT 64
                    EQ 109 1
                    MES 27

> _        _        AT 64
                    GT 109 1
                    MES 28

> _        _        AT 89
                    ZERO 111
                    MES 52

> _        _        AT 89
                    EQ 111 2
                    MES 47

> _        _        EQ 111 255
                    MES 104

> _        _        ATGT 13
                    ATLT 20
                    MES 65
                    AT 14
                    MES 66
                    MES 69
                    MES 67
                    MES 68

> _        _        AT 15
                    MES 70
                    MES 66
                    MES 67
                    MES 68

> _        _        AT 16
                    MES 71
                    MES 66
                    MES 67
                    MES 69
                    SYSMESS 14
                    MES 72

> _        _        AT 17
                    MES 70
                    MES 66
                    MES 68
                    MES 67

> _        _        AT 18
                    MES 66
                    MES 67
                    MES 69
                    MES 68

> _        _        AT 19
                    MES 71
                    MES 66
                    MES 69
                    MES 67
                    SYSMESS 14
                    MES 73

> _        _        ATGT 13
                    ATLT 20
                    SYSMESS 14

> _        _        ATGT 9
                    ATLT 12
                    MES 160
                    ZERO 160
                    MES 120
                    MES 131
                    DONE

> _        _        ATGT 9
                    ATLT 12
                    MES 119
                    MES 131

; ----------------------------------------------------------------------
; TABLE 23 - SUB-PROCESS, called from table 4.
; ----------------------------------------------------------------------
/PRO 23

> _        _        NOTZERO 0
                    ABSENT 0
                    DONE

> _        _        AT 54
                    EQ 107 1
                    MESSAGE 22

> _        _        AT 54
                    EQ 107 2
                    MESSAGE 21

> _        _        AT 54
                    EQ 107 3
                    MESSAGE 20

; ----------------------------------------------------------------------
; TABLE 24 - SUB-PROCESS, called from table 5.
; ----------------------------------------------------------------------
/PRO 24

> DAR      TORTI    AT 89
                    ZERO 111
                    CARRIED 3
                    LET 111 2
                    MESSAGE 42
                    DESTROY 3
                    DONE

> DAR      _        AT 54
                    WHATO
                    EQ 54 254
                    PUTO 1
                    MESSAGE 53
                    DONE

> DAR      _        AT 89
                    WHATO
                    EQ 54 254
                    PUTO 1
                    MESSAGE 53
                    DONE

> DAR      _        NOTAT 75
                    GT 111 1
                    WHATO
                    EQ 54 254
                    PUTO 1
                    MESSAGE 53
                    DONE

> DAR      _        NOTAT 75
                    GT 111 1
                    SYSMESS 28
                    DONE

> DAR      _        AT 89
                    SYSMESS 28
                    DONE

> DAR      _        ATGT 73
                    ATLT 76
                    WHATO
                    NOTEQ 54 254
                    MESSAGE 38
                    DONE

> DAR      _        ATGT 73
                    ATLT 76
                    LT 51 16
                    MESSAGE 37
                    PUTO 1
                    DONE

> DAR      _        ATGT 73
                    ATLT 76
                    EQ 51 255
                    MESSAGE 38
                    DONE

> DAR      _        ATGT 73
                    ATLT 76
                    PUTO 1
                    MESSAGE 39
                    SET 110
                    DONE

> DAR      _        WHATO
                    NOTEQ 54 254
                    SYSMESS 28
                    DONE

> DAR      _        SYSMESS 8

; ----------------------------------------------------------------------
; TABLE 25 - SUB-PROCESS, called from table 3, 5.
; ----------------------------------------------------------------------
/PRO 25

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

