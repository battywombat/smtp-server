package smtpserver

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
type client struct {
	server        *SMTPServer
	conn          net.Conn
	in            *bufio.Scanner
	out           *bufio.Writer
	isESMTP       bool
	ip            string
	inProgress    bool
	currentLetter *envelope
}

type response struct {
	code    int
	message string
}

// newClient creates a new client object from a given net.Conn
func newClient(conn net.Conn, s *SMTPServer) *client {
	cli := &client{}
	cli.conn = conn
	cli.in = bufio.NewScanner(conn)
	cli.out = bufio.NewWriter(conn)
	cli.server = s
	return cli
}

// Close implements the Closer interface on a client connection object
func (cli *client) Close() {
	cli.conn.Close()
}

func (cli *client) handleHELO(tokens []string) error {
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

func (cli *client) handleMAIL(tokens []string) error {
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
	cli.currentLetter = &envelope{}
	cli.currentLetter.from = parseAddress(addr[1])
	cli.writeResponse(ok, "Ok")
	return nil
}

func (cli *client) handleRCPT(tokens []string) error {
	if len(tokens) < 2 {
		cli.writeResponse(syntaxError, "No reciver specified")
		return nil
	}
	addr := strings.Split(tokens[1], ":")
	if len(addr) < 2 {
		cli.writeResponse(syntaxError, "No reciver specified")
		return nil
	}
	cli.currentLetter.to = append(cli.currentLetter.to, parseAddress(addr[1]))
	l := len(cli.currentLetter.to) - 1
	if id := cli.server.mdir.IsValidAddress(cli.currentLetter.to[l]); id == -1 {
		cli.writeResponse(mailboxNotFound, "Can't find mailbox")
	} else {
		cli.writeResponse(ok, "Ok")
	}
	return nil
}

func (cli *client) handleDATA() (err error) {
	var t []string
	// need to make sure that we've recieved MAIL TO and RCPT TO messages already
	if cli.currentLetter == nil || cli.currentLetter.from == (emailAddress{}) || len(cli.currentLetter.to) == 0 {
		return cli.writeResponse(outOfSequence, "")
	}
	err = cli.writeResponse(startMail, "start mail input")
	for t, err = cli.nextTokens(); err == nil && !(len(t) == 1 && t[0] == "."); t, err = cli.nextTokens() {
		switch t[0] {
		case "Subject:":
			cli.currentLetter.subject = strings.Join(t[1:], " ")
		}
		cli.currentLetter.body += strings.Join(t, " ") + "\r\n"
	}
	cli.writeResponse(ok, "OK")
	return
}

func (cli *client) handleRSET() error {
	cli.currentLetter = &envelope{}
	cli.inProgress = false
	cli.writeResponse(ok, "ok")
	return nil
}

func (cli *client) handleVRFY(tokens []string) error {
	var addr emailAddress
	switch {
	case len(tokens) == 1:
		addr = parseAddress(tokens[0])
	case len(tokens) == 3 && tokens[0] == "User" && tokens[1] == "Name":
		addr = parseAddress(tokens[2])
	default:
		cli.writeResponse(syntaxError, "Invalid username format")
	}
	if cli.server.mdir.IsValidAddress(addr) == -1 {
		cli.writeResponse(mailboxNotFound, addr.String())
	} else {
		cli.writeResponse(ok, addr.String())
	}
	return nil
}

func (cli *client) writeResponse(code int, message string) error {
	s := fmt.Sprintf("%v %v \r\n", strconv.Itoa(code), message)
	if _, err := cli.out.Write([]byte(s)); err != nil {
		return err
	}
	cli.out.Flush()
	return nil
}

func (cli *client) nextLine() (s string, err error) {
	if !cli.in.Scan() {
		err = errors.New("Trouble reading from socket")
	} else {
		s = cli.in.Text()
	}
	return
}

func (cli *client) nextTokens() ([]string, error) {
	s, err := cli.nextLine()
	return strings.Split(s, " "), err
}

// Append the Recieved line to the
func (cli *client) addTimeStamp() {

}

// Handler is the main handler function for a particular client connection
func (cli *client) Handler() {
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
			if len(tokens) > 1 {
				cli.writeResponse(syntaxError, fmt.Sprintf("Arguments after DATA: %v", tokens[1:]))
				break
			}
			if err := cli.handleDATA(); err != nil {
				fmt.Println("Error responding to DATA: ", err.Error())
			} else {
				cli.server.mdir.mchan <- cli.currentLetter
				cli.currentLetter = &envelope{}
			}
		case "VRFY":
			if err := cli.handleVRFY(tokens); err != nil {
				fmt.Println("Error responding to VRFY: ", err.Error())
			}
		case "RSET":
			if len(tokens) > 1 {
				cli.writeResponse(syntaxError, fmt.Sprintf("Arguments after RSET: %v", tokens[1:]))
				break
			}
			if err := cli.handleRSET(); err != nil {
				fmt.Println("error responding to RSET: ", err.Error())
			}
		case "NOOP":
			cli.writeResponse(ok, "OK")
		case "QUIT":
			if len(tokens) > 1 {
				cli.writeResponse(syntaxError, fmt.Sprintf("Arguments after QUIT: %v", tokens[1:]))
				break
			}
			cli.writeResponse(quit, "Closing connection")
			return
		default:
			fmt.Println("Error: unrecognized command: ", tokens[0])
			cli.writeResponse(syntaxError, fmt.Sprintf("Unrecognized command: %v", tokens[0]))
		}
	}
}
