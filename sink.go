package main

import (
	"os"
	"fmt"
	"path"
)

type Sink struct {
	team int				// Team number
	station int				// Station number
	logfile *os.File		// Logfile
	incoming chan []byte	// Incoming log data
}

func (s *Sink) Start() {
	go s.doSink()
}

func (s *Sink) Stop() {
	for len(s.incoming) > 0 {
		s.doSink()
	}

	s.logfile.Close()
}

func (s *Sink) doSink() {
	select {
		case v := <-s.incoming:
			fmt.Fprintln(s.logfile, v)
		default:
			return
	}
}

func (s* Sink) Feed(data []byte) {
	s.incoming <- data
}

func NewSink(team, station int, dir string) (*Sink, os.Error) {

	fileName := fmt.Sprintf("sink-%d-%d.log", team, station)

	logFilePath := path.Join(dir, fileName)

	file, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)

	if err != nil {
		return nil, err
	}

	return &Sink{
		team,
		station,
		file,
		make(chan []byte, 32),
	}, nil
}
