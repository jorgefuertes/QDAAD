package legacy

import (
	"errors"
	"fmt"
)

// An error that says where in the source it happened, and says it once.
//
// The front end finds a failure deep down and then adds context on the way
// back up — "parameter 1", then the condact's name, then "process 0" — and
// every one of those layers knows a position of its own. Each of them adding
// one put four copies of the same file:line in a single sentence, with the
// innermost and most precise of them buried in the middle:
//
//	t.sce:242:17: process 0: t.sce:242:17: XMES: t.sce:242:17: parameter 1:
//	t.sce:241:25: in "Un xmensaje": "Un" is not defined
//
// So the position travels in the error rather than in its text. The deepest
// layer sets it, the layers above it add words only, and it comes out once and
// at the front — which is where an editor goes looking for it, and where a
// person reading a column of errors wants it.
type placed struct {
	where string
	// message is the text, with no position anywhere in it.
	message string
	// err is what this wraps, kept so that errors.Is and errors.As still see
	// through it.
	err error
}

func (p *placed) Error() string { return p.where + ": " + p.message }
func (p *placed) Unwrap() error { return p.err }

// placedf builds an error at the position given.
//
// A position carried by one of the args wins over the one passed in: that one
// was set further down, closer to what actually went wrong. "In this process,
// in this condact, in this parameter" is context; the line the parameter is on
// is the answer.
func placedf(where, format string, args ...any) error {
	text := make([]any, len(args))
	copy(text, args)

	var wrapped error

	for i, a := range args {
		err, ok := a.(error)
		if !ok {
			continue
		}

		if wrapped == nil {
			wrapped = err
		}

		// Its text goes in without its position, because the position is
		// about to go in front of the whole sentence.
		var deep *placed
		if errors.As(err, &deep) {
			where = deep.where
			text[i] = errors.New(deep.message)
		}
	}

	// fmt.Errorf and not Sprintf: the format may carry a %w and only one of
	// the two understands it. What is taken from it is the text — the wrapping
	// is placed's own, through Unwrap.
	return &placed{
		where:   where,
		message: fmt.Errorf(format, text...).Error(),
		err:     wrapped,
	}
}
