package main

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"protogo"
	"protogo/telnet"
)

func main() {
	log.Printf("Start echo server.\n")

	// Binds SIGTERM signal on this channel
	closeChannel := make(chan os.Signal, 1)
	signal.Notify(closeChannel, os.Interrupt)

	server, err := protogo.Listen(8025, telnet.NewServer(welcome))
	if err != nil {
		log.Fatalf("Unable to start the server : %s\n", err.Error())
	}

	defer server.Stop()

	// Server CONNECTED !!
	log.Printf("Server Connected on %s\n", server.Address)

	log.Println("Ctrl-C / SIGTERM to quit.")
	<-closeChannel
}

// ----------------------------------->

// The current Mail, and the telnet EventHandler
// Each property defined the current step
type Mail struct {
	// who created sayed HELO
	who string
	// once we start the Mail, the from address
	from string
	// the list of MAIL recipients
	recipients []string
	// the MAIL content
	content bytes.Buffer
}

// Accept the connection
func welcome() (telnet.Response, telnet.EventHandler) {
	mail := &Mail{}
	// we start with a REQUEST
	return response(220, "Welcome, SMTP Ready", telnet.REQUEST), mail
}

// receive a command line as request
func (m *Mail) OnRequest(line telnet.Line) telnet.Response {

	// first we parse the command line
	cmd := parseCmd(line)

	// handle common cmds
	switch cmd.t {
	case QUIT:
		return quit()
	case NOOP:
		return ok("Ok")
	case UNKNOWN:
		return unknown()

	// reset the current mail
	case RSET:
		{
			m.reset()
			return ok("Ok")
		}

	// say HELO again
	case HELO:
		{
			if m.who != "" {
				return alreadyMet(m.who)
			}
			break
		}
	}

	// define the current step
	// and jump to the method that handle this step
	switch {
	case m.who == "":
		return m.heloStep(cmd)
	case m.from == "":
		return m.mailStep(cmd)
	}

	// we probably are in the RCPT step
	return m.rcptStep(cmd)
}

// read the request as Data
// the only way this method may be called, is when we're receiving mail data
func (m *Mail) OnData(req *telnet.Request) telnet.Response {

	// read line by line, until finding the final point
	for {

		// read the next line
		line, err := req.NextLine()
		if err != nil {
			return unavailable()
		}

		// the end ??
		if line.Value == "." {
			m.send()
			m.reset()
			return ok("Mail accepted")
		}

		// write the <CRLF> for the previous line
		if m.content.Len() > 0 {
			m.content.WriteString("\r\n")
		}

		if line.Value == ".." {
			// the "." is escaped
			m.content.WriteString(".")
		} else {
			// write the entire line
			m.content.WriteString(line.Value)
		}
	}
}

// reset this mail
func (m *Mail) reset() {
	m.from = ""
	m.recipients = nil
	m.content.Reset()
}

// send this mail
// (print it on the stdout)
func (m *Mail) send() {
	log.Println("Accepted mail")
	log.Println("-------------")
	log.Println("From : ", m.from)
	log.Println("To : ", m.recipients)
	log.Println(m.content.String())
}

// ----------------------------------->
// Steps

// helo step
func (m *Mail) heloStep(cmd *cmdLine) telnet.Response {

	// we expect HELO
	if cmd.t != HELO {
		return badSequence("polite people say HELO first")
	}

	// check the arguments
	if len(cmd.Command.Args) != 1 {
		return syntaxError("argument expected")
	}

	m.who = cmd.Command.Args[0]

	return ok("Helo, pleased to meet you " + m.who) // FIXME how to get the IP address
}

// mail step
func (m *Mail) mailStep(cmd *cmdLine) telnet.Response {

	// we expect MAIL
	if cmd.t != MAIL {
		return badSequence("need MAIL before " + strings.ToUpper(cmd.Command.Name))
	}

	switch {
	case len(cmd.Command.Args) > 2:
		return syntaxError("Too many arguments given")
	case len(cmd.Command.Args) != 2:
		return syntaxError("Not enough arguments given")
	case cmd.Command.Args[0] != "FROM:":
		return syntaxError("FROM: was expected as first argument")
	}

	// TODO parse the email address
	m.from = cmd.Command.Args[1]

	return ok("Sender ok : " + m.from)
}

