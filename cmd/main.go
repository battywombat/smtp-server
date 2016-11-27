package main

import smtpserver "github.com/battywombat/smtpserver/smtpserver"

func main() {
	s := smtpserver.NewSMTPServer(8000, "./mail.db")
	s.DoServer()
}
