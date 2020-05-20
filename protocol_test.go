package i2r_test

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/teawithsand/i2r"
	"github.com/teawithsand/rgz"
)

func newIMPC() (l, r *IMPC) {
	c1 := make(chan []byte, 1)
	c2 := make(chan []byte, 1)
	l = &IMPC{
		rc: c1,
		sc: c2,
	}
	r = &IMPC{
		rc: c2,
		sc: c1,
	}
	return
}

type IMPC struct {
	rc chan []byte
	sc chan []byte
}

func (p *IMPC) WritePacket(ctx context.Context, data []byte) (err error) {
	select {
	case p.sc <- data:
	case <-ctx.Done():
		err = ctx.Err()
	}
	return
}

func (p *IMPC) ReadPacked(ctx context.Context, appendTo []byte) (res []byte, err error) {
	select {
	case res = <-p.rc:
	case <-ctx.Done():
		err = ctx.Err()
	}
	return
}

// NOP closer
func (p *IMPC) Close() error {
	return nil
}

func newPacketConns() (c1, c2 i2r.ProtocolConnection) {
	p1, p2 := newIMPC()
	m := rgz.BasicPolyMarshaler(rgz.MarshalerFunc(json.Marshal), i2r.MsgTypeRegistry)
	um := rgz.BasicPolyUnmarshaler(rgz.UnmarshalerFunc(json.Unmarshal), i2r.MsgTypeRegistry)

	c1 = &i2r.DefaultProtocolConnection{
		PacketConnection: p1,
		Marshaler:        m,
		Unmarshaler:      um,
	}
	c2 = &i2r.DefaultProtocolConnection{
		PacketConnection: p2,
		Marshaler:        m,
		Unmarshaler:      um,
	}

	return
}

func TestProtocolCanSendAndReceive(t *testing.T) {
	c1, c2 := newPacketConns()
	wg := &sync.WaitGroup{}
	wg.Add(2)
	ctx := context.Background()
	id := []byte{0xab, 0xcd, 0xef, 0x12}
	go func() {
		err := c1.Write(ctx, i2r.IMAPNoopRequest{}, id)
		if err != nil {
			t.Error(err)
		}
		wg.Done()
	}()
	go func() {
		msg, rid, err := c2.Read(ctx)
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(rid[:], id[:]) != 0 {
			t.Error("IDs not equal")
		}
		_, ok := msg.(*i2r.IMAPNoopRequest) // returned types are always poitner types
		if !ok {
			t.Error("Msg is not imap noop request ")
		}

		wg.Done()
	}()
	wg.Wait()
}
