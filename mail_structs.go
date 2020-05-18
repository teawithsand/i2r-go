package i2r

import (
	"net/textproto"
	"time"

	"github.com/emersion/go-imap"
)

type Address struct {
	// The personal name.
	PersonalName string `json:"personal_name"`
	// The SMTP at-domain-list (source route).
	AtDomainList string `json:"at_domain_list"`
	// The mailbox name.
	MailboxName string `json:"mailbox_name"`
	// The host name.
	HostName string `json:"host_name"`
}

type PartSpecifier string

type BodyPartName struct {
	// The specifier of the requested part.
	Specifier PartSpecifier `json:"part_specifier"`
	// The part path. Parts indexes start at 1.
	Path []int `json:"path"`
	// If Specifier is HEADER, contains header fields that will/won't be returned,
	// depending of the value of NotFields.
	Fields []string `json:"fields"`
	// If set to true, Fields is a blacklist of fields instead of a whitelist.
	NotFields bool `json:"not_fields"`
}

type BodySectionName struct {
	BodyPartName `json:"body_part_name"`

	// If set to true, do not implicitly set the \Seen flag.
	Peek bool `json:"peek"`
	// The substring of the section requested. The first value is the position of
	// the first desired octet and the second value is the maximum number of
	// octets desired.
	Partial []int `json:"partial"`
	// contains filtered or unexported fields
}

type MailboxInfo struct {
	// The mailbox attributes.
	Attributes []string `json:"attributes"`
	// The server's path separator.
	Delimiter string `json:"delimeter"`
	// The mailbox name.
	Name string `json:"name"`
}

type Envelope struct {
	// The message date.
	Date time.Time `json:"date"`
	// The message subject.
	Subject string `json:"subject"`
	// The From header addresses.
	From []*Address `json:"from"`
	// The message senders.
	Sender []*Address `json:"sender"`
	// The Reply-To header addresses.
	ReplyTo []*Address `json:"reply_to"`
	// The To header addresses.
	To []*Address `json:"to"`
	// The Cc header addresses.
	Cc []*Address `json:"cc"`
	// The Bcc header addresses.
	Bcc []*Address `json:"bcc"`
	// The In-Reply-To header. Contains the parent Message-Id.
	InReplyTo string `json:"in_reply_to"`
	// The Message-Id header.
	MessageId string `json:"message_id"`
}

type Seq struct {
	/*
		Start uint32 `json:"start"`
		Stop  uint32 `json:"stop"`
	*/
	Start, Stop uint32
}

type SeqSet struct {
	Set []Seq `json:"set"`
}

func (ss *SeqSet) FromNative(nss *imap.SeqSet) {
	ss.Set = make([]Seq, len(nss.Set))
	for i, s := range nss.Set {
		ss.Set[i] = Seq(s)
	}
}

func (ss *SeqSet) ToNative(nss *imap.SeqSet) {
	nss.Set = make([]imap.Seq, len(ss.Set))
	for i, s := range ss.Set {
		nss.Set[i] = imap.Seq(s)
	}
}

type SearchCriteria struct {
	SeqNum *SeqSet `json:"seq_num,omitempty"` // Sequence number is in sequence set
	Uid    *SeqSet `json:"uid,omitempty"`     // UID is in sequence set

	// Time and timezone are ignored
	Since      time.Time `json:"since,omitempty"`       // Internal date is since this date
	Before     time.Time `json:"before,omitempty"`      // Internal date is before this date
	SentSince  time.Time `json:"sent_since,omitempty"`  // Date header field is since this date
	SentBefore time.Time `json:"sent_before,omitempty"` // Date header field is before this date

	Header textproto.MIMEHeader `json:"header,omitempty"` // Each header field value is present
	Body   []string             `json:"body,omitempty"`   // Each string is in the body
	Text   []string             `json:"text,omitempty"`   // Each string is in the text (header + body)

	WithFlags    []string `json:"with_flags,omitempty"`    // Each flag is present
	WithoutFlags []string `json:"without_flags,omitempty"` // Each flag is not present

	Larger  uint32 `json:"larger,omitempty"`  // Size is larger than this number
	Smaller uint32 `json:"smaller,omitempty"` // Size is smaller than this number

	Not []*SearchCriteria    `json:"not,omitempty"` // Each criteria doesn't match
	Or  [][2]*SearchCriteria `json:"or,omitempty"`  // Each criteria pair has at least one match of two
}

