package main

import (
	"os"
	"fmt"
	"net"
	"log"
	"flag"
	"time"
	"rand"
	"bytes"
	"strconv"
	"io/ioutil"
)

const (
	TEAM_NUMBER = 17

	// Time per frame in nanoseconds
	FRAME_TIME = 1 * 1e9

	// Number of slots per frame
	SLOTS = 20

	// Time per slot in nanoseconds
	SLOT_TIME = int64(FRAME_TIME / SLOTS)
)


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


// Reads all bytes from the connection in a time interval set by
// setProperReadTimeout().
//
// If there was more than one packet, collision will be true and the
// first packet arrived will be returned. In case nothing was received
// at all, empty will be true and packet will be nil.
func readSlot(conn *MultiCastConn, frameBegin int64, slot byte) (p *Packet, empty bool, collision bool, err os.Error) {
	setProperReadTimeout(conn, frameBegin, slot)

	// Read everything until timeout is hit
	data, err := ioutil.ReadAll(conn)

	if err != nil {
		// Don't throw an error on a timeout (=> empty slot)
		if v,ok := err.(net.Error); ok && v.Timeout() {
			if len(data) == 0 {
				return nil, true, false, nil
			}
		} else {
			return nil, false, false, err
		}
	}

	buffer := bytes.NewBuffer(data)

	if len(data) < PACKET_SIZE {
		return nil, false, false, os.NewError(
			fmt.Sprintf("Invalid packet size: %d", len(data)))
	}

	packet, perr := NewPacketFromReader(buffer)

	if perr != nil {
		return nil, false, false, perr
	}

	return packet, false, len(data) > PACKET_SIZE, nil
}


func sendPacket(conn *MultiCastConn, payload []byte, slot byte) os.Error {
	ms := time.Nanoseconds() / 1e6

	p := NewPacket(payload, slot, ms)

	_, err := p.SendTo(conn)

	return err
}


func findFreeSlot(occupiedSlots []int, currentFrame int) (next byte, ok bool) {
	slots := make([]int, len(occupiedSlots))
	for i,e := range occupiedSlots {
		if e == 0 || (currentFrame - e) > 1 {
			slots = append(slots, i)
		}
	}

	if len(slots) == 0 {
		return 0, false
	}

	return byte(slots[rand.Intn(len(slots))]), true
}


func receiveLoop(source *Source, sink *Sink, conn *MultiCastConn) {
	nextSlot := byte(0)
	currentPayload := source.Data()

	searchNewSlot := true	// indicator that we need a new slot
	occupiedSlots := make([]int, SLOTS)

	// Wait for next frame to begin and start process
	syncWithNextFrame()

	for frameNr := 1; ; frameNr++ {
		frameBegin := frameBeginTime()
		mySlot := nextSlot

		go log.Println("Begin! Slot: ", mySlot, searchNewSlot)

		for i:= byte(0); i < SLOTS; i++ {
			go log.Println("Slot", i)

			// No packet was sent in this slot yet
			packetSent := false

			// We have a slot and the current slot is ours, send a packet
			// with the next slot number we want to occupy.
			if i == mySlot && !searchNewSlot {
				// Determine new slot for next frame
				if t, ok := findFreeSlot(occupiedSlots, frameNr); !ok {
					searchNewSlot = true
					go log.Println("No free slot! Won't send.")

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

			packet, empty, collision, err := readSlot(conn, frameBegin, i)

			if err != nil {
				log.Fatalf("Error while reading slot %d: %s", i, err.String())
			}

			// Collision detected, set this slot to occupied and by 50% chance
			// start a search for a free slot. The randomness is an attempt to
			// prevent both senders to search again.
			if collision && packetSent {
				go log.Println("Collision in slot", i)

				occupiedSlots[i] = frameNr

				searchNewSlot = true
			}

			// Mark slot as empty if it's occupation is older than one frame
			if empty && occupiedSlots[i]+1 < frameNr {
				occupiedSlots[i] = 0
			}

			// I sent a packet on my slot, no collisions, packet sent,
			// calculate new payload.
			if packet != nil && packetSent && i == mySlot && !collision {
				packetSent = false
				currentPayload = source.Data()
			}

			// Put every received packet in the sink and register the slot
			// mentioned in the packet as occupied with this frame as origin
			if packet != nil {
				if 0 <= packet.Slot && packet.Slot < SLOTS {
					occupiedSlots[packet.Slot] = frameNr
				}

				sink.Feed(fmt.Sprintf("%d: Received on slot %d: %s", time.Nanoseconds() / 1e6, i, packet.String()))
				go log.Println("Received", packet.String())
			}

			// At the end of the frame: Search for free slot if needed
			if searchNewSlot && i == SLOTS-1 {
				if t,ok := findFreeSlot(occupiedSlots, frameNr); ok {
					nextSlot = t
					searchNewSlot = false
				}
			}

			syncWithSlotEnd(frameBegin, i)
		}
	}
}


// Command line parameters
var g_station *int = flag.Int("station", 1, "Station number")
var g_port *int = flag.Int("port", 15017, "Port to listen on")
var g_ip *string = flag.String("ip", "225.10.1.2", "Multicast address to listen on")
var g_logDir *string = flag.String("logdir", "../log/", "Log directory")


func main() {
	flag.Parse()

	station := *g_station
	logPath := *g_logDir

	// Setup source & sink
	source := NewSource(TEAM_NUMBER, station)
	sink, serr := NewSink(TEAM_NUMBER, station, logPath)

	if serr != nil {
		log.Fatal(serr)
	}

	sink.Start()

	defer sink.Stop()

	// Open network connection
	ip := *g_ip
	port := *g_port

	conn, cerr := JoinMulticast(ip, strconv.Itoa(port))

	if cerr != nil {
		log.Fatal("JoinMulticast Error: ", cerr.String())
	}

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix(fmt.Sprintf("Station %d: ", station))

	// Reseed the PRNG for each station
	rand.Seed(int64(station) * 1e6)

	receiveLoop(source, sink, conn)
}
