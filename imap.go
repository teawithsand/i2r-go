package i2r

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"io/ioutil"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/teawithsand/i2r/internal/util"
)

var errUnsupportedMessageType = errors.New("i2r: internal: Unsupported message type, fallback to other required")

const handlerIdLength = 16 // UUID length

var defaultHandlerID = []byte{
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
	0, 0, 0, 0,
}

type IMAPRequest struct {
	ID []byte
	// if message is instance of error
	// its handled automatically as error
	Msg interface{}
}

type IMAPResponse struct {
	ID      []byte
	IsFinal bool
	Msg     interface{}
}

type NewIMAPHandler struct {
	backgroundTaskContext context.Context
	backgroundTaskCloser  context.CancelFunc
	// Message router routes variable messages being interface{}
	router        *util.MessageRouter
	outputChannel chan<- IMAPResponse

	internalHandler internalIMAPHandler
}

func NewNewIMAPHandler(outputChannel chan<- IMAPResponse, cl *client.Client) *NewIMAPHandler {
	ctx := context.Background()
	ctx, closer := context.WithCancel(ctx)

	fallbackChannel := make(chan interface{})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case rawMsg := <-fallbackChannel:
				msg := rawMsg.(IMAPRequest)
				_ = msg

				select {
				case <-ctx.Done():
					return
				case outputChannel <- IMAPResponse{}: // TODO(teaiwthsand): here invalid ID error send to outputChannel
				}
			}
		}
	}()

	h := &NewIMAPHandler{
		backgroundTaskContext: ctx,
		backgroundTaskCloser:  closer,
		router:                util.NewMessageRouter(fallbackChannel),
		outputChannel:         outputChannel,

		internalHandler: internalIMAPHandler{
			Client: cl,
		},
	}

	h.launchBackgroundTask(defaultHandlerID, func(msgChan <-chan interface{}) {
		for {
			var msg interface{}
			select {
			case <-h.backgroundTaskContext.Done():
				return
			case msg = <-msgChan:
			}

			sendMessage := func(r IMAPResponse) {
				select {
				case h.outputChannel <- r:
				case <-h.backgroundTaskContext.Done():
					return
				}
			}

			// launch new background task for each message?
			// In fact ID regen is easy.

			_ = msg

			res, err := h.forwardSingleToInternal(h.backgroundTaskContext, msg)
			fetchRequest, ok := msg.(IMAPFetchRequest)
			if errors.Is(err, errUnsupportedMessageType) && ok {
				// launch background task using separate id for fetch messages
				h.launchBackgroundTask(fetchRequest.RespChanID, func(msgChan <-chan interface{}) {
					responseChannel := make(chan interface{})
					doneChannel := make(chan struct{})
					go func() {
						for {
							select {
							case <-doneChannel:
								return
							case res := <-responseChannel:
								_, isErr := res.(error)
								_, isFinalMsg := res.(IMAPFetchResponse)

								isFinal := isErr || isFinalMsg
								if isFinal {
									sendMessage(IMAPResponse{
										ID:      fetchRequest.RespChanID,
										Msg:     res,
										IsFinal: isFinal,
									})
								}
							}
						}
					}()
					defer func() { doneChannel <- struct{}{} }()
					h.forwardFetchToInternal(
						ctx,
						fetchRequest,
						responseChannel,
						msgChan,
					)

				})
			} else if err != nil {
				sendMessage(IMAPResponse{
					ID:  []byte(defaultHandlerID),
					Msg: err,
				})
			} else {
				sendMessage(IMAPResponse{
					ID:  []byte(defaultHandlerID),
					Msg: res,
				})
			}

			// TODO(teaiwhtsand): Separate quit handler here, which sets IsFinal to true
		}
	})

	// TODO(teawithsand): register main background handler here

	return h
}

// CancelationContext is context which should be used with outputChannel.
// It will be closed as soon as Close methods will be called on handler.
func (h *NewIMAPHandler) CancelationContext() context.Context {
	return h.backgroundTaskContext
}

// Close kills all background tasks.
// After handler is closed Handle* functions must be called no more.
func (h *NewIMAPHandler) Close() error {
	h.backgroundTaskCloser()
	return nil
}

// launchBackgroundTask creates new goroutine and registers handler.
// It registers handler before running it and unregisters it after it's done.
func (h *NewIMAPHandler) launchBackgroundTask(id []byte, handler func(<-chan interface{})) {
	c := make(chan interface{})
	h.router.RegisterSink(util.SinkID(id), c)
	go func() {
		defer h.router.DropSink(util.SinkID(id))

		handler(c)
	}()
}

func generateNewTaskID() (string, error) {
	var arr [16]byte
	_, err := io.ReadFull(rand.Reader, arr[:])
	if err != nil {
		return "", err
	}

	return string(arr[:]), nil
}

// HandleRequest handles requests and sends response to outputChannel.
// Output channel is passed during handler construction, since it allows background tasks to access it during NewIMAPHandler lifetime.
// Note: this function returns only critical errors, which prevent it from operating.
// Note #2: context passed here times out only this given request. Handler won't fail and future calls to handleRequest may be done safely
// despite the fact that error is returned from this func.
func (h *NewIMAPHandler) HandleRequest(ctx context.Context, r IMAPRequest) (err error) {
	// should the client generate query id rather than server?
	id := r.ID
	_ = id

	msg := r.Msg
	_ = msg
	return
}

