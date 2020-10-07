package logical_test

import (
	"bytes"
	"gate/logical"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
	"time"
)

func TestNormalRead(t *testing.T) {
	ser := &mockSerial{}
	l := logical.New(ser)

	for i := 0; i < 3; i++ {
		ser.readData = bytes.NewBuffer([]byte{
			0xFF, // start character
			2,    // address
			4,    // message size
			//
			0x1b, // message type
			0x01,
			0x01,
			0x01,
			0xDD,
		})
		_, err := l.Read()
		if err != nil {
			panic(err)
		}
	}

	return
}

func TestWrongMessageLength(t *testing.T) {
	ser := &mockSerial{}
	l := logical.New(ser)

	ser.readData = bytes.NewBuffer([]byte{
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
	})

	for i := 0; i < 2; i++ {
		_, err := l.Read()
		if err != nil {
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
	l := logical.New(ser)

	ser.readData = bytes.NewBuffer([]byte{
		// garbaled frame
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
	})

	for i := 0; i < 2; i++ {
		_, err := l.Read()
		if err != nil {
			if i == 0 {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
			}
		}
	}
	return
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
	readData io.Reader
}

func (s *mockSerial) Read(b []byte) (int, error) {
	return s.readData.Read(b)
}

func (s *mockSerial) Write([]byte) (int, error) {
	return 0, nil
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