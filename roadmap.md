Things to do:
Restructure program so that it has a single outward facing API (a SMTPServer struct)
Implement database backend for server
Remaining behavior for minimum SMTP server (VRFY, RSET)
Server should shut down cleanly (using the WaitGroup struct added to SMTPServer)
Look into implementable SMTP extensions
An http-based API that can be used to query mail for a given address

Refactor constantly