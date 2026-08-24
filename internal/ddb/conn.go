package ddb

type Conn struct {
	From ID
	Word ID `valid:"required,min=0,max=13"`
	To   ID
}
