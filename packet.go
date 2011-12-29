package main

import (
	"io"
	"os"
	"fmt"
	"bytes"
	"encoding/binary"
)

// Packet size in byte
const PACKET_SIZE = 33

type Packet struct {
	Payload	[24]byte
	Slot	byte
	Time	int64
}


func NewPacket(payload []byte, slot byte, time int64) *Packet {
	p := &Packet{[24]byte{}, slot, time}

	copy(p.Payload[:], payload)

	return p
}


// Interface for NewPacketFromReader
type byteReader interface {
	io.Reader
	io.ByteReader
}


// Try to read a packet from a reader (e.g. a UDP connection)
func NewPacketFromReader(r byteReader) (*Packet, os.Error) {
	packet := new(Packet)
	payloadBuffer := make([]byte, 24)

	n, err := r.Read(payloadBuffer)

	if n != 24 {
		return nil, os.NewError(fmt.Sprintf("Payload too short: %d", n))
	}

	if err != nil {
		return nil, err
	}

	copy(packet.Payload[:], payloadBuffer)

	packet.Slot, err = r.ReadByte()

	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &packet.Time)

	if err != nil {
		return nil, err
	}

	return packet, nil
}


func (p *Packet) String() string {
	return fmt.Sprintf("Packet{Payload: %s, Slot: %d, Time: %d}\n",
					   p.Payload, p.Slot, p.Time)
}


func (p *Packet) SendTo(w io.Writer) (int64, os.Error) {
	pbuf := bytes.NewBufferString("")

	// Fill output buffer
	pbuf.Write(p.Payload[:])
	pbuf.WriteByte(p.Slot)
	binary.Write(pbuf, binary.BigEndian, p.Time)

	return pbuf.WriteTo(w)
}

