package main

import (
	"net"
	"log"
	"os"
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