func (sc *SearchCriteria) ToNative(nsc *imap.SearchCriteria) {
	if nsc == nil {
		panic("Provided nil native search criteria to native")
	}
	if sc.SeqNum != nil {
		nsc.SeqNum = &imap.SeqSet{}
		sc.SeqNum.ToNative(nsc.SeqNum)
	} else {
		nsc.SeqNum = nil
	}

	if sc.Uid != nil {
		nsc.Uid = &imap.SeqSet{}
		sc.Uid.ToNative(nsc.Uid)
	} else {
		nsc.Uid = nil
	}

	nsc.Since = sc.Since
	nsc.Before = sc.Before
	nsc.SentSince = sc.SentSince
	nsc.SentBefore = sc.SentBefore

	nsc.Header = sc.Header
	nsc.Body = sc.Body
	nsc.Text = sc.Text

	nsc.Larger = sc.Larger
	nsc.Smaller = sc.Smaller

	// go does quite well with big stacks
	// so recursing is fine
	if len(sc.Not) == 0 {
		nsc.Not = nil
	} else {
		nsc.Not = make([]*imap.SearchCriteria, len(sc.Not))
		for i, c := range sc.Not {
			if c == nil {
				continue
			}
			nsc.Not[i] = &imap.SearchCriteria{}
			c.ToNative(nsc.Not[i])
		}
	}

	if len(sc.Or) == 0 {
		nsc.Or = nil
	} else {
		nsc.Or = make([][2]*imap.SearchCriteria, len(sc.Or))
		for i, c := range sc.Or {
			if c[0] != nil {
				nsc.Or[i][0] = &imap.SearchCriteria{}
				c[0].ToNative(nsc.Or[i][0])
			}

			if c[1] != nil {
				nsc.Or[i][1] = &imap.SearchCriteria{}
				c[1].ToNative(nsc.Or[i][1])
			}
		}
	}
}

type MailboxStatus struct {
	// The mailbox name.
	Name string `json:"name"`
	// True if the mailbox is open in read-only mode.
	ReadOnly bool `json:"read_only"`

	// mailbox status is simplified.
	// This way it can be JSON marshalled directly.
	/*
		    // The mailbox items that are currently filled in. This map's values
		    // should not be used directly, they must only be used by libraries
		    // implementing extensions of the IMAP protocol.
		    Items map[StatusItem]interface{}

		    // The Items map may be accessed in different goroutines. Protect
		    // concurrent writes.
			ItemsLocker sync.Mutex
	*/

	// The mailbox flags.
	Flags []string `json:"flags"`
	// The mailbox permanent flags.
	PermanentFlags []string `json:"permantent_flags"`
	// The sequence number of the first unseen message in the mailbox.
	UnseenSeqNum uint32 `json:"unseen_seq_num"`

	// The number of messages in this mailbox.
	Messages uint32 `json:"messages"`
	// The number of messages not seen since the last time the mailbox was opened.
	Recent uint32 `json:"recent"`
	// The number of unread messages.
	Unseen uint32 `json:"unseen"`

	// The next UID.
	UidNext uint32 `json:"uid_next"`

	// Together with a UID, it is a unique identifier for a message.
	// Must be greater than or equal to 1.
	UidValidity uint32 `json:"uid_validity"`
}

func (ms *MailboxStatus) LoadFromNative(status *imap.MailboxStatus) {
	ms.Name = status.Name
	ms.ReadOnly = status.ReadOnly
	ms.Flags = status.Flags
	ms.PermanentFlags = status.PermanentFlags
	ms.UnseenSeqNum = status.UnseenSeqNum
	ms.Messages = status.Messages
	ms.Unseen = status.Unseen
	ms.UidNext = status.UidNext
	ms.UidValidity = status.UidValidity
}
