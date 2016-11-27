package smtpserver

import (
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strconv"
	"sync"
)

const (
	softwareName = "PSMTP"
	version      = "0.1"
)

// SMTPServer is the top level SMTP Server exported to clients
type SMTPServer struct {
	port int
	mdir *mailDirectory
	wg   sync.WaitGroup
}

func clientThread() {

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	to := []string{"recipient@" + domain}
	msg := []byte("To: recipient@example.net\r\n" +
		"Subject: discount Gophers!\r\n" +
		"\r\n" +
		"This is the email body.\r\n")
	err := smtp.SendMail("localhost:8000", nil, "sender@example.org", to, msg)
	if err != nil {
		log.Fatal(err)
	}
}

// DoServer is the main loop used to run an SMTP server instance
func (s *SMTPServer) DoServer() {
	fmt.Println("Beginning server process...")
	l, err := net.Listen("tcp", "localhost:"+strconv.Itoa(s.port))
	if err != nil {
		fmt.Println("Error opening socket: ", err.Error())
	}
	defer l.Close()
	go clientThread()
	go s.mdir.MainLoop()
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error with connection: ", err.Error())
		} else {
			cli := newClient(conn, s)
			go cli.Handler()
		}
	}
}

// NewSMTPServer creates a pointer to an SMTPServer object, and initalizes
// all of its fields to their default values
func NewSMTPServer(port int, database string) (s *SMTPServer) {
	var err error
	s = &SMTPServer{}
	s.port = port
	s.mdir, err = newMailDirectory(database)
	if err != nil {
		return
	}
	s.mdir.AddAddress(&emailAddress{"recipient", domain})
	return
}
