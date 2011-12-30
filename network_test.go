package main

import "testing"

func TestConnection(t *testing.T) {
	conn1, err1 := JoinMulticast("225.10.1.2", "15017")

	if err1 != nil {
		t.Fatal(err1.String())
	}

	conn2, err2 := JoinMulticast("225.10.1.2", "15017")

	if err2 != nil {
		t.Fatal(err2.String())
	}

	sync := make(chan bool)

	go Listener(conn1, sync)
	go Listener(conn2, sync)

	<-sync
	<-sync

	t.Log("Sending...")

	pack := NewPacket([]byte("foobar"), 0, 1)

	n, err := pack.SendTo(conn1)

	if err != nil {
		t.Fatal(err.String())
	}

	t.Log("Sent", n, "Bytes")

	// Wait for listeners before close
	<-sync
	<-sync

	t.Log("Closing connection...")

	conn1.Close()
	conn2.Close()
}
