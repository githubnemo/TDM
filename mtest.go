package main

import (
	"net"
	"fmt"
	"log"
	"os"
)

type MultiCastConn struct {
	conn	*net.UDPConn
	addr	*net.UDPAddr
}

func (m *MultiCastConn) Close() {
	m.conn.Close()
}

func (m *MultiCastConn) Write(val []byte) (int, os.Error) {
	return m.conn.WriteTo(val, m.addr)
}

func (m *MultiCastConn) Read(val []byte) (int, os.Error) {
	return m.conn.Read(val)
}


func JoinMulticast(a string) (*MultiCastConn, os.Error) {
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
        fmt.Println("Joined to multicast IP", ua.IP, "on port", ua.Port)
        return &MultiCastConn{c, ua}, nil
}


func Listener(conn *MultiCastConn) {
	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)

		if err != nil {
			log.Fatal(err.String())
		}

		fmt.Println(conn, "Received:", string(buffer[:n]))
	}
}


func main() {

	addr := "225.10.1.2:15017"

	conn1, err1 := JoinMulticast(addr)

	if err1 != nil {
		log.Fatal(err1.String())
	}

	conn2, err2 := JoinMulticast(addr)

	if err2 != nil {
		log.Fatal(err2.String())
	}

	go Listener(conn1)
	go Listener(conn2)

	conn1.Write([]byte("foobar"))

	conn1.Close()
	conn2.Close()
}
