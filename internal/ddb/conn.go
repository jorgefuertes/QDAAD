package ddb

type Conn struct {
	From ID
	// Word is the vocabulary value that leads out of From. The classic
	// directions are the values under 14, but nothing caps it there: the
	// compiler of reference only asks that the word be a verb or a noun and
	// takes its value as it comes, so a stricter check here would refuse
	// sources it accepts.
	Word ID
	To   ID
}

type Conns []Conn

func NewConnStore() Conns {
	return Conns{}
}

func (cs Conns) Get(from ID, word ID) (Conn, bool) {
	for _, c := range cs {
		if c.From == from && c.Word == word {
			return c, true
		}
	}

	return Conn{}, false
}

func (cs Conns) GetByLocation(from ID) []Conn {
	var conns []Conn
	for _, c := range cs {
		if c.From == from {
			conns = append(conns, c)
		}
	}

	return conns
}

func (cs *Conns) Add(from ID, word ID, to ID) {
	*cs = append(*cs, Conn{
		From: from,
		Word: word,
		To:   to,
	})
}
