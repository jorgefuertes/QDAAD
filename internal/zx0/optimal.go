package zx0

import "math/bits"

// The parse: which literals and which copies a block comes out as.
//
// Choosing is a separate problem from coding, and the harder of the two. Any
// run of literals and copies that spells the data back is a correct ZX0
// stream, so a coder cannot be wrong here, only wasteful — and the waste is
// real money on a 48K tape, where what fits is counted in pictures.
//
// Taking the longest copy available at each step, which is what this did
// first, is the obvious way and not the cheap one: a long copy here can cost
// more than it saves by leaving the next few bytes with nothing good to point
// at, and a copy worth 8 bits and a copy worth 1 bit are not the same copy.
// Measured against Saukas' own coder over the screens of La Aventura Original,
// greedy came out a tenth bigger, and a third bigger on the worst block.
//
// So the choice is made the way he makes it: as a shortest path. Every
// position is a node, every literal run and every copy is an edge weighted by
// the bits it costs, and what comes out is the cheapest way through — not a
// good parse but the best one. Over those same screens it now ties with his
// coder to the byte, which is what make compare-zx0 reports.
//
// The algorithm is the one in work/ZXDAAD128/DCP/optimize.c, and the debt is
// stated rather than hidden: the cost model is his because it is the format's,
// and a second opinion about what a copy costs would only be a worse one. What
// is here is that model in Go over our own writer.
//
// What is not his is how little of the search is written down. His coder makes
// a node for every candidate it weighs, and a 16K Amstrad screen weighs a
// hundred and eighty-three million of them; all but a handful are overwritten
// by the next position and thrown away. Here a candidate is a few numbers in an
// array until something actually asks it for a node, which is the same parse
// over fifty-five million nodes instead, and a 16K screen in half a second
// instead of a second and a quarter.
//
// The parse is unchanged by all of it — the same stream byte for byte, not
// merely one of the same size — and each place where that is not obvious says
// why beside the code.

// move is one thing a stream says.
type move struct {
	// offset is where a copy reads from, and zero means this is a run of
	// literals instead.
	offset int
	length int
}

// initialOffset is what the decoder starts with in hand. The first copy of a
// stream can be a repeat of it without ever having been given an offset, which
// is why the search starts from a node that claims it.
const initialOffset = 1

// noRun is where an offset has no run of literals behind it yet. Zero will not
// do for that: position zero is where the first run of all of them ends.
const noRun int32 = -1

