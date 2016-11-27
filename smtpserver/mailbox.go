package smtpserver

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"runtime"
	"strings"
	"sync"

	// Necessary to get the sqlite driver
	_ "github.com/mattn/go-sqlite3"
)

const chanSize = 100
const domain = "localhost"

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
	mut   sync.RWMutex
	mchan chan *envelope
	db    *sql.DB
}

// NewMailDirectory creates and initalizes a new MailDirectory object
func newMailDirectory(database string) (m *mailDirectory, err error) {
	if db, err := sql.Open("sqlite3", database); err == nil {
		m = &mailDirectory{
			mchan: make(chan *envelope, chanSize),
			db:    db,
		}
		_, filename, _, _ := runtime.Caller(0)
		if s, err := ioutil.ReadFile(path.Join(path.Dir(filename), "schema.sql")); err == nil {
			db.Exec(string(s))
		}
		if id := m.IsValidAddress(emailAddress{"Postmaster", domain}); id == -1 {
			m.AddAddress(emailAddress{"Postmaster", domain})
		}
	}
	return
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
	}
}

// AddAddress adds an address to this mail server with the addres addr
func (m *mailDirectory) AddAddress(addr emailAddress) (id int) {
	m.mut.Lock()
	m.db.Exec("INSERT INTO address(name, domain) VALUES($1, $2)", addr.name, addr.domain)
	m.mut.Unlock()
	return m.IsValidAddress(addr)
}

// AddMail adds the Envelope e to the mailbox specified by addr
func (m *mailDirectory) AddMail(e *envelope) {
	var fromID, toID int
	if fromID = m.IsValidAddress(e.from); fromID == -1 {
		fromID = m.AddAddress(e.from)
	}
	if toID = m.IsValidAddress(e.to); toID == -1 {
		return
	}
	m.mut.Lock()
	m.db.Exec("INSERT INTO emails(from_addr, to_addr, subject, body) VALUES($1, $2, $3, $4)", fromID, toID, e.subject, e.body)
	m.mut.Unlock()
}

// GetMail returns all mail recieved by the inbox addr.
func (m *mailDirectory) GetMail(addr emailAddress) (e []*envelope, err error) {
	var id, toAddr, fromAddr int
	var subject, body string
	m.mut.RLock()
	defer m.mut.RUnlock()
	row := m.db.QueryRow("SELECT id FROM address WHERE name=$1 AND domain=$2", addr.name, addr.domain)
	if err := row.Scan(&id); err != nil {
		log.Fatal(err)
		return nil, err
	}
	rows, err := m.db.Query("SELECT * FROM emails WHERE to_addr=$1", id)
	if err != nil {
		log.Fatal(err)
		return
	}
	e = make([]*envelope, 0)
	for rows.Next() {
		if err := rows.Scan(&id, &fromAddr, &toAddr, &subject, &body); err == nil {
			e = append(e, &envelope{
				to:      m.GetAddr(toAddr),
				from:    m.GetAddr(fromAddr),
				subject: subject,
				body:    body,
			})
		}
	}
	return
}

func (m *mailDirectory) GetAddr(id int) emailAddress {
	e := emailAddress{}
	row := m.db.QueryRow("SELECT addr, domain FROM address WHERE id=$1", id)
	row.Scan(&e.name, &e.domain)
	return e
}

// IsValidAddress returns true if the address given by addr is registered on this domain
func (m *mailDirectory) IsValidAddress(addr emailAddress) (id int) {
	m.mut.RLock()
	err := m.db.QueryRow("SELECT id from address WHERE name=$1 and domain=$2", addr.name, addr.domain).Scan(&id)
	m.mut.RUnlock()
	if err != nil {
		id = -1
	}
	return
}
