package main

import "fmt"
import "testing"

func TestSource(t *testing.T) {
	s := NewSource(123, 234)

	for i := 0; i < 100; i++ {
		d := s.Data()

		if false {
			fmt.Println("output:",string(d))
		}

		if len(d) > 24 {
			t.Fatal("len(d) > 24:", string(d))
		}
	}

}
