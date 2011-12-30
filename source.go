package main

import "fmt"

type Source struct {
	team int
	station int
	seq int
}

func (s *Source) Data() []byte {
	defer func() { s.seq++ }()

	str := fmt.Sprintf("team %d-%d Message %d from us!", s.team, s.station, s.seq)

	return []byte(str)
}

func NewSource(team, station int) *Source {
	return &Source{
		team: team,
		station: station,
		seq: 0,
	}
}
