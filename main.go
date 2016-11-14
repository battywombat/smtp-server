package main

import (
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strconv"
)

const (
	softwareName = "PSMTP"
	version      = "0.1"
)

// GLOBAL OBJECTS
var mailboxes *MailDirectory

func server(port int) {
	fmt.Println("Beginning server process...")
	l, err := net.Listen("tcp", "localhost:"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error opening socket: ", err.Error())
	}
	defer l.Close()
	go client()
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error with connection: ", err.Error())
		} else {
			cli := NewClient(conn)
			go cli.Handler()
		}
	}
}

func client() {

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	to := []string{"recipient@example.net"}
	msg := []byte("To: recipient@example.net\r\n" +
		"Subject: discount Gophers!\r\n" +
		"\r\n" +
		"This is the email body.\r\n")
	err := smtp.SendMail("localhost:8000", nil, "sender@example.org", to, msg)
	if err != nil {
		log.Fatal(err)
	}
}

func loadMailboxes() {
	mailboxes = NewMailDirectory()
	mailboxes.AddAddress("recipient")
	go mailboxes.MainLoop()
}

func main() {
	loadMailboxes()
	server(8000)
}
