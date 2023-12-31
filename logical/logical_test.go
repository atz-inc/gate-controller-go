package logical_test

import (
	"bufio"
	"bytes"
	"fmt"
	"gate/logical"
	"github.com/stretchr/testify/require"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

func TestNormalRead(t *testing.T) {
	ser := &mockSerial{}
	l := logical.New(ser, true)

	ser.readData = bufio.NewReader(bytes.NewBuffer([]byte{
		0xFF, // start character
		2,    // address
		4,    // message size
		//
		0x1b, // message type
		0x01,
		0x01,
		0x01,
		0xDD,

		// new msg
		0xFF, // start character
		2,    // address
		4,    // message size
		//
		0x1b, // message type
		0x01,
		0x01,
		0x01,
		0xDD,

		// new msg
		0xFF, // start character
		2,    // address
		4,    // message size
		//
		0x1b, // message type
		0x01,
		0x01,
		0x01,
		0xDD,
	}))

	packetRead := make(chan *logical.Packet)
	packetWrite := make(chan *logical.Packet)
	packetErr := make(chan error)

	go l.Start(packetRead, packetWrite, packetErr)

	for i := 0; i < 3; i++ {
		select {
		case p := <-packetRead:
			require.Equal(t, byte(0x1b), p.MessageType)
		case err := <-packetErr:
			fmt.Println("error")
			panic(err)
		}
	}

	return
}

func TestWrongMessageLength(t *testing.T) {
	ser := &mockSerial{}
	l := logical.New(ser, true)

	ser.readData = bufio.NewReader(bytes.NewBuffer([]byte{
		0xFF, // start character
		2,    // address
		4,    // message size
		0x1b, // message type
		0x01, // 1-byte message
		0xDD, //check

		//

		// normal frame to recover
		0xFF, // start character
		2,    // address
		4,    // message size
		//
		0x1b, // message type
		0x01,
		0x01,
		0x01,
		0xDD,
	}))

	packetRead := make(chan *logical.Packet)
	packetWrite := make(chan *logical.Packet)
	packetErr := make(chan error)

	go l.Start(packetRead, packetWrite, packetErr)

	for i := 0; i < 2; i++ {
		select {
		case <-packetRead:
			fmt.Println("read")
		case err := <-packetErr:
			fmt.Println("error")
			if i == 0 {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
			}
		}
	}

	return
}

func TestBadFrame(t *testing.T) {
	ser := &mockSerial{}
	l := logical.New(ser, true)

	ser.readData = bufio.NewReader(bytes.NewBuffer([]byte{
		// garbled frame
		0xFF,
		2,

		// start of good frame
		0xFF, // start character
		2,    // address
		4,    // message size
		0x1b, // message type
		0x01,
		0x01,
		0x01,
		0xDD,
	}))

	packetRead := make(chan *logical.Packet)
	packetWrite := make(chan *logical.Packet)
	packetErr := make(chan error)

	go l.Start(packetRead, packetWrite, packetErr)

	for i := 0; i < 2; i++ {
		select {
		case <-packetRead:
			fmt.Println("read")
		case err := <-packetErr:
			fmt.Println("error")
			if i == 0 {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
			}
		}
	}
	return
}

func TestMemoryLeak(t *testing.T) {

	ser := &mockSerial{}
	l := logical.New(ser, true)

	packetRead := make(chan *logical.Packet)
	packetWrite := make(chan *logical.Packet)
	packetErr := make(chan error)

	go l.Start(packetRead, packetWrite, packetErr)

	for i := 0; i < 10000; i++ {
		ser.readData = bufio.NewReader(bytes.NewBuffer([]byte{
			// start of good frame
			0xFF, // start character
			2,    // address
			4,    // message size
			0x1b, // message type
			0x01,
			0x01,
			0x01,
			0xDD,
		}))

		select {
		case <-packetRead:
			PrintMemUsage()
		}
	}

	f, err := os.Create("mem.prof")
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	defer f.Close() // error handling omitted for example
	runtime.GC()    // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
}

// todo: need to understand how device behaves here
// func TestTimeout(t *testing.T) {
// 	ser := &mockSerialTimeout{}
// 	l := logical.New(ser)

// 	ser.readData = bytes.NewBuffer([]byte{
// 		// start of valid message...
// 		0xFF,
// 		2,
// 	})
// 	_, err := l.Read()
// 	if err != nil {
// 		panic(err)
// 	}

// 	return
// }

type mockSerial struct {
	readData *bufio.Reader
}

func (s *mockSerial) Read(b []byte) (int, error) {
	return s.readData.Read(b)
}

func (s *mockSerial) Write([]byte) (int, error) {
	return 0, nil
}

func (s *mockSerial) Open() error {
	return nil
}

func (s *mockSerial) Close() error {
	return nil
}

func (l *mockSerial) ReadByte() (byte, error) {
	return l.readData.ReadByte()
}

type mockSerialTimeout struct {
	readData io.Reader
}

func (s *mockSerialTimeout) Read(b []byte) (int, error) {
	n, err := s.readData.Read(b)
	if err == io.EOF {
		// instead of EOF, sleep 3 seconds to simulate a nonresponsive io stream
		<-time.After(3 * time.Second)
	}
	return n, err
}

func (s *mockSerialTimeout) Write([]byte) (int, error) {
	return 0, nil
}

func (s *mockSerialTimeout) Close() error {
	return nil
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
