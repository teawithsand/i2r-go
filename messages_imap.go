package i2r

// TODO(teawithsand): implement uid operations

type IMAPListMailboxRequest struct {
	Ref  string `json:"ref"`
	Name string `json:"name"`
}
type IMAPListMailboxResponse struct {
	Mailboxes []MailboxInfo `json:"mailboxes"`
}

type IMAPListSubsRequest struct {
	Ref  string `json:"ref"`
	Name string `json:"name"`
}
type IMAPListSubsResponse struct {
	Mailboxes []MailboxInfo `json:"mailboxes"`
}

type IMAPSubMailboxRequest struct {
	MailboxName string `json:"mailbox_name"`
	SetSub      bool   `json:"set_sub"`
}
type IMAPSubMailboxResponse struct {
}

type IMAPCreateMailboxRequest struct {
	Name string `json:"name"`
}

type IMAPCreatMailboxResponse struct {
}

type IMAPDeleteMailboxRequest struct {
	Name string `json:"name"`
}
type IMAPDeleteMailboxResponse struct{}

type IMAPRenameMailboxRequest struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}
type IMAPRenameMailboxResponse struct{}

type IMAPNoopRequest struct{}
type IMAPNoopResponse struct{}

type IMAPQuitRequest struct {
	DoLogout    bool `json:"do_logout"`
	DoTerminate bool `json:"do_terminate"`
}
type IMAPQuitResponse struct{}

type IMAPSelectMailboxRequest struct {
	Name     string `json:"name"`
	ReadOnly bool   `json:"read_only"`
}
type IMAPSelectMaibloxResponse struct {
	Status MailboxStatus `json:"status"`
}

type IMAPExpungeRequest struct{}
type IMAPExpungeResponse struct {
	// ignore returned seq nums?
}

type IMAPFetchRequest struct {
	SS    SeqSet   `json:"ss"`
	Items []string `json:"items"`
}
type IMAPFetchResponse struct {
	// respoonse is now streammed in chunks
}

type IMAPSearchRequest struct {
	Criteria SearchCriteria `json:"criteria"`
}
type IMAPSearchResponse struct {
	SeqNums []uint32 `json:"seq_nums"`
}

type IMAPStoreRequest struct {
	SS        SeqSet   `json:"ss"`
	StoreItem string   `json:"store_item"` // in fact it's enum
	Flags     []string `json:"flags"`
}
type IMAPStoreResponse struct {
	// ignore returned message info
}

type IMAPCheckRequest struct{}
type IMAPCheckResponse struct{}
