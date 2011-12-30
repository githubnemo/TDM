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
	incoming chan string	// Incoming log data
	stop chan bool			// Stop looping
}

func (s *Sink) Start() {
	go s.sinkLoop()
}

func (s *Sink) Stop() {
	close(s.incoming)

	s.stop <- true

	for len(s.incoming) > 0 {
		s.write(<-s.incoming)
	}

	s.logfile.Close()
}

func (s *Sink) sinkLoop() {
	for {
		select {
		case v := <-s.incoming:
			s.write(v)
		case <-s.stop:
			break
		}
	}
}

func (s *Sink) write(v string) {
	fmt.Fprintln(s.logfile, v)
}


func (s* Sink) Feed(data string) {
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
		make(chan string, 32),
		make(chan bool),
	}, nil
}
