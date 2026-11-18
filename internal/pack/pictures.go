package pack

import (
	"fmt"
	"runtime"
	"sync"
)

// compressPicturesAtOnce is how many pictures are compressed at a time at most,
// however many cores there are.
//
// A ceiling because what the search wants is memory and not cores. A 16K
// Amstrad screen holds about 28 MB of it while ZX0 parses the screen as a
// shortest path, every worker holds its own, and none of that fits in any cache
// — so past a handful of them the workers are queueing for the same memory
// rather than working. Measured over the 29 pictures of an Amstrad part on a
// ten-core M4, where one at a time is 18.6s:
//
//	two      9.9s wall, 18.7s of processor
//	four     5.9s wall, 21.8s
//	six      5.1s wall, 28.4s
//	eight    4.9s wall, 34.9s
//	ten      5.3s wall, 36.3s
//
// Six is where it stops being worth it: the second of wall clock that eight
// buys costs a quarter more processor and another 56 MB, and ten is slower than
// six outright. Below the ceiling it follows GOMAXPROCS, so a machine with
// fewer cores uses fewer and anybody who wants fewer still can say so.
const compressPicturesAtOnce = 6

// compressPictures builds the body of every picture given, several at a time,
// and hands them back in the order the numbers came in.
//
// That order is the whole of the contract. What the caller does with the bodies
// — which of them fit, which are left behind, where each one lands — is decided
// afterwards and in sequence, because two builds of the same adventure have to
// come out the same file, and "whichever finished first" is not an order. For
// the same reason the error that comes back belongs to the first number that
// failed and not to the first failure to arrive, and it comes back as build
// gave it: which picture it was about is build's to say, and it says it
// differently in each of the three places this is called from.
//
// One goroutine per picture and no finer: a picture is a whole screen through a
// quadratic search, which is tenths of a second, and there are tens of them.
// zx0.Compress holds nothing between calls, so there is nothing to share.
func compressPictures(
	numbers []int, report Progress, build func(number int) ([]byte, error),
) ([][]byte, error) {
	bodies := make([][]byte, len(numbers))
	errs := make([]error, len(numbers))

	var (
		// taken is the next picture to pick up and done how many are finished,
		// both under the one lock, which also serializes the reporting.
		taken, done int
		turn        sync.Mutex
		workers     sync.WaitGroup
	)

	for range min(runtime.GOMAXPROCS(0), compressPicturesAtOnce, len(numbers)) {
		workers.Go(func() {
			for {
				turn.Lock()
				at := taken
				taken++
				turn.Unlock()

				if at >= len(numbers) {
					return
				}

				bodies[at], errs[at] = build(numbers[at])

				// Told under the lock because a progress line is drawn over the
				// last one rather than after it, and two of those at once is
				// neither of them. What it counts is how many are finished,
				// which is the only count that means anything when eight are in
				// hand at the same time.
				turn.Lock()
				done++
				report.at(done, len(numbers), fmt.Sprintf("compressing picture %03d", numbers[at]))
				turn.Unlock()
			}
		})
	}

	workers.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return bodies, nil
}