// forwardSingleToInternal runs internal handling routine for specified message.
// This function handles methods which return single message in response.
func (h *NewIMAPHandler) forwardSingleToInternal(ctx context.Context, rawMsg interface{}) (res interface{}, err error) {
	var rerr error

	defer func() {
		if err == nil && rerr != nil {
			err = rerr
		}
	}()

	var mbxes []MailboxInfo
	switch msg := rawMsg.(type) {
	case IMAPListMailboxRequest:
		mbxes, err = h.internalHandler.handleListMailboxes(ctx, msg.Ref, msg.Name)
		if err != nil {
			return
		}
		res = IMAPListMailboxResponse{
			Mailboxes: mbxes,
		}
	case IMAPListSubsRequest:
		mbxes, err = h.internalHandler.handleListSubs(ctx, msg.Ref, msg.Name)
		if err != nil {
			return
		}
		res = IMAPListMailboxResponse{
			Mailboxes: mbxes,
		}
	case IMAPSubMailboxRequest:
		err = h.internalHandler.handleSubscribeMailbox(ctx, msg.MailboxName, msg.SetSub)
		res = IMAPSubMailboxResponse{}
	case IMAPCreateMailboxRequest:
		err = h.internalHandler.handleCreateMailbox(ctx, msg.Name)
		res = IMAPCreateMailboxResponse{}
	case IMAPDeleteMailboxRequest:
		err = h.internalHandler.handleDeleteMailbox(ctx, msg.Name)
		res = IMAPDeleteMailboxResponse{}
	case IMAPRenameMailboxRequest:
		err = h.internalHandler.handleRenameMailbox(ctx, msg.OldName, msg.NewName)
		res = IMAPDeleteMailboxResponse{}
	case IMAPSelectMailboxRequest:
		status, err := h.internalHandler.handleSelectMailbox(ctx, msg.Name, msg.ReadOnly)
		rerr = err
		res = IMAPSelectMailboxResponse{
			Status: status,
		}
	case IMAPExpungeRequest:
		err = h.internalHandler.handleExpunge(ctx)
		res = IMAPExpungeResponse{}
	case IMAPSearchRequest:
		data, err := h.internalHandler.handleSearchMails(ctx, msg.Criteria)
		rerr = err
		res = IMAPSearchResponse{
			SeqNums: data,
		}
	default:
		err = errUnsupportedMessageType
		return
	}
	return
}

func (h *NewIMAPHandler) forwardFetchToInternal(
	ctx context.Context,
	initMsg IMAPFetchRequest,
	responseChan chan<- interface{},
	requestChan <-chan interface{},
) {
	cl := h.internalHandler.Client

	fi := make([]imap.FetchItem, len(initMsg.Items))
	for i, it := range initMsg.Items {
		fi[i] = imap.FetchItem(it)
	}

	bsns := make([]*imap.BodySectionName, len(fi))
	for i, it := range fi {
		var err error
		bsns[i], err = imap.ParseBodySectionName(it)
		if err != nil {
			panic("NIY HANDLE PREPROCESSING ERROR HERE")
		}
	}

	dataCtx := context.Background()
	dataCtx, cancel := context.WithCancel(dataCtx)
	defer cancel()

	doneChan := make(chan error)
	msgChan := make(chan *imap.Message, 1)
	go func() {
		nss := &imap.SeqSet{}
		for _, seq := range initMsg.SS.Set {
			nss.Set = append(nss.Set, imap.Seq(seq))
		}
		// these interm* channels are required
		// since we may want to interrupt handling because client disconected,
		// which would block fetch on sending to channel
		// which would leak goroutines

		internMsgChan := make(chan *imap.Message, 1)
		internDoneChan := make(chan error)
		go func() {
			for {
				select {
				case err := <-internDoneChan:
					doneChan <- err
					return
				case msg := <-internMsgChan:
					// either:
					// 1. Drop value if function returned
					// 2. Send message to awaiting channel, since it's need there
					select {
					case msgChan <- msg:
					case <-dataCtx.Done():
					}
				}
			}
		}()
		internDoneChan <- cl.Fetch(nss, fi, internMsgChan)
	}()

	var err error

	// recv command from data channel
	// or fail with timeout
	commandOrDone := func() (interface{}, error) {
		select {
		case err := <-doneChan:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		case msg := <-requestChan:
			return msg, nil
		}
	}

	sendResponse := func(msg interface{}) bool {
		select {
		case <-ctx.Done():
			return false
		case responseChan <- msg:
		}
		return true
	}

Outer:
	for {
		select {
		case <-ctx.Done():
			sendResponse(ctx.Err())
			return
		case err = <-doneChan:
			sendResponse(err)
			return
		case recMsg := <-msgChan:
			// 0. Check if we ran out of messages
			if recMsg == nil {
				break Outer
			}

			// 1. Notify client about received data.
			rawCmd, err := commandOrDone()
			if err != nil {
				sendResponse(err)
				return
			}

		Inner:
			for {
				switch cmd := rawCmd.(type) {
				case IMAPFetchBodyRequest:
					if len(bsns) >= cmd.BodyPartNameIdx {
						// TODO(teawithsand): error here
						// sendErr(err)
						sendResponse(errors.New("Invalid body section idnex"))
						return
					}
					lit := recMsg.GetBody(bsns[cmd.BodyPartNameIdx])
					if lit == nil {
						// error here as well (TODO)
						// sendErr(err)
						sendResponse(errors.New("Nil literal returned"))
						return
					}

					if !sendResponse(IMAPFetchStreamResponse{
						Reader: ioutil.NopCloser(lit),
					}) {
						// if send response filed then return
						return
					}
				case IMAPGoToNextMailRequest:
					break Inner
				default:
					panic("report err here")
				}
			}
		}
	}
	// handling is done!
	// Notify client about that

	sendResponse(IMAPFetchResponse{})
	return
}