// optimalParse works out the cheapest way to say data, as the moves that say
// it in order.
//
// The three edges are the three things a stream can say, and the weights are
// what our writer spends on them:
//
//   - a run of L literals: one bit, the length, and the bytes
//   - a copy of L from the offset in hand: one bit and the length
//   - a copy of L from a new offset: the offset's two halves and the length,
//     less the one bit the length's first rides out inside the low byte
//
// Only the last of those needs explaining, and match says it: the length's
// leading bit is written into the bottom of the offset's low byte, so a new
// offset and its length together cost one bit less than the two apart.
func optimalParse(data []byte) []move {
	size := len(data)

	limit := func(index int) int {
		switch {
		case index > maxOffset:
			return maxOffset
		case index < initialOffset:
			return initialOffset
		default:
			return index
		}
	}

	room := limit(size - 1)
	p := newPool(size)

	var (
		// Per offset, the cheapest way to be here having just written a copy.
		lastMatch = make([]handle, room+1)
		// And per offset, the cheapest way to be here having just written
		// literals, which is the other thing a copy from it can follow: a
		// repeat can only come after literals, because after a copy the zero
		// bit means "back to literals" instead.
		//
		// It is held as the three numbers that describe it and not as a node,
		// because almost none of these are ever looked at: one is made at every
		// position an offset fails to match at, and the next position throws it
		// away. litNode is the node when something has asked for one, and none
		// when the run is still only described.
		litBits = make([]int32, room+1)
		litPos  = make([]int32, room+1)
		litNode = make([]handle, room+1)
		// Per position, the cheapest way to be here at all.
		best = make([]handle, size)
		// How far the copy from each offset currently reaches.
		reach = make([]int32, room+1)
		// For a copy ending here, the length that pays best. Cut short rather
		// than always the longest: a shorter copy can leave the stream in a
		// cheaper place than a longer one.
		pays = make([]int32, size)
		// What a copy spends saying which offset it is, which depends on the
		// offset and on nothing else — so it is worked out once per offset here
		// instead of once per offset per position, which over a 16K screen is
		// the difference between sixteen thousand divisions and sixty million.
		offsetBits = make([]uint8, room+1)
	)

	for offset := range litPos {
		litPos[offset] = noRun
	}

	for offset := 1; offset <= room; offset++ {
		offsetBits[offset] = uint8(eliasBits((offset-1)/128 + 1))
	}

	if size > 2 {
		pays[2] = 2
	}

	// The node from before the first byte, which is what makes a literal run
	// at position zero an edge like any other.
	p.assign(&lastMatch[initialOffset], p.allocate(-1, -1, initialOffset, none))

	// And position zero itself, which the loop below is not shaped for: no
	// offset can match where there is nothing behind, so the only edge out of
	// here is that first run of literals.
	firstFrom := lastMatch[initialOffset]
	firstLength := 0 - p.node(firstFrom).index
	firstBits := p.node(firstFrom).bits + 1 + eliasBits(firstLength) + firstLength*8

	litBits[initialOffset], litPos[initialOffset] = int32(firstBits), 0
	p.assign(&best[0], p.allocate(firstBits, 0, 0, firstFrom))

	for index := 1; index < size; index++ {
		paysUpTo := 2
		room = limit(index)
		here := data[index]

		// The cheapest way to be at this position, chosen before anything is
		// built for it: the node when the candidate that won already has one,
		// and otherwise the offset whose run of literals would have to become
		// one. Every candidate here ends at this same position, so one node
		// covers whichever of them wins.
		var (
			cheapest   int
			cheapestAt handle
			cheapestOf = -1
			chosen     bool
		)

		// offset counts up and back counts down, which is the same pair of
		// numbers without the subtraction. Neither of the guards the condition
		// used to carry is needed: there is something behind every position
		// from the first, and limit never hands back more room than there is.
		for offset, back := 1, index-1; offset <= room; offset, back = offset+1, back-1 {
			if here == data[back] {
				// A copy of the offset already in hand, which is a bit and a
				// length and no offset at all.
				repeat, repeatFrom, hasRepeat := 0, none, false

				if litPos[offset] != noRun {
					length := index - int(litPos[offset])
					repeat = int(litBits[offset]) + 1 + eliasBits(length)
					hasRepeat = true

					// The run has to become a node here and it cannot wait: its
					// chain is the offset's last copy, and that is what the
					// lines below are about to replace. Once made it stands
					// until the offset fails to match again, which is when a
					// different run starts.
					if repeatFrom = litNode[offset]; repeatFrom == none {
						repeatFrom = p.allocate(
							int(litBits[offset]), int(litPos[offset]), 0, lastMatch[offset])
						p.assign(&litNode[offset], repeatFrom)
					}
				}

				// And a copy from this offset given afresh. Two bytes is the
				// shortest the format can say, so one matching byte is not yet
				// a copy.
				reach[offset]++

				fresh, freshFrom, hasFresh := 0, none, false

				if reach[offset] >= minMatch {
					// Which length pays best, worked out once per position and
					// carried forward: the answer for a reach of n is either n
					// itself or whatever answered for n-1.
					if paysUpTo < int(reach[offset]) {
						at := int(pays[paysUpTo])
						cheapestLength := p.node(best[index-at]).bits + eliasBits(at-1)

						for paysUpTo < int(reach[offset]) {
							paysUpTo++

							other := p.node(best[index-paysUpTo]).bits + eliasBits(paysUpTo-1)
							if other <= cheapestLength {
								pays[paysUpTo] = int32(paysUpTo)
								cheapestLength = other
							} else {
								pays[paysUpTo] = pays[paysUpTo-1]
							}
						}
					}

					length := int(pays[reach[offset]])
					freshFrom = best[index-length]
					fresh = p.node(freshFrom).bits + 8 + int(offsetBits[offset]) + eliasBits(length-1)
					hasFresh = true
				}

				// Only one of the two is worth a node. The dearer is not what
				// the offset is left holding, and it is not the cheapest way to
				// be here either, because the other one is cheaper and is here
				// as well. A tie goes to the repeat, which is the order the two
				// were weighed in when each was built before being compared.
				switch {
				case hasRepeat && (!hasFresh || repeat <= fresh):
					at := p.allocate(repeat, index, offset, repeatFrom)
					p.assign(&lastMatch[offset], at)

					if !chosen || repeat < cheapest {
						cheapest, cheapestAt, cheapestOf, chosen = repeat, at, -1, true
					}

				case hasFresh:
					at := p.allocate(fresh, index, offset, freshFrom)
					p.assign(&lastMatch[offset], at)

					if !chosen || fresh < cheapest {
						cheapest, cheapestAt, cheapestOf, chosen = fresh, at, -1, true
					}
				}

				continue
			}

			// This offset does not match here, so the byte is a literal and
			// the run it belongs to starts wherever the last copy left off.
			reach[offset] = 0

			if from := lastMatch[offset]; from != none {
				length := index - p.node(from).index
				bits := p.node(from).bits + 1 + eliasBits(length) + length*8

				litBits[offset], litPos[offset] = int32(bits), int32(index)

				// Whatever was made of the run a byte ago ends one byte short
				// of this one, so it is not this run and nothing will ask for
				// it again.
				if at := litNode[offset]; at != none {
					p.release(at)
					litNode[offset] = none
				}

				if !chosen || bits < cheapest {
					cheapest, cheapestAt, cheapestOf, chosen = bits, none, offset, true
				}
			}
		}

		// And now that it is known which candidate won, it gets its node. For a
		// run of literals that is the only node it will ever have.
		//
		// Keeping the strictly cheaper of two while the offsets go up in order
		// settles a tie the way building every candidate and keeping the cheaper
		// of it and whatever was there already did: the first of them wins. That
		// is what makes this the same stream and not merely one of the same
		// length.
		if chosen {
			at := cheapestAt
			if at == none {
				at = p.allocate(cheapest, index, 0, lastMatch[cheapestOf])
			}

			p.assign(&best[index], at)
		}
	}

	return p.moves(best[size-1])
}