// the recipient step
func (m *Mail) rcptStep(cmd *cmdLine) telnet.Response {

	// handle commands
	switch {

	// we can continue on data only
	// if we have at least one recipient
	case cmd.t == DATA:
		{

			if len(m.recipients) == 0 {
				return badSequence("need RCPT before " + strings.ToUpper(cmd.Command.Name))
			} else {
				return input()
			}
		}

	case cmd.t == MAIL:
		{
			return badSequence("MAIL already started")
		}

	// we expect RCPT
	case cmd.t != RCPT:
		{
			return badSequence("Illegal command " + strings.ToUpper(cmd.Command.Name))
		}
	}

	// handle arguments
	switch {
	case len(cmd.Command.Args) > 2:
		return syntaxError("Too many arguments given")
	case len(cmd.Command.Args) != 2:
		return syntaxError("Not enough arguments given")
	case cmd.Command.Args[0] != "TO:":
		return syntaxError("TO: was expected as first argument")
	}

	// TODO parse the email address
	recipient := cmd.Command.Args[1]
	m.recipients = append(m.recipients, recipient)

	return ok("Recipient ok : " + recipient)
}

// ----------------------------------->
// Command

type cmdType int

const (
	QUIT    cmdType = iota // quit the server
	NOOP                   // do nothing
	HELO                   // identify yourself
	MAIL                   // start a new mail
	RCPT                   // give the mail recipients
	DATA                   // start mail input
	RSET                   // cancel the mail
	SEND                   // finalize and send the mail
	UNKNOWN                // unknown command
)

// an smtp command line
type cmdLine struct {
	// the parent telnet command line
	telnet.Command
	// the type of this command
	t cmdType
}

// parse the cmd
func parseCmd(line telnet.Line) *cmdLine {

	cmd := line.AsCommand()
	switch cmd.Name {
	case "noop":
		return &cmdLine{cmd, NOOP}
	case "helo":
		return &cmdLine{cmd, HELO}
	case "mail":
		return &cmdLine{cmd, MAIL}
	case "rcpt":
		return &cmdLine{cmd, RCPT}
	case "data":
		return &cmdLine{cmd, DATA}
	case "rset":
		return &cmdLine{cmd, RSET}
	case "send":
		return &cmdLine{cmd, SEND}

	case "quit", "bye", "exit":
		return &cmdLine{cmd, QUIT}
	}

	return &cmdLine{cmd, UNKNOWN}
}

// ----------------------------------->
// Responses

// an smtp response, including a code
// 2xx - ok
// 3xx - continue
// 5xx - error
type smtpResponse struct {
	// the line code
	*telnet.LineResponse
	// the response code
	code int
}

// write this SMTP response
func (resp *smtpResponse) WriteTo(writer *bufio.Writer) {
	// write the code
	writer.WriteString(strconv.Itoa(resp.code) + " ")
	// call the parent
	resp.LineResponse.WriteTo(writer)
}

// creates a new SMTP response
func response(code int, text string, next telnet.State) *smtpResponse {

	return &smtpResponse{
		telnet.NewLineResponse(text, next),
		code,
	}
}

// standard ok response
func ok(text string) *smtpResponse {
	return response(221, text, telnet.REQUEST)
}

// quit response
func quit() *smtpResponse {
	return response(250, "Sayonara!", telnet.QUIT)
}

// creates the unavailable response, will quit
func unavailable() *smtpResponse {
	return response(550, "Server temporary unavailable", telnet.QUIT)
}

// bad sequence of commands
func badSequence(text string) *smtpResponse {
	return response(503, text, telnet.REQUEST)
}

// unknown command
func unknown() *smtpResponse {
	return response(504, "Command unrecognized", telnet.REQUEST)
}

// who has already been given
func alreadyMet(who string) *smtpResponse {
	return badSequence("We already met together, " + who)
}

// syntax error (missing argument, bad argument ...)
func syntaxError(text string) *smtpResponse {
	return response(501, text, telnet.REQUEST)
}

// wait for input
func input() *smtpResponse {
	return response(354, "enter mail, end with \".\" on a line by itself", telnet.DATA)
}
