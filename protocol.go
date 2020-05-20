package i2r

import (
	"context"
	"errors"
	"io"

	"github.com/teawithsand/rgz"
)

var ErrInvalidID = errors.New("i2r: Given streamID is invalid and can't be transmitted through ProtocolConnection")
var ErrInvalidPacket = errors.New("i2r: Givne packet received from PacketConetion is not valid")

// ProtocolConnection handles three things:
// 1. Marshaling and sending messages
// 2. Receiving and unmarshaling messages
type ProtocolConnection interface {
	io.Closer
	Write(ctx context.Context, msg interface{}, streamID []byte) error
	Read(ctx context.Context) (interface{}, []byte, error)
}

// PacketConnection is any connection capable of per-packet communication(like websocket).
// It should be implemented by client in order to provide custom communication protocol.
// By default it's implemented for gorilla websocket.
type PacketConnection interface {
	WritePacket(ctx context.Context, data []byte) (err error)
	ReadPacked(ctx context.Context, appendTo []byte) (res []byte, err error)
}

// DefaultProtocolConnection wraps PacketConnection and creates ProtocolConnection from it.
type DefaultProtocolConnection struct {
	PacketConnection PacketConnection
	Marshaler        rgz.PolyMarshaler
	Unmarshaler      rgz.PolyUnmarshaler
}

// DPC requires no cleanup
func (dpc *DefaultProtocolConnection) Close() error {
	return nil
}

func (dpc *DefaultProtocolConnection) Write(ctx context.Context, msg interface{}, streamID []byte) error {
	idLen := len(streamID)
	if idLen >= 255 { // 255 is reserved
		// Too long id
		return ErrInvalidID
	}

	data, err := dpc.Marshaler.PolyMarshal(msg)
	if err != nil {
		return err
	}
	res := make([]byte, len(data)+len(streamID)+1)
	res[0] = uint8(idLen)
	copy(res[1:1+len(streamID)], streamID[:])
	copy(res[1+len(streamID):], data[:])

	// TODO(teawithsand): implement memmove based approach
	/*
		data = append(data, 0, 0)
		for i := len(data) - (1 + 2); i >= 0; i++ {
			data[i+2] = data[i]
		}
		data[0] = 1
		data[1] = uint8(idLen)
	*/

	return dpc.PacketConnection.WritePacket(ctx, res)
}

func (dpc *DefaultProtocolConnection) Read(ctx context.Context) (msg interface{}, streamID []byte, err error) {
	packet, err := dpc.PacketConnection.ReadPacked(ctx, nil)
	if err != nil {
		return
	}
	// TODO(teawithsand): fuzztest it OR do complete walk over all possible buggy data lengths
	if len(packet) < 1 || len(packet) < 1+int(packet[0]) {
		err = ErrInvalidPacket
		return
	}
	streamID = packet[1 : 1+int(packet[0])]
	data := packet[1+int(packet[0]):]
	msg, err = dpc.Unmarshaler.PolyUnmarshal(data)
	return
}

/*
type messageCode uint32

var registeredMessages = map[reflect.Type]messageCode{}
var reverseRegisteredMessages = map[messageCode]reflect.Type{}

func serializeMessage(msg interface{}) (id messageCode, data []byte, err error) {
	id, ok := registeredMessages[reflect.TypeOf(msg)]
	if !ok {
		// report error here
	}
	data, err = json.Marshal(msg)
	return
}

func deserializeMessage(id messageCode, data []byte) (msg interface{}, err error) {
	ty, ok := reverseRegisteredMessages[id]
	if !ok {
		// report error
	}
	msg = reflect.New(ty)
	err = json.Unmarshal(data, msg)
	return
}

// DefaultProtocolConnection wraps PacketConnection and provides ProtocolConnection, which
// may be used to transport data.
type DefaultProtocolConnection struct {
	PacketConnection PacketConnection
}

type defaultProtoConnMsg struct {
	ID  []byte      `json:"id"`
	Msg interface{} `json:"msg"`
}

func (dpc *DefaultProtocolConnection) Write(ctx context.Context, msg interface{}, streamID []byte) error {
	if streamMsg, ok := msg.(IMAPFetchStreamResponse); ok {
		defer streamMsg.Reader.Close()

		_ = streamMsg
		panic("NIY IMPLEMENT STREAM MESSAGE MARSHALING ")
	}
	mmsg := defaultProtoConnMsg{
		ID:  streamID,
		Msg: msg,
	}
	data, err := json.Marshal(mmsg)
	if err != nil {
		return err
	}
	w := dpc.PacketConnection.WritePacket(ctx)
	defer w.Close()
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return nil
}

// note: Read reads only messages from client.

func (dpc *DefaultProtocolConnection) Read(ctx context.Context) (msg interface{}, streamID []byte, err error) {
	r := dpc.PacketConnection.ReadPacket(ctx)
	defer r.Close()

	buffer, err := ioutil.ReadAll(io.LimitReader(r, 1024*1024*4)) // constatnt limit of 4MB per from user message(should packet conn limit that?)
	if err != nil {
		return
	}

	if len(buffer) < 4 {
		panic("Err invalid sz")
	}

	idSize := int(binary.BigEndian.Uint16(buffer[:2]))
	buffer = buffer[2:]
	if len(buffer) < idSize {
		panic("Err invalid sz")
	}
	streamID = buffer[:idSize]
	buffer = buffer[idSize:]

	if len(buffer) < 4 {
		panic("Err invalid sz")
	}

	msgTag := buffer[:4]
	_ = msgTag
	buffer = buffer[4:]

	err = r.Close()
	if err != nil {
		return
	}
	return
}
*/
