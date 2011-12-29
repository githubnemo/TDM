package main

import (
	"os"
	"fmt"
	"tdm"
	"time"
	"log"
	"bytes"
	"strconv"
)

const (
	// Time per frame in seconds
	FRAME_TIME = 1

	// Number of slots per frame
	SLOTS = 20

	// Time per slot in nanoseconds
	SLOT_TIME = int64((float64(FRAME_TIME) / SLOTS) * 10e9)
)

// milli => 10e3
// micro => 10e6
// nano  => 10e9

func syncWithNextFrame() {
	cns := time.Nanoseconds()

	time.Sleep((cns / 10e9 + 1) * 10e9 - cns)
}

func syncWithSlotCenter(slot byte) {
	ns := int64(slot) * SLOT_TIME + SLOT_TIME/2

	time.Sleep(ns)
}

func syncWithSlotEnd(slot byte) {
	ns := int64(slot) * SLOT_TIME + SLOT_TIME

	time.Sleep(ns)
}


func readSlot(conn *MultiCastConn) (*Packet, bool, os.Error) {
	byteSlice := make([]byte, PACKET_SIZE)
	buffer := bytes.NewBuffer(byteSlice)

	n, err := conn.Read(byteSlice)

	if err != nil {
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


func sendPacket(conn *MultiCastConn, payload string, slot byte) os.Error {
	ms := time.Nanoseconds() / 10e6

	p := NewPacket([]byte(payload), slot, ms)

	_, err := p.SendTo(conn)

	return err
}


func main() {
	team, station := 17, 1
	logPath := "/home/nemo/Code/tdm/go/log/"

	source := tdm.NewSource(team, station)

	sink, serr := tdm.NewSink(team, station, logPath)

	if serr != nil {
		fmt.Println(serr.String())
		return
	}

	sink.Start()
	defer sink.Stop()

	// Open network connection
	ip := "225.10.1.2"
	port := 15000

	conn, cerr := JoinMulticast(ip, strconv.Itoa(port + team))

	// Limit the max. read time to the slot time
	conn.SetReadTimeout(SLOT_TIME)

	if cerr != nil {
		log.Fatal("Network join error:", cerr.String())
	}

	// initially use the first slot
	mySlot := byte(0)
	currentPayload := source.Data()
	searchNewSlot := false

	for {
		syncWithNextFrame()

		for i:= byte(0); i < SLOTS; i++ {
			// We have a good slot and this is our slot, send!
			if i == mySlot && !searchNewSlot {
				syncWithSlotCenter(mySlot)
				sendPacket(conn, currentPayload, mySlot)
			}

			packet, empty, err := readSlot(conn)

			if err != nil {
				log.Fatalf("Error while reading slot %d: %s", i, err.String())
			}

			// Empty slot and in need of one? Take it!
			if empty && searchNewSlot {
				mySlot = i
			}

			stringPayload := string(packet.Payload[:])

			// Detect collision
			if(i == mySlot && stringPayload != currentPayload) {
				searchNewSlot = true
				sink.Feed(fmt.Sprintf("Collision in slot %d!", i))
			} else {
				currentPayload = source.Data()
			}

			sink.Feed(stringPayload)

			syncWithSlotEnd(i)
		}
	}
}
