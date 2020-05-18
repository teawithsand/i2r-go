package i2r

/*
// Connection abstracts any kind of connection which may be used to communicate with clients.
type Connection interface {
	Read() ([]byte, error)
	Write(data []byte) error
}

// ConnID allows transfering multiple data streams over single connection using multiplexing.
type ConnID []byte

func NewConnID() ConnID {
	var arr [16]byte
	_, err := io.ReadFull(rand.Reader, arr[:])
	if err != nil {
		panic(err)
	}
	return ConnID(arr[:])
}

// StructConn is connection able to send and receive data structures over/from connection.
// It also has support for multiplexing. Each message connection has it's id.
type StructConn interface {
	io.Closer

	ReadMessage() (id ConnID, msg interface{}, err error)
	WriteMessage(id ConnID, msg interface{}) (err error)
}
*/

// TODO(teawithsand): add websocket adapter here which creates connection from WS connection
// and structured conn implementations for IMAP and SMTP messaging.

// ConnManager handles things like sending/receiving data on channel with specified id.
// It also handles (de)serialization.
type ConnManager interface {
	WriteMessage(cid []byte, msg interface{}) (err error)
	ReadMessage() (id []byte, msg interface{}, err error)
}
