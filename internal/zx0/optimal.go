package zx0

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
		// Per offset, the cheapest way to be here having just written
		// literals, and having just written a copy. A repeat can only follow
		// literals — after a copy the zero bit means "back to literals"
		// instead — so the two are kept apart rather than merged.
		lastLiteral = make([]handle, room+1)
		lastMatch   = make([]handle, room+1)
		// Per position, the cheapest way to be here at all.
		best = make([]handle, size)
		// How far the copy from each offset currently reaches.
		reach = make([]int, room+1)
		// For a copy ending here, the length that pays best. Cut short rather
		// than always the longest: a shorter copy can leave the stream in a
		// cheaper place than a longer one.
		pays = make([]int, size)
	)

	if size > 2 {
		pays[2] = 2
	}

	// The node from before the first byte, which is what makes a literal run
	// at position zero an edge like any other.
	p.assign(&lastMatch[initialOffset], p.allocate(-1, -1, initialOffset, none))

	for index := range size {
		paysUpTo := 2
		room = limit(index)

		for offset := 1; offset <= room; offset++ {
			if index > 0 && index >= offset && data[index] == data[index-offset] {
				// A copy of the offset already in hand, which is a bit and a
				// length and no offset at all.
				if from := lastLiteral[offset]; from != none {
					length := index - p.node(from).index
					bits := p.node(from).bits + 1 + eliasBits(length)

					p.assign(&lastMatch[offset], p.allocate(bits, index, offset, from))
					p.keepCheapest(&best[index], lastMatch[offset], bits)
				}

				// And a copy from this offset given afresh. Two bytes is the
				// shortest the format can say, so one matching byte is not yet
				// a copy.
				reach[offset]++
				if reach[offset] < minMatch {
					continue
				}

				// Which length pays best, worked out once per position and
				// carried forward: the answer for a reach of n is either n
				// itself or whatever answered for n-1.
				if paysUpTo < reach[offset] {
					cheapest := p.node(best[index-pays[paysUpTo]]).bits + eliasBits(pays[paysUpTo]-1)

					for paysUpTo < reach[offset] {
						paysUpTo++

						other := p.node(best[index-paysUpTo]).bits + eliasBits(paysUpTo-1)
						if other <= cheapest {
							pays[paysUpTo] = paysUpTo
							cheapest = other
						} else {
							pays[paysUpTo] = pays[paysUpTo-1]
						}
					}
				}

				length := pays[reach[offset]]
				from := best[index-length]
				bits := p.node(from).bits + 8 + eliasBits((offset-1)/128+1) + eliasBits(length-1)

				if at := lastMatch[offset]; at == none || p.node(at).index != index || p.node(at).bits > bits {
					p.assign(&lastMatch[offset], p.allocate(bits, index, offset, from))
					p.keepCheapest(&best[index], lastMatch[offset], bits)
				}

				continue
			}

			// This offset does not match here, so the byte is a literal and
			// the run it belongs to starts wherever the last copy left off.
			reach[offset] = 0

			if from := lastMatch[offset]; from != none {
				length := index - p.node(from).index
				bits := p.node(from).bits + 1 + eliasBits(length) + length*8

				p.assign(&lastLiteral[offset], p.allocate(bits, index, 0, from))
				p.keepCheapest(&best[index], lastLiteral[offset], bits)
			}
		}
	}

	return p.moves(best[size-1])
}

// eliasBits is what a value costs as an interlaced Elias gamma: two bits for
// every one below the leading one, and a bit to stop.
func eliasBits(value int) int {
	bits := 1
	for value >>= 1; value > 0; value >>= 1 {
		bits += 2
	}

	return bits
}

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
// reaches down 39 million nodes; left to the collector that was a gigabyte and
// a quarter of garbage and six tenths of a second apiece, which over the 110
// pictures of a two-part +3 build is a minute of packaging. Recycled, the same
// screen allocates 56 times, holds 7 MB, and takes a quarter of a second.
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

// keepCheapest points the slot at the node when it is the better way to be
// where it is.
func (p *pool) keepCheapest(slot *handle, to handle, bits int) {
	if *slot == none || p.nodes[*slot].bits > bits {
		p.assign(slot, to)
	}
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
