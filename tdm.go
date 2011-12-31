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
	"io/ioutil"
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


// Sets the read timeout according to the elapsed time of the slot.
func setProperReadTimeout(conn *MultiCastConn, frameBegin int64, slot byte) {
	now := time.Nanoseconds()

	// rest slot time = SLOT_TIME - elapsed slot time
	timeout := SLOT_TIME - (now - (frameBegin + int64(slot) * SLOT_TIME))
	timeout -= 20e6 // some threshold for code execution after read

	conn.SetReadTimeout(timeout)
}


// Read SLOTS * PACKET_SIZE bytes from the connection and
// return the first arrived packet.
//
// If there was more than one packet, collision will be true.
func readSlot(conn *MultiCastConn, frameBegin int64, slot byte) (p *Packet, collision bool, err os.Error) {
	setProperReadTimeout(conn, frameBegin, slot)

	// Read everything until timeout is hit
	data, err := ioutil.ReadAll(conn)

	if err != nil {
		// Don't throw an error on a timeout (=> empty slot)
		if v,ok := err.(net.Error); ok && v.Timeout() {
			if len(data) == 0 {
				return nil, false, nil
			}
		} else {
			return nil, false, err
		}
	}

	buffer := bytes.NewBuffer(data)

	if len(data) < PACKET_SIZE {
		return nil, false, os.NewError(
			fmt.Sprintf("Invalid packet size: %d", len(data)))
	}

	packet, perr := NewPacketFromReader(buffer)

	if perr != nil {
		return nil, false, perr
	}

	return packet, len(data) > PACKET_SIZE, nil
}


func sendPacket(conn *MultiCastConn, payload []byte, slot byte) os.Error {
	ms := time.Nanoseconds() / 1e6

	p := NewPacket(payload, slot, ms)

	_, err := p.SendTo(conn)

	return err
}


func findFreeSlot(upcomingSlots []bool) (next byte, ok bool) {
	for i,e := range upcomingSlots {
		if !e {
			return byte(i), true
		}
	}
	return 0, false
}


func receiveLoop(source *Source, sink *Sink, conn *MultiCastConn) {
	// initially use the first slot
	nextSlot := byte(0)
	currentPayload := source.Data()

	searchNewSlot := true	// indicator that we need a new slot
	packetSent := false		// indicator that a packet was sent in the current slot

	upcomingSlots := make([]bool, SLOTS)


	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Wait for next frame to begin and start process
	syncWithNextFrame()

	for {
		frameBegin := frameBeginTime()

		mySlot := nextSlot

		// Log output
		sink.Feed("Frame begin.")
		go log.Println("Begin! Slot: ", mySlot)

		for i:= byte(0); i < SLOTS; i++ {
			go log.Println("Slot", i)

			// We have a slot and the current slot is ours, send a packet
			if i == mySlot && !searchNewSlot {
				// Determine new slot for next frame
				if t, ok := findFreeSlot(upcomingSlots); !ok {
					searchNewSlot = true

				} else {
					nextSlot = t
					packetSent = true

					// Do the send concurrently so we don't affect the read
					go func() {
						syncWithSlotCenter(frameBegin, mySlot)
						sendPacket(conn, currentPayload, nextSlot)
					}()

					go log.Println("Sent package!")
				}
			}

			packet, collision, err := readSlot(conn, frameBegin, i)

			if err != nil {
				log.Fatalf("Error while reading slot %d: %s", i, err.String())
			}

			if collision {
				go log.Println("Collision in slot", i)
			}

			// I sent a packet on my slot, no collisions, packet sent!
			if packet != nil && packetSent && i == mySlot && !collision {
				packetSent = false

				currentPayload = source.Data()
			}

			// Put every received packet in the sink
			if packet != nil {
				if 0 <= packet.Slot && packet.Slot < SLOTS {
					upcomingSlots[packet.Slot] = false
				}

				sink.Feed(fmt.Sprintf("Received on slot %d: %s", i, packet.String()))
				go log.Println("Received", packet.String())
			}

			// At the end of the frame: Search for free slot
			if searchNewSlot && i == SLOTS-1 {
				if t,ok := findFreeSlot(upcomingSlots); ok {
					nextSlot = t
					searchNewSlot = false
				}
			}

			syncWithSlotEnd(frameBegin, i)
		}
	}
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

	if cerr != nil {
		log.Fatal("JoinMulticast Error: ", cerr.String())
	}

	receiveLoop(source, sink, conn)
}
