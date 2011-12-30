package main

import (
	"net"
	"log"
	"os"
	"bytes"
)

type MultiCastConn struct {
	*net.UDPConn

	addr	*net.UDPAddr
}

func (m *MultiCastConn) Close() {
	m.LeaveGroup(m.addr.IP)
	m.UDPConn.Close()
}

func (m *MultiCastConn) Write(val []byte) (int, os.Error) {
	return m.WriteTo(val, m.addr)
}

// Create a new multicast connection joining the given network
func JoinMulticast(host, port string) (*MultiCastConn, os.Error) {
		a := net.JoinHostPort(host, port)

        ua, e := net.ResolveUDPAddr("udp4", a)
        if e != nil {
                return nil, e
        }
        addr := &net.UDPAddr{
                IP:   net.IPv4zero,
                Port: ua.Port,
        }
        c, e := net.ListenUDP("udp4", addr)
        if e != nil {
                return nil, e
        }
        e = c.JoinGroup(ua.IP)
        if e != nil {
                return nil, e
        }
        log.Println("Joined to multicast IP", ua.IP, "on port", ua.Port)
        return &MultiCastConn{c, ua}, nil
}


func Listener(conn *MultiCastConn, sync chan bool) {
	defer func() {
		sync <- true
	}()
	sync <- true

	byteSlice := make([]byte, 1024)
	buffer := bytes.NewBuffer(byteSlice)

	// 50ms for each slot
	conn.SetReadTimeout(50 * 10e6)

	for {
		n, err := conn.Read(byteSlice)

		if err != nil {
			log.Fatal(err.String())
		}

		buffer.Write(byteSlice)

		if n != PACKET_SIZE {
			continue
		}

		log.Println("Enough bytes... Creating packet instance!")

		packet, perr := NewPacketFromReader(buffer)

		if perr == nil {
			log.Println(conn, "Received:", packet)
		} else {
			log.Println(conn, "Packet error:", perr)
		}

		// buffer.Reset()

		return
	}
}


func _main() {
	conn1, err1 := JoinMulticast("225.10.1.2", "15017")

	if err1 != nil {
		log.Fatal(err1.String())
	}

	conn2, err2 := JoinMulticast("225.10.1.2", "15017")

	if err2 != nil {
		log.Fatal(err2.String())
	}

	sync := make(chan bool)

	go Listener(conn1, sync)
	go Listener(conn2, sync)

	<-sync
	<-sync

	log.Println("Sending...")

	pack := NewPacket([]byte("foobar"), 0, 1)

	n, err := pack.SendTo(conn1)

	if err != nil {
		log.Fatal(err.String())
	}

	log.Println("Sent", n, "Bytes")

	// Wait for listeners before close
	<-sync
	<-sync

	log.Println("Closing connection...")

	conn1.Close()
	conn2.Close()


}
