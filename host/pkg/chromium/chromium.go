package chromium

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
)

type Chromium struct {
	logger *log.Logger
}

type Message struct {
	Type    string //command, response
	Payload interface{}
}

func New(logger *log.Logger) *Chromium {
	return &Chromium{logger: logger}
}

/* Start listening messages from the extension */
func (c *Chromium) StartReading(handler func(msg *Message)) {

	for {
		s := bufio.NewReader(os.Stdin)
		length := make([]byte, 4)
		_, _ = s.Read(length)
		lengthNum := readMessageLength(length)
		content := make([]byte, lengthNum)
		_, _ = s.Read(content)

		incomingMessage, err := unmarshal(content)

		if err != nil {
			continue
		}

		if incomingMessage == nil {
			continue
		}

		c.logger.Printf("Received raw message: %s", content)

		// pass the message to the handler
		go handler(incomingMessage)
	}
}

func (c *Chromium) Send(msg Message) {
	byteMsg := encodeMessage(msg)
	var msgBuf bytes.Buffer
	writeMessageLength(byteMsg)
	msgBuf.Write(byteMsg)
	_, _ = msgBuf.WriteTo(os.Stdout)
}

func readMessageLength(msg []byte) int {
	var length uint32
	buf := bytes.NewBuffer(msg)
	_ = binary.Read(buf, binary.LittleEndian, &length)
	return int(length)
}

func encodeMessage(msg Message) []byte {
	return dataToBytes(msg)
}

func dataToBytes(msg Message) []byte {
	byteMsg, _ := json.Marshal(msg)
	return byteMsg
}

func writeMessageLength(msg []byte) {
	_ = binary.Write(os.Stdout, binary.LittleEndian, uint32(len(msg)))
}

func unmarshal(msg []byte) (*Message, error) {
	var m Message
	err := json.Unmarshal(msg, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
