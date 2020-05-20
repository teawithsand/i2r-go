package i2r

import (
	"reflect"

	"github.com/teawithsand/rgz"
)

// MsgTypeRegistry registers all requests and responses
// and can be used in order to create polymorphic marshalers for messages sent to clients.
// Note: some messages are not registered, because they require custom marshaling
var MsgTypeRegistry = rgz.NewTypeRegistry()

// ACL for MsgTypeRegistry.
func registerMessageType(ty reflect.Type, tag rgz.Tag) {
	MsgTypeRegistry.RegisterType(ty, tag)
}

func init() {
	// TODO(teawithsand): make these tags indexes constant across compilations
	// right now it can be changed too easily

	i := rgz.Tag(0)
	r := func(f interface{}) {
		registerMessageType(reflect.TypeOf(f), i)
		i++
	}

	r(IMAPListMailboxRequest{})
	r(IMAPListMailboxResponse{})

	r(IMAPListSubsRequest{})
	r(IMAPListSubsResponse{})

	r(IMAPSubMailboxRequest{})
	r(IMAPSubMailboxResponse{})

	r(IMAPCreateMailboxRequest{})
	r(IMAPCreateMailboxResponse{})

	r(IMAPDeleteMailboxRequest{})
	r(IMAPDeleteMailboxResponse{})

	r(IMAPRenameMailboxRequest{})
	r(IMAPRenameMailboxResponse{})

	r(IMAPNoopRequest{})
	r(IMAPNoopResponse{})

	r(IMAPQuitRequest{})
	r(IMAPQuitResponse{})

	r(IMAPSelectMailboxRequest{})
	r(IMAPSelectMailboxResponse{})

	r(IMAPExpungeRequest{})
	r(IMAPExpungeResponse{})

	r(IMAPSearchRequest{})
	r(IMAPSearchResponse{})

	r(IMAPStoreRequest{})
	r(IMAPStoreResponse{})

	r(IMAPCheckRequest{})
	r(IMAPCheckResponse{})

	r(IMAPFetchRequest{})

	// this one has custom marshaling
	// as this is stream one
	// r(IMAPFetchStreamResponse{})

	r(IMAPFetchStreamResponsePart{})
	r(IMAPFetchGotMessageResponse{})
	r(IMAPFetchBodyRequest{})
	r(IMAPGoToNextMailRequest{})

	r(TextError{})
}
