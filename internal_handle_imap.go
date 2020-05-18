package i2r

import (
	"context"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type internalIMAPHandler struct {
	Client *client.Client
}

func (h *internalIMAPHandler) handleListMailboxes(ctx context.Context, ref, name string) (mbxes []MailboxInfo, err error) {
	resChan := make(chan *imap.MailboxInfo)
	doneChan := make(chan error)
	go func() {
		doneChan <- h.Client.List(ref, name, resChan)
	}()

Outer:
	for {
		select {
		case err := <-doneChan:
			return mbxes, err
			break Outer
		case c := <-resChan:
			if c != nil {
				mbxes = append(mbxes, MailboxInfo(*c))
			}
		}
	}
	return
}

func (h *internalIMAPHandler) handleListSubs(ctx context.Context, ref, name string) (mbxes []MailboxInfo, err error) {
	resChan := make(chan *imap.MailboxInfo)
	doneChan := make(chan error)
	go func() {
		doneChan <- h.Client.Lsub(ref, name, resChan)
	}()

Outer:
	for {
		select {
		case err := <-doneChan:
			return mbxes, err
			break Outer
		case c := <-resChan:
			if c != nil {
				mbxes = append(mbxes, MailboxInfo(*c))
			}
		}
	}
	return
}

func (h *internalIMAPHandler) handleCreateMailbox(
	ctx context.Context,
	name string,
) (err error) {
	err = h.Client.Create(name)
	return
}

func (h *internalIMAPHandler) handleDeleteMailbox(
	ctx context.Context,
	name string,
) (err error) {
	err = h.Client.Delete(name)
	return
}

func (h *internalIMAPHandler) handleSelectMailbox(
	ctx context.Context,
	name string,
	ro bool,
) (data MailboxStatus, err error) {
	d, err := h.Client.Select(name, ro)
	if d != nil {
		data.LoadFromNative(d)
	}
	return
}

func (h *internalIMAPHandler) handleRenameMailbox(
	ctx context.Context,
	oldName string,
	newName string,
) (err error) {
	err = h.Client.Rename(oldName, newName)
	return
}

func (h *internalIMAPHandler) handleSubscribeMailbox(
	ctx context.Context,
	name string,
	doSubscribe bool,
) (err error) {
	if doSubscribe {
		err = h.Client.Subscribe(name)
	} else {
		err = h.Client.Unsubscribe(name)
	}
	return
}

func (h *internalIMAPHandler) handleExpunge(
	ctx context.Context,
) (err error) {
	err = h.Client.Expunge(nil)
	return
}

func (h *internalIMAPHandler) handleGetMailbox(
	ctx context.Context,
) (mbx *MailboxStatus, err error) {
	nmbx := h.Client.Mailbox()
	if nmbx != nil {
		mbx = &MailboxStatus{}
		mbx.LoadFromNative(nmbx)
	}
	return
}

func (h *internalIMAPHandler) handleSearchMails(
	ctx context.Context,
	sc SearchCriteria,
) (nums []uint32, err error) {
	nsc := imap.SearchCriteria{}
	sc.ToNative(&nsc)
	nums, err = h.Client.Search(&nsc)
	return
}

/*
import (
	"context"
	"io/ioutil"
	"sync"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type imapHandlerBackgroundTask struct {
	reqChan  chan IMAPRequest
	resChan  chan IMAPResponse
	doneChan chan error
	id       []byte
}

// IMAPHandler is connection-lifetime structure which holds IMAP client and handles incoming messages.
type IMAPHandler struct {
	Client *client.Client

	backgroundTasksLock *sync.RWMutex
	backgroundTasks     map[string]imapHandlerBackgroundTask
}

// Handle handles specified message and returns results to final channel.
// This function starts goroutines for handling commands in background.
// Note: only single goroutine should use handle at a time. This limitation may be removed in future.
func (h *IMAPHandler) Handle(ctx context.Context, reqChan <-chan IMAPRequest, resChan chan<- IMAPResponse) (err error) {
	var reqID []byte
	simpleRespond := func(err error, transform func() interface{}) {
		if err != nil {
			resChan <- IMAPResponse{
				ID:      reqID,
				IsFinal: true,

				Msg: err,
			}
		} else {
			resChan <- IMAPResponse{
				ID:      reqID,
				IsFinal: true,

				Msg: transform(),
			}
		}
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case recvMsg := <-reqChan:
			reqID = recvMsg.ID

			switch msg := recvMsg.Msg.(type) {
			case IMAPListMailboxRequest:
				res, err := h.handleListMailboxes(ctx, msg.Ref, msg.Name)
				simpleRespond(err, func() interface{} {
					return IMAPListMailboxResponse{
						Mailboxes: res,
					}
				})
			case IMAPListSubsRequest:
				res, err := h.handleListSubs(ctx, msg.Ref, msg.Name)
				simpleRespond(err, func() interface{} {
					return IMAPListSubsResponse{
						Mailboxes: res,
					}
				})
			case IMAPSubMailboxRequest:
				err = h.handleSubscribeMailbox(ctx, msg.MailboxName, msg.SetSub)
				simpleRespond(err, func() interface{} {
					return IMAPSubMailboxResponse{}
				})
			case IMAPCreateMailboxRequest:
				err = h.handleCreateMailbox(ctx, msg.Name)
				simpleRespond(err, func() interface{} {
					return IMAPCreateMailboxResponse{}
				})
			case IMAPDeleteMailboxRequest:
				err = h.handleDeleteMailbox(ctx, msg.Name)
				simpleRespond(err, func() interface{} {
					return IMAPDeleteMailboxResponse{}
				})
			case IMAPRenameMailboxRequest:
				err = h.handleRenameMailbox(ctx, msg.OldName, msg.NewName)
				simpleRespond(err, func() interface{} {
					return IMAPDeleteMailboxResponse{}
				})
			case IMAPSelectMailboxRequest:
				status, err := h.handleSelectMailbox(ctx, msg.Name, msg.ReadOnly)
				simpleRespond(err, func() interface{} {
					return IMAPSelectMailboxResponse{
						Status: status,
					}
				})
			case IMAPExpungeRequest:
				err = h.handleExpunge(ctx)
				simpleRespond(err, func() interface{} {
					return IMAPExpungeResponse{}
				})
			case IMAPSearchRequest:
				res, err := h.handleSearchMails(ctx, msg.Criteria)
				simpleRespond(err, func() interface{} {
					return IMAPSearchResponse{
						SeqNums: res,
					}
				})
			case IMAPFetchRequest:
				task := imapHandlerBackgroundTask{
					reqChan:  make(chan IMAPRequest),
					resChan:  make(chan IMAPResponse),
					doneChan: make(chan error),
					id:       reqID,
				}
				h.backgroundTasksLock.Lock()
				//detect duplicate ids?
				h.backgroundTasks[string(msg.RespChanID)] = task
				h.backgroundTasksLock.Unlock()
				/*
					ctx context.Context,
					ss SeqSet,
					items []string,
					id []byte, // id is set by caller(?) // and it's specific to this single fetch action
					reqChan <-chan IMAPRequest,
					resChan chan<- IMAPResponse,
					globalDoneChan chan<- error,
				* /
				go h.handleFetchMails(
					ctx,
					msg.SS,
					msg.Items,
					msg.RespChanID,
					task.reqChan,
					task.resChan,
					task.doneChan,
				)

				go func() {
					rcid := msg.RespChanID
				Loop:
					for {
						select {
						case err := <-task.doneChan:
							// TODO(teawithsand): here attach some label if error should be sent to client or handled as internal one
							// since it may come from context
							resChan <- IMAPResponse{
								ID:      rcid,
								IsFinal: true,

								Msg: err,
							}
							break Loop
						// error on done chan should return in this case
						/*
							case <-ctx.Done():
								resChan <- IMAPResponse{
									ID:      rcid,
									IsFinal: true,

									Msg: err,
								}
								break Loop
						* /
						case res := <-task.resChan:
							// note: this may be somewhat racy
							// when res chan may be closed at any time by receiver...
							// to fix this closing channel in this Handle fn is required
							//
							// or some wait group waiting for all goroutines to exit
							resChan <- res
						}
					}
				}()

				// do respond on main channel and then responses on secondary should be done
				simpleRespond(nil, func() interface{} {
					return IMAPFetchResponse{}
				})

			default:
				// It's unhandled message.
				// Is there some background task waiting for this message?

				h.backgroundTasksLock.RLock()
				task, ok := h.backgroundTasks[string(reqID)]
				if !ok {
					h.backgroundTasksLock.RUnlock()

					// REPORT ERROR HERE
					panic("NIY invalid id/msg error here")
				} else {
					h.backgroundTasksLock.RUnlock()
				}

				task.reqChan <- recvMsg
			}
		}
	}
	return
}

func (h *IMAPHandler) handleListMailboxes(ctx context.Context, ref, name string) (mbxes []MailboxInfo, err error) {
	resChan := make(chan *imap.MailboxInfo)
	doneChan := make(chan error)
	go func() {
		doneChan <- h.Client.List(ref, name, resChan)
	}()

Outer:
	for {
		select {
		case err := <-doneChan:
			return mbxes, err
			break Outer
		case c := <-resChan:
			if c != nil {
				mbxes = append(mbxes, MailboxInfo(*c))
			}
		}
	}
	return
}

func (h *IMAPHandler) handleListSubs(ctx context.Context, ref, name string) (mbxes []MailboxInfo, err error) {
	resChan := make(chan *imap.MailboxInfo)
	doneChan := make(chan error)
	go func() {
		doneChan <- h.Client.Lsub(ref, name, resChan)
	}()

Outer:
	for {
		select {
		case err := <-doneChan:
			return mbxes, err
			break Outer
		case c := <-resChan:
			if c != nil {
				mbxes = append(mbxes, MailboxInfo(*c))
			}
		}
	}
	return
}

func (h *IMAPHandler) handleCreateMailbox(
	ctx context.Context,
	name string,
) (err error) {
	err = h.Client.Create(name)
	return
}

func (h *IMAPHandler) handleDeleteMailbox(
	ctx context.Context,
	name string,
) (err error) {
	err = h.Client.Delete(name)
	return
}

func (h *IMAPHandler) handleSelectMailbox(
	ctx context.Context,
	name string,
	ro bool,
) (data MailboxStatus, err error) {
	d, err := h.Client.Select(name, ro)
	if d != nil {
		data.LoadFromNative(d)
	}
	return
}

func (h *IMAPHandler) handleRenameMailbox(
	ctx context.Context,
	oldName string,
	newName string,
) (err error) {
	err = h.Client.Rename(oldName, newName)
	return
}

func (h *IMAPHandler) handleSubscribeMailbox(
	ctx context.Context,
	name string,
	doSubscribe bool,
) (err error) {
	if doSubscribe {
		err = h.Client.Subscribe(name)
	} else {
		err = h.Client.Unsubscribe(name)
	}
	return
}

func (h *IMAPHandler) handleExpunge(
	ctx context.Context,
) (err error) {
	err = h.Client.Expunge(nil)
	return
}

func (h *IMAPHandler) handleGetMailbox(
	ctx context.Context,
) (mbx *MailboxStatus, err error) {
	nmbx := h.Client.Mailbox()
	if nmbx != nil {
		mbx = &MailboxStatus{}
		mbx.LoadFromNative(nmbx)
	}
	return
}

func (h *IMAPHandler) handleSearchMails(
	ctx context.Context,
	sc SearchCriteria,
) (nums []uint32, err error) {
	nsc := imap.SearchCriteria{}
	sc.ToNative(&nsc)
	nums, err = h.Client.Search(&nsc)
	return
}

func (h *IMAPHandler) handleFetchMails(
	ctx context.Context,
	ss SeqSet,
	items []string,
	id []byte, // id is set by caller(?) // and it's specific to this single fetch action
	reqChan <-chan IMAPRequest,
	resChan chan<- IMAPResponse,
	globalDoneChan chan<- error,
) {
	once := sync.Once{}
	sendErr := func(err error) {
		once.Do(func() {
			globalDoneChan <- err
		})
	}
	defer sendErr(nil)

	msgChan := make(chan *imap.Message)
	doneChan := make(chan error)

	fi := make([]imap.FetchItem, len(items))
	for i, it := range items {
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

	go func() {
		nss := &imap.SeqSet{}
		for _, seq := range ss.Set {
			nss.Set = append(nss.Set, imap.Seq(seq))
		}
		doneChan <- h.Client.Fetch(nss, fi, msgChan)
	}()

	msgOrDone := func() (*imap.Message, error) {
		for {
			select {
			case err := <-doneChan:
				return nil, err
			case <-ctx.Done():
				return nil, ctx.Err()
			case msg := <-msgChan:
				if msg == nil {
					continue
				}
				return msg, nil
			}
		}
	}

	commandOrDone := func() (interface{}, error) {
		select {
		case err := <-doneChan:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		case msg := <-reqChan:
			return msg, nil
		}
	}

	i := uint64(0)
	// message handle loop for processing big amount of messages
	for {
		// 1. Fetch message(or done err)
		msg, err := msgOrDone()
		if err != nil {
			sendErr(err)
			// TODO(teawithsand): handle error
			break
		}

		if msg == nil {
			// handle no more messages without error!
			resChan <- IMAPResponse{
				ID:  id,
				Msg: IMAPFetchResponse{}, // some summary here?
			}
			break
		}

		// 3/2. (one and a half) notify client about that
		resChan <- IMAPResponse{
			ID: id,
			Msg: IMAPFetchGotMessageResponse{
				Index: i,
			},
		}

		// 2. We have message. Read messages from underlying connection in order to determine what we should do with it.
		rawCmd, err := commandOrDone()
		if err != nil {
			sendErr(err)
			// TODO(teawithsand): handle error
			break
		}

	Outer:
		for {
			switch cmd := rawCmd.(type) {
			case IMAPFetchBodyRequest:
				if len(bsns) >= cmd.BodyPartNameIdx {
					// TODO(teawithsand): error here
					//sendErr(err)
				}
				lit := msg.GetBody(bsns[cmd.BodyPartNameIdx])
				if lit == nil {
					// error here as well (TODO)
					//sendErr(err)
				}
				resChan <- IMAPResponse{
					ID: id,
					Msg: IMAPFetchStreamResponse{
						Reader: ioutil.NopCloser(lit),
					},
				}

			case IMAPGoToNextMailRequest:
				break Outer
			default:
				//sendErr(err)
				panic("Invalid command type supplied! TODO handle it better way")
			}
		}

		i++
	}
	// TOOD(teawithsand): send message here with IsFinal: true,
	// OR send this message in task in handler goroutine
}
*/
