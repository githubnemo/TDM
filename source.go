package main

import "fmt"
import "strings"
import "strconv"

type Source struct {
	team int
	station int
	seq int
}

// Returns 24 Byte of data
func (s *Source) Data() []byte {
	defer func() { s.seq++ }()

	seq := strconv.Itoa(s.seq)
	seq  = strings.Repeat("0", 11-len(seq)) + seq

	str := fmt.Sprintf("team %0d-%0d M#%s",
					   s.team % 100,
					   s.station % 100,
					   seq)

	return []byte(str)
}

func NewSource(team, station int) *Source {
	return &Source{
		team: team,
		station: station,
		seq: 0,
	}
}
