package main

import (
	"fmt"
	"strings"
	"sync"
)

const chanSize = 100

// EmailAddress represents a single address within a
type EmailAddress struct {
	name   string
	domain string
}

func (e EmailAddress) String() string {
	return fmt.Sprintf("%v@%v", e.name, e.domain)
}

// Envelope represents a single email
type Envelope struct {
	to      EmailAddress
	from    EmailAddress
	body    string
	subject string
}

// Mailbox is the structure that represents a mailbox on this machine.
type Mailbox struct {
	addr      string
	mail      []*Envelope
	spoolPath string
}

// A MailDirectory stores all the state of the smtp server, including actual emails
type MailDirectory struct {
	sync.RWMutex
	boxes map[string]*Mailbox
	mchan chan *Envelope
}

// NewMailDirectory creates and initalizes a new MailDirectory object
func NewMailDirectory() *MailDirectory {
	m := &MailDirectory{
		boxes: make(map[string]*Mailbox),
		mchan: make(chan *Envelope, chanSize),
	}
	m.AddAddress("Postmaster")
	return m
}

func newMailbox() (mbox *Mailbox) {
	return &Mailbox{
		mail: make([]*Envelope, 0),
	}
}

// ParseAddress parses the email address within a string and returns a pointer
// to the new object
func ParseAddress(s string) (e EmailAddress) {
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
func (m *MailDirectory) MainLoop() {
	fmt.Println("Starting mail loop...")
	for {
		mail := <-m.mchan
		mailboxes.AddMail(mail)
		fmt.Printf("Current mail for %s: %v", mail.to, mailboxes.GetMail(mail.to))
	}
}

// AddAddress adds an address to this mail server with the addres addr
func (m *MailDirectory) AddAddress(addr string) {
	m.Lock()
	if _, ok := m.boxes[addr]; !ok {
		m.boxes[addr] = newMailbox()
	}
	m.Unlock()
}

// AddMail adds the Envelope e to the mailbox specified by addr
func (m *MailDirectory) AddMail(e *Envelope) {
	m.Lock()
	dest := e.to.name
	if box, ok := m.boxes[dest]; ok {
		box.mail = append(box.mail, e)
	}
	m.Unlock()
}

// GetMail returns all mail recieved by the inbox addr.
func (m *MailDirectory) GetMail(addr EmailAddress) (e []*Envelope) {
	m.RLock()
	if box, ok := m.boxes[addr.name]; ok {
		e = box.mail
	}
	m.RUnlock()
	return e
}

// IsValidAddress returns true if the address given by addr is registered on this domain
func (m *MailDirectory) IsValidAddress(addr EmailAddress) bool {
	_, ok := m.boxes[addr.name]
	return ok
}
