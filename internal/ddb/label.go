package ddb

const LabelUndefined ID16 = 0

type Label struct {
	ID   ID16
	Name string
}

type Labels []Label

func NewLabelStore() Labels {
	return Labels{
		{ID: LabelUndefined, Name: "undefined"},
	}
}

func (ls *Labels) Add(name string) (ID16, error) {
	if id, exists := ls.GetByName(name); exists {
		return id, nil
	}

	var maxID ID16 = 1
	for _, l := range *ls {
		if l.ID > maxID {
			maxID = l.ID
		}
	}

	*ls = append(*ls, Label{
		ID:   maxID + 1,
		Name: name,
	})

	return maxID + 1, nil
}

func (ls Labels) Get(id ID16) (string, bool) {
	for _, l := range ls {
		if l.ID == id {
			return l.Name, true
		}
	}

	return "", false
}

func (ls Labels) GetByName(name string) (ID16, bool) {
	for _, l := range ls {
		if l.Name == name {
			return l.ID, true
		}
	}

	return 0, false
}
