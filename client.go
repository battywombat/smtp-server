package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	greeting          = 220
	quit              = 221
	ok                = 250
	startMail         = 354
	syntaxError       = 500
	outOfSequence     = 503
	noValidRecipients = 504
	mailboxNotFound   = 550
)

// Client stores all the variables related to a single client connection
type Client struct {
	conn          net.Conn
	in            *bufio.Scanner
	out           *bufio.Writer
	isESMTP       bool
	ip            string
	inProgress    bool
	currentLetter *Envelope
}

type response struct {
	code    int
	message string
}

// NewClient creates a new client object from a given net.Conn
func NewClient(conn net.Conn) *Client {
	cli := &Client{}
	cli.conn = conn
	cli.in = bufio.NewScanner(conn)
	cli.out = bufio.NewWriter(conn)
	return cli
}

// Close implements the Closer interface on a client connection object
func (cli *Client) Close() {
	cli.conn.Close()
}

func (cli *Client) handleHELO(tokens []string) error {
	if len(tokens) < 2 {
		cli.writeResponse(syntaxError, "No IP provided")
		return nil
	}
	cli.ip = tokens[1]
	// I think getting your own hostname can literally never go wrong
	hostname, _ := os.Hostname()
	hostIP, _ := net.LookupHost(hostname)
	return cli.writeResponse(ok, fmt.Sprintf("%v Hello", hostIP[3]))
}

func (cli *Client) handleMAIL(tokens []string) error {
	if len(tokens) < 2 {
		cli.writeResponse(syntaxError, "No sender specified")
		return nil
	}
	addr := strings.Split(tokens[1], ":")
	if len(addr) < 2 {
		cli.writeResponse(syntaxError, "No sender specified")
		return nil
	}
	// SMTP standard says that a MAIL command should always reset state
	// cli.currentLetter = &Envelope{from: strings.Trim(addr[1], "<>")}
	cli.currentLetter = new(Envelope)
	cli.currentLetter.from = ParseAddress(addr[1])
	cli.writeResponse(ok, "Ok")
	return nil
}

func (cli *Client) handleRCPT(tokens []string) error {
	if len(tokens) < 2 {
		cli.writeResponse(syntaxError, "No reciver specified")
		return nil
	}
	addr := strings.Split(tokens[1], ":")
	if len(addr) < 2 {
		cli.writeResponse(syntaxError, "No reciver specified")
		return nil
	}
	cli.currentLetter.to = ParseAddress(addr[1])
	if !mailboxes.IsValidAddress(cli.currentLetter.to) {
		fmt.Printf("Unknown mailbox: %v\n", cli.currentLetter.to)
		cli.writeResponse(mailboxNotFound, "Can't find mailbox")
	} else {
		cli.writeResponse(ok, "Ok")
	}
	return nil
}

func (cli *Client) handleDATA() (err error) {
	var t []string
	// need to make sure that we've recieved MAIL TO and RCPT TO messages already
	if cli.currentLetter == nil || cli.currentLetter.from == (EmailAddress{}) || cli.currentLetter.to == (EmailAddress{}) {
		return cli.writeResponse(outOfSequence, "")
	}
	err = cli.writeResponse(startMail, "start mail input")
	for t, err = cli.nextTokens(); err == nil && t[0] != "."; t, err = cli.nextTokens() {
		if len(t) == 1 && t[0] == "" { // Email body always follows empty line
			cli.currentLetter.body, err = cli.nextLine()
			if err != nil {
				break
			}
			continue
		}
		switch t[0] {
		case "To:":
			cli.currentLetter.to = ParseAddress(t[1])
		case "From:":
			cli.currentLetter.from = ParseAddress(t[1])
		case "Subject:":
			cli.currentLetter.subject = strings.Join(t[1:], " ")
		default: // invalid command
			cli.writeResponse(syntaxError, "")
			return
		}
	}
	cli.writeResponse(ok, "ok")
	return
}

func (cli *Client) writeResponse(code int, message string) error {
	s := fmt.Sprintf("%v %v \r\n", strconv.Itoa(code), message)
	if _, err := cli.out.Write([]byte(s)); err != nil {
		return err
	}
	cli.out.Flush()
	return nil
}

func (cli *Client) nextLine() (s string, err error) {
	if !cli.in.Scan() {
		err = errors.New("Trouble reading from socket")
	} else {
		s = cli.in.Text()
	}
	return
}

func (cli *Client) nextTokens() ([]string, error) {
	s, err := cli.nextLine()
	return strings.Split(s, " "), err
}

// Handler is the main handler function for a particular client connection
func (cli *Client) Handler() {
	defer cli.Close()
	info := fmt.Sprintf("%v %v", softwareName, version)
	if err := cli.writeResponse(greeting, info); err != nil {
		fmt.Println("Couldn't write message to client: ", err.Error())
		return
	}
	var tokens []string
	var err error
	for {
		if tokens, err = cli.nextTokens(); err != nil {
			fmt.Println("Error reading in command: ", cli.in.Err())
			return
		}
		if len(tokens) < 1 {
			cli.writeResponse(syntaxError, "No command")
			continue
		}
		switch tokens[0] {
		case "EHLO":
			cli.isESMTP = true
			fallthrough
		case "HELO":
			if err := cli.handleHELO(tokens); err != nil {
				fmt.Println("issue responding to HELO: ", err.Error())
				return
			}
		case "MAIL":
			if err := cli.handleMAIL(tokens); err != nil {
				fmt.Println("issue responding to MAIL: ", err.Error())
			}
		case "RCPT":
			if err := cli.handleRCPT(tokens); err != nil {
				fmt.Println("error responding to RCPT: ", err.Error())
			}
		case "DATA":
			if err := cli.handleDATA(); err != nil {
				fmt.Println("Error responding to DATA: ", err.Error())
			} else {
				mailboxes.mchan <- cli.currentLetter
			}
		case "QUIT":
			cli.writeResponse(quit, "Closing connection")
			return
		default:
			fmt.Println("Error: unrecognized command: ", tokens[0])
			return
		}
	}
}
