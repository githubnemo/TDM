package main

import "fmt"

type Source struct {
	team int
	station int
	seq int
}

func (s *Source) Data() string {
	defer func() { s.seq++ }()
	return fmt.Sprintf("team %d-%d Message %d from us!", s.team, s.station, s.seq)
}

func NewSource(team, station int) *Source {
	return &Source{
		team: team,
		station: station,
		seq: 0,
	}
}
