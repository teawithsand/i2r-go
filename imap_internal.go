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
