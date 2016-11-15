package smtpserver

import (
	"fmt"
	"strings"
	"sync"
)

const chanSize = 100

// emailAddress represents a single address within a
type emailAddress struct {
	name   string
	domain string
}

func (e emailAddress) String() string {
	return fmt.Sprintf("%v@%v", e.name, e.domain)
}

// Envelope represents a single email
type envelope struct {
	to      emailAddress
	from    emailAddress
	body    string
	subject string
}

// Mailbox is the structure that represents a mailbox on this machine.
type mailbox struct {
	addr      string
	mail      []*envelope
	spoolPath string
}

// A MailDirectory stores all the state of the smtp server, including actual emails
type mailDirectory struct {
	sync.RWMutex
	boxes map[string]*mailbox
	mchan chan *envelope
}

// NewMailDirectory creates and initalizes a new MailDirectory object
func newMailDirectory() *mailDirectory {
	m := &mailDirectory{
		boxes: make(map[string]*mailbox),
		mchan: make(chan *envelope, chanSize),
	}
	m.AddAddress("Postmaster")
	return m
}

func newMailbox() (mbox *mailbox) {
	return &mailbox{
		mail: make([]*envelope, 0),
	}
}

// ParseAddress parses the email address within a string and returns a pointer
// to the new object
func parseAddress(s string) (e emailAddress) {
	s = strings.Trim(s, " <>")
	sep := strings.Split(s, "@")
	switch len(sep) {
	case 1:
		e.name = sep[0]
	case 2:
		e.name = sep[0]
		e.domain = sep[1]
	}
	return
}

// MainLoop is a consumer loop that handles Envelopes that are sent through its channel
func (m *mailDirectory) MainLoop() {
	fmt.Println("Starting mail loop...")
	for {
		mail := <-m.mchan
		m.AddMail(mail)
		fmt.Printf("Current mail for %s: %v", mail.to, m.GetMail(mail.to))
	}
}

// AddAddress adds an address to this mail server with the addres addr
func (m *mailDirectory) AddAddress(addr string) {
	m.Lock()
	if _, ok := m.boxes[addr]; !ok {
		m.boxes[addr] = newMailbox()
	}
	m.Unlock()
}

// AddMail adds the Envelope e to the mailbox specified by addr
func (m *mailDirectory) AddMail(e *envelope) {
	m.Lock()
	dest := e.to.name
	if box, ok := m.boxes[dest]; ok {
		box.mail = append(box.mail, e)
	}
	m.Unlock()
}

// GetMail returns all mail recieved by the inbox addr.
func (m *mailDirectory) GetMail(addr emailAddress) (e []*envelope) {
	m.RLock()
	if box, ok := m.boxes[addr.name]; ok {
		e = box.mail
	}
	m.RUnlock()
	return e
}

// IsValidAddress returns true if the address given by addr is registered on this domain
func (m *mailDirectory) IsValidAddress(addr emailAddress) bool {
	_, ok := m.boxes[addr.name]
	return ok
}
