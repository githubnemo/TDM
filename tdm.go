package main

import (
	"os"
	"fmt"
	"net"
	"log"
	"flag"
	"time"
	"bytes"
	"strconv"
)

const (
	// Time per frame in nanoseconds
	FRAME_TIME = 1 * 1e9

	// Number of slots per frame
	SLOTS = 20

	// Time per slot in nanoseconds
	SLOT_TIME = int64(FRAME_TIME / SLOTS)
)

// milli => 1e3
// micro => 1e6
// nano  => 1e9


// Wait for next frame and return start time of next frame in NS
func syncWithNextFrame() int64 {
	cns := time.Nanoseconds()

	startns := (cns / 1e9 + 1) * 1e9

	time.Sleep(startns - cns)

	return startns
}

// Return timestamp of current frame beginning
func frameBeginTime() int64 {
	return time.Nanoseconds() / 1e9 * 1e9
}

func syncWithSlotCenter(frameBegin int64, slot byte) {
	waitNs := (frameBegin - time.Nanoseconds())
	waitNs += int64(slot) * SLOT_TIME + SLOT_TIME / 2

	time.Sleep(waitNs)
}


func syncWithSlotEnd(frameBegin int64, slot byte) {
	waitNs := (frameBegin - time.Nanoseconds())
	waitNs += int64(slot) * SLOT_TIME + SLOT_TIME

	time.Sleep(waitNs)
}


// Return a packet if there was one, return a nil packet
// empty=true and a nil error if there was no packet and
// the read timed out, return nil, false and an error
// otherwise.
func readSlot(conn *MultiCastConn) (*Packet, bool, os.Error) {
	byteSlice := make([]byte, PACKET_SIZE)
	buffer := bytes.NewBuffer(byteSlice)

	n, err := conn.Read(byteSlice)

	if err != nil {
		if err.(net.Error).Timeout() {
			return nil, true, nil
		}
		return nil, false, err
	}

	// timeout, assume empty
	if n == 0 {
		log.Println("Empty slot!")
		return nil, true, nil
	}

	buffer.Write(byteSlice[:n])

	// Grab some more bytes if we didn't get all
	if n != PACKET_SIZE {
		return nil, false, os.NewError(
			fmt.Sprintf("Invalid packet size: %d", n))
	}

	packet, perr := NewPacketFromReader(buffer)

	if perr != nil {
		return nil, false, perr
	}

	return packet, false, nil
}


func sendPacket(conn *MultiCastConn, payload []byte, slot byte) os.Error {
	ms := time.Nanoseconds() / 1e6

	p := NewPacket(payload, slot, ms)

	_, err := p.SendTo(conn)

	return err
}


var g_station *int = flag.Int("station", 1, "Station number")
var g_port *int = flag.Int("port", 15000, "Base port, real port = port + team")
var g_team *int = flag.Int("team", 17, "Team number")
var g_ip *string = flag.String("ip", "225.10.1.2", "Multicast address to listen on")
var g_logDir *string = flag.String("logdir", "../log/", "Log directory")

func main() {
	flag.Parse()

	team, station := *g_team, *g_station

	logPath := *g_logDir

	// Setup source & sink
	source := NewSource(team, station)

	sink, serr := NewSink(team, station, logPath)

	if serr != nil {
		fmt.Println(serr.String())
		return
	}

	sink.Start()
	defer sink.Stop()

	// Open network connection
	ip := *g_ip
	port := *g_port

	conn, cerr := JoinMulticast(ip, strconv.Itoa(port + team))

	// Limit the max. read time to the slot time
	log.Printf("Setting read timeout to: %d ns", SLOT_TIME)
	conn.SetReadTimeout(SLOT_TIME)

	if cerr != nil {
		log.Fatal("Network join error:", cerr.String())
	}

	// initially use the first slot
	mySlot := byte(0)
	currentPayload := source.Data()
	searchNewSlot := false

	// Wait for next frame to begin and start process
	syncWithNextFrame()

	for {
		frameBegin := frameBeginTime()
		go log.Println("Begin!")

		for i:= byte(0); i < SLOTS; i++ {
			go log.Println("Slot", i)

			// We have a good slot and this is our slot, send!
			if i == mySlot && !searchNewSlot {
				syncWithSlotCenter(frameBegin, mySlot)
				sendPacket(conn, currentPayload, mySlot)
				go log.Println("Sent package!")
			}

			packet, empty, err := readSlot(conn)

			if err != nil {
				log.Fatalf("Error while reading slot %d: %s", i, err.String())
			}

			// Empty slot and in need of one? Take it!
			if empty && searchNewSlot {
				mySlot = i
				searchNewSlot = false
			}

			// No empty slot, a packet was there
			if packet != nil {

				// Detect collision
				if i == mySlot && packet.EqualPayload(currentPayload) {
					searchNewSlot = true
					go log.Println("Collision in slot", i)
					sink.Feed(fmt.Sprintf("Collision in slot %d!", i))
				} else {
					currentPayload = source.Data()
					go log.Println("Everything went fine, sent data.")
				}

				sink.Feed(packet.String())
			}

			syncWithSlotEnd(frameBegin, i)
		}
	}
}
