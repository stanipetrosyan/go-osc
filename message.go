package osc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
)

// Message represents a single OSC message. An OSC message consists of an OSC
// address pattern and zero or more arguments.
type Message struct {
	Address   string
	Arguments []interface{}
}

// Verify that Messages implements the Packet interface.
// var _ Packet = (*Message)(nil)

// Append appends the given arguments to the arguments list.
func (msg *Message) Append(args ...interface{}) {
	msg.Arguments = append(msg.Arguments, args...)
}

// Equals returns true if the given OSC Message `m` is equal to the current OSC
// Message. It checks if the OSC address and the arguments are equal. Returns
// true if the current object and `m` are equal.
func (msg *Message) Equals(m *Message) bool {
	return reflect.DeepEqual(msg, m)
}

// Clear clears the OSC address and all arguments.
func (msg *Message) Clear() {
	msg.Address = ""
	msg.ClearData()
}

// ClearData removes all arguments from the OSC Message.
func (msg *Message) ClearData() {
	msg.Arguments = msg.Arguments[len(msg.Arguments):]
}

// Match returns true, if the OSC address pattern of the OSC Message matches the given
// address. The match is case sensitive!
func (msg *Message) Match(addr string) bool {
	return getRegEx(msg.Address).MatchString(addr)
}

// typeTags returns the type tag string.
func (msg *Message) typeTags() string {
	if len(msg.Arguments) == 0 {
		return ","
	}

	var tags strings.Builder
	_ = tags.WriteByte(',')

	for _, m := range msg.Arguments {
		tags.WriteByte(getTypeTag(m))
	}

	return tags.String()
}

// String implements the fmt.Stringer interface.
func (msg *Message) String() string {
	if msg == nil {
		return ""
	}

	var s strings.Builder
	tags := msg.typeTags()
	s.WriteString(fmt.Sprintf("%s %s", msg.Address, tags))

	for _, arg := range msg.Arguments {
		switch argType := arg.(type) {
		case bool, int32, int64, float32, float64, string:
			s.WriteString(fmt.Sprintf(" %v", argType))

		case nil:
			s.WriteString(" Nil")

		case []byte:
			s.WriteString(fmt.Sprintf(" %s", argType))

		case Timetag:

			s.WriteString(fmt.Sprintf(" %d", Timetag(argType)))
		}
	}

	return s.String()
}

// MarshalBinary serializes the OSC message to a byte buffer. The byte buffer
// has the following format:
// 1. OSC Address Pattern
// 2. OSC Type Tag String
// 3. OSC Arguments.
func (msg *Message) MarshalBinary() ([]byte, error) {
	// We can start with the OSC address and add it to the buffer
	data := new(bytes.Buffer)

	_, err := writePaddedString(msg.Address, data)
	if err != nil {
		return nil, err
	}

	// Type tag string starts with ","
	lenArgs := len(msg.Arguments)
	typetags := make([]byte, lenArgs+1)
	typetags[0] = ','

	// Process the type tags and collect all arguments
	payload := new(bytes.Buffer)

	for i, arg := range msg.Arguments {
		switch t := arg.(type) {
		case bool:
			if t {
				typetags[i+1] = 'T'
				continue
			}

			typetags[i+1] = 'F'

		case nil:
			typetags[i+1] = 'N'

		case int32:
			typetags[i+1] = 'i'

			err = binary.Write(payload, binary.BigEndian, t)
			if err != nil {
				return nil, err
			}

		case float32:
			typetags[i+1] = 'f'

			err := binary.Write(payload, binary.BigEndian, t)
			if err != nil {
				return nil, err
			}

		case string:
			typetags[i+1] = 's'

			_, err = writePaddedString(t, payload)
			if err != nil {
				return nil, err
			}

		case []byte:
			typetags[i+1] = 'b'

			_, err = writeBlob(t, payload)
			if err != nil {
				return nil, err
			}

		case int64:
			typetags[i+1] = 'h'

			err = binary.Write(payload, binary.BigEndian, t)
			if err != nil {
				return nil, err
			}

		case float64:
			typetags[i+1] = 'd'

			err = binary.Write(payload, binary.BigEndian, t)
			if err != nil {
				return nil, err
			}

		case Timetag:
			typetags[i+1] = 't'

			b, err := t.MarshalBinary()
			if err != nil {
				return nil, err
			}

			_, err = payload.Write(b)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported type: %T", t)
		}
	}

	// Write the type tag string to the data buffer
	if _, err := writePaddedString(string(typetags), data); err != nil {
		return nil, err
	}

	// Write the payload (OSC arguments) to the data buffer
	if _, err := data.Write(payload.Bytes()); err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

// NewMessage returns a new Message. The address parameter is the OSC address.
func NewMessage(addr string, args ...interface{}) *Message {
	return &Message{Address: addr, Arguments: args}
}
