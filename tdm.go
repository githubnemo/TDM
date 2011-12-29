package main

import (
	"fmt"
	"tdm"
	"time"
)

const (
	FRAME_TIME = 1 // seconds
	SLOTS = 20
	SLOT_TIME = float64(FRAME_TIME) / SLOTS // milliseconds
	PACKET_SIZE = 33 // byte
)

type Packet struct {
	payload [24]byte	// user data
	slot	byte	// slot index from 0-19
	time	int64	// send time encoded in big endian
}

// milli => 10e3
// micro => 10e6
// nano  => 10e9

func syncWithNextFrame() {
	cns := time.Nanoseconds()

	time.Sleep((cns / 10e9 + 1) * 10e9 - cns)
}

func syncWithSlotCenter(slot byte) {
	ns := (slot * SLOT_TIME + SLOT_TIME/2) * 10e9

	time.Sleep(ns)
}

// Wait for next frame, listen the whole frame for occupied slots
func lookForNewSlot() {

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

	sink.Feed(source.Data())
	sink.Feed(source.Data())
	sink.Feed(source.Data())

	// TODO: open network connection

	// initially use the first slot
	currentSlot := 0
	currentPayload := source.Data()

	for {
		syncWithNextFrame()

		syncWithSlotCenter(currentSlot)

		sendPacket(currentSlot, currentPayload)

		if(detectCollision(currentSlot, currentPayload)) {
			// watch a full frame and determine which slot could be free
			currentSlot = lookForNewSlot()
		} else {
			// fetch new data for the next run
			currentPayload = source.Data()
		}
	}


	/*

   	frametime = 1 second
	slottime = frametime / 20

	TODO: receiving of messages

    1. Open network connection on 15000 + team
	loop {
		1. Sync with next frame (wait till next second)
		2. Build packet (initially assume slot 0)
		3. Sync to slot middle (wait slottime/2 seconds)
		4. Send packet
		5. Listen frame (frametime - (slotnumber * slottime) seconds)
		6. Determine new slot if necessary (collision detected)
	}

	*/
}
