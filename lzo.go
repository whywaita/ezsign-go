package ezsign

// lzoCompress encodes LZO1X with literal runs and M3 matches. Blocks are small
// (2000 bytes), so the 16 KiB M3 window covers every possible back-reference.
// The greedy encoder is independent of MiniLZO; byte streams need not match it.
func lzoCompress(b []byte) []byte {
	if len(b) == 0 {
		return []byte{17, 0, 0}
	}
	type match struct{ pos, length, distance int }
	var matches []match
	positions := make(map[uint32]int)
	key := func(i int) uint32 { return uint32(b[i])<<16 | uint32(b[i+1])<<8 | uint32(b[i+2]) }
	for i := 0; i+3 <= len(b); {
		k := key(i)
		previous, found := positions[k]
		positions[k] = i
		n := 0
		if found && i-previous <= 16384 {
			for i+n < len(b) && b[previous+n] == b[i+n] {
				n++
			}
		}
		if n < 4 {
			i++
			continue
		}
		matches = append(matches, match{i, n, i - previous})
		for j := i + 1; j < i+n && j+3 <= len(b); j++ {
			positions[key(j)] = j
		}
		i += n
	}
	if len(matches) == 0 {
		return lzoLiteral(b)
	}
	var out []byte
	extended := func(n int) {
		for n > 255 {
			out = append(out, 0)
			n -= 255
		}
		out = append(out, byte(n))
	}
	first := matches[0].pos
	if first <= 238 {
		out = append(out, byte(first+17))
	} else {
		out = append(out, 0)
		extended(first - 18)
	}
	out = append(out, b[:first]...)
	for i, m := range matches {
		if m.length <= 33 {
			out = append(out, byte(32+m.length-2))
		} else {
			out = append(out, 32)
			extended(m.length - 33)
		}
		offset := m.distance - 1
		offsetPos := len(out)
		out = append(out, byte(offset<<2), byte(offset>>6))
		end := len(b)
		if i+1 < len(matches) {
			end = matches[i+1].pos
		}
		literals := b[m.pos+m.length : end]
		if len(literals) <= 3 {
			out[offsetPos] |= byte(len(literals))
		} else if len(literals) <= 18 {
			out = append(out, byte(len(literals)-3))
		} else {
			out = append(out, 0)
			extended(len(literals) - 18)
		}
		out = append(out, literals...)
	}
	return append(out, 17, 0, 0)
}