// eliasBits is what a value costs as an interlaced Elias gamma: two bits for
// every one below the leading one, and a bit to stop.
//
// Which is two bits per bit of the value, less the one the stop bit replaces.
// It was a loop over the shifts and that loop was a fifth of the whole search,
// so it is the one instruction that counts them instead. Nothing asks what zero
// costs — every value through here is a length or a count of 128s, and both
// start at one.
func eliasBits(value int) int { return 2*bits.Len(uint(value)) - 1 }

// A node is not a position but a way of having reached one, and the difference
// is the whole reason this works: arriving at byte 400 with an offset of 12 in
// hand is worth more than arriving there with an offset of 3000, because the
// next copy from 12 costs a bit and the next copy from 3000 costs eleven.
type node struct {
	// bits is what the stream costs up to and including index.
	bits int
	// index is the last byte consumed.
	index int
	// offset is the one in hand here, and zero means a run of literals.
	offset int
	// chain is how this was arrived at.
	chain handle
	// refs is how many places point at this one, and ghost chains it onto the
	// free list once nothing does. See pool.
	refs  int
	ghost handle
}

// handle names a node, and none names no node at all.
//
// They are positions in a slice and not pointers, which is the whole point of
// them: a search over a 6912-byte screen reaches down a few million of these,
// and a few million pointers is a few million things for the collector to walk
// every time it runs. A slice of plain values holds nothing it has to look at.
type handle int32

const none handle = 0

// pool hands nodes out and takes them back.
//
// The search allocates far more nodes than it keeps — every offset at every
// position is a candidate and almost none of them survive — so what matters is
// not how fast a node is made but that making one does not cost memory. This
// is memory.c in Go: a node nothing points at goes on a free list and the next
// allocation takes it back, so what is held at once is the live set rather
// than the history of the search.
//
// It is the difference between this being usable and not. A 6912-byte screen
// reaches down 18 million nodes; left to the collector that was a gigabyte and
// a quarter of garbage and six tenths of a second apiece, which over the 110
// pictures of a two-part +3 build is a minute of packaging. Recycled, the same
// screen allocates 58 times, holds 7 MB, and takes an eighth of a second.
type pool struct {
	// nodes[0] is never used: it is what none refers to, so that a zero
	// handle means nothing without a separate flag beside every one of them.
	nodes []node
	ghost handle
}

func newPool(size int) *pool {
	return &pool{nodes: make([]node, 1, size+2)}
}

// node reads one. A caller must not hold what this returns across an allocate,
// which may move the slice.
func (p *pool) node(h handle) *node { return &p.nodes[h] }

func (p *pool) allocate(bits, index, offset int, chain handle) handle {
	var h handle

	if p.ghost != none {
		h = p.ghost
		p.ghost = p.nodes[h].ghost

		// What it used to point at loses a reference, and may itself fall out
		// of use. One step is enough: whatever that frees will be freed in
		// its turn when it is reused.
		p.release(p.nodes[h].chain)
	} else {
		p.nodes = append(p.nodes, node{})
		h = handle(len(p.nodes) - 1)
	}

	if chain != none {
		p.nodes[chain].refs++
	}

	p.nodes[h] = node{bits: bits, index: index, offset: offset, chain: chain}

	return h
}

// assign points a slot at a node, taking a reference for it and giving up the
// one the slot held.
func (p *pool) assign(slot *handle, to handle) {
	p.nodes[to].refs++
	p.release(*slot)
	*slot = to
}

// release gives up one reference, and frees the node if that was the last.
func (p *pool) release(h handle) {
	if h == none {
		return
	}

	p.nodes[h].refs--
	if p.nodes[h].refs > 0 {
		return
	}

	p.nodes[h].ghost = p.ghost
	p.ghost = h
}

// moves turns the chain the search built, which runs backwards from the last
// node, into the moves a writer puts down in order.
func (p *pool) moves(last handle) []move {
	var chain []handle

	for at := last; at != none; at = p.nodes[at].chain {
		chain = append(chain, at)
	}

	// The last of these is the node from before the first byte, which stands
	// for nothing written: every move is the step from the node behind it.
	out := make([]move, 0, len(chain)-1)

	for i := len(chain) - 2; i >= 0; i-- {
		at, from := p.nodes[chain[i]], p.nodes[chain[i+1]]
		out = append(out, move{offset: at.offset, length: at.index - from.index})
	}

	return out
}
