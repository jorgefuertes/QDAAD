package pack

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// What comes back is in the order it was asked for and not the order it
// finished in, which is the whole of what compressing them several at a time is
// allowed to change.
//
// The first picture is made to take its time so that the others really do
// finish ahead of it. Without that the test would pass on a machine with one
// core and prove nothing.
func TestThePicturesComeBackInTheOrderTheyWereAskedFor(t *testing.T) {
	numbers := []int{3, 7, 11, 19, 23, 29, 31}

	bodies, err := compressPictures(numbers, nil, func(number int) ([]byte, error) {
		if number == numbers[0] {
			time.Sleep(20 * time.Millisecond)
		}

		return []byte{byte(number)}, nil
	})
	require.NoError(t, err)
	require.Len(t, bodies, len(numbers))

	for i, number := range numbers {
		require.Equal(t, []byte{byte(number)}, bodies[i], "picture %d came back as somebody else", number)
	}
}

// And the error is the first picture's that failed, not the first failure to
// arrive, so that a build that cannot be made says the same thing every time.
func TestTheFirstPictureThatFailedIsTheOneReported(t *testing.T) {
	numbers := []int{3, 7, 11, 19, 23, 29, 31}

	_, err := compressPictures(numbers, nil, func(number int) ([]byte, error) {
		switch number {
		case 7:
			time.Sleep(20 * time.Millisecond)

			return nil, errors.New("the seventh")
		case 29:
			return nil, errors.New("the twenty-ninth")
		}

		return []byte{byte(number)}, nil
	})

	require.ErrorContains(t, err, "the seventh")
}

// Told once per picture, counting up to the total, however many are in hand at
// the same time. A count that repeats or stops short is a progress line that
// lies about how much is left.
func TestEveryPictureIsCountedOnce(t *testing.T) {
	numbers := []int{3, 7, 11, 19, 23, 29, 31}

	var counts []string

	_, err := compressPictures(numbers, func(done, total int, _ string) {
		counts = append(counts, fmt.Sprintf("%d/%d", done, total))
	}, func(number int) ([]byte, error) {
		return []byte{byte(number)}, nil
	})
	require.NoError(t, err)

	want := make([]string, 0, len(numbers))
	for i := range numbers {
		want = append(want, fmt.Sprintf("%d/%d", i+1, len(numbers)))
	}

	require.Equal(t, want, counts)
}

// Two builds of the same adventure are the same file, which is what the order
// above is for. The pictures go into the bank first-fit, so one that arrives out
// of turn would not merely be in the wrong place: it would push a different
// picture out of the bank altogether.
func TestTwoRunsOfTheSameBankAreTheSameBytes(t *testing.T) {
	opts := cpcBankOptions(5)

	first, err := cpcPictureBank(opts, 0x6000)
	require.NoError(t, err)

	second, err := cpcPictureBank(opts, 0x6000)
	require.NoError(t, err)

	require.Equal(t, first, second)
}
