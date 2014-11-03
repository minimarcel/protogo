package telnet

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

// ----------------------------------->
// The server

//
// The simple telnet server handler implementation
//
type Server struct {
	// the initial handler
	// invoked on each newly opened connection
	welcome WelcomeHandler
}

// Create a new TelnetServer handler with the welcome handler
// TODO manage options, like timeout (conn.SetDeadline)
func NewServer(welcome WelcomeHandler) *Server {
	return &Server{welcome}
}

// Handle accepted connections
func (telnet *Server) OnAccepted(conn net.Conn) {

	defer conn.Close()

	// creates the reader and writer
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	// create the request
	request := &Request{r}

	// first we have to warmly welcome this new connection
	response, handler := telnet.welcome()

	if handler == nil {
		log.Println("No handler given")
		return
	}

	// until we quit, handle the responses
	for {

		if response == nil {
			log.Println("Invalid empty response")
			return
		}

		// write the response
		response.WriteTo(w)
		w.Flush()

		// handle the next State
		switch response.Next() {

		// Quit this connection
		case QUIT:
			return

		// Waiting for a request
		case REQUEST:
			{
				line, err := request.NextLine()
				if err != nil {
					log.Println("Error while reading the next line")
					return
				}

				response = handler.OnRequest(*line)
			}

		// Waiting for data
		case DATA:
			{
				response = handler.OnData(request)
				break
			}
		}
	}
}

// ----------------------------------->
// The request handlers

// The entry point
// Must return the handler that will handle next requests
// and the response to this welcome connnection
type WelcomeHandler func() (Response, EventHandler)

// Handle events that may occurred on telnet server
// Each response will define if
type EventHandler interface {
	// handle line requests
	OnRequest(Line) Response
	// handle data request
	OnData(*Request) Response
}

// simple function handlers
// in case of the handler hasn't any context, or dosn't need to
type RequestHandler func(Line) Response
type DataHandler func(*Request) Response

// a default handler,
// that is created from the functions
type defaultHandler struct {
	request RequestHandler
	data    DataHandler
}

func EventHandlerFrom(request RequestHandler, data DataHandler) EventHandler {
	return &defaultHandler{request, data}
}

// implements the EventHandler welcome method
func (h *defaultHandler) OnRequest(line Line) Response {
	return h.request(line)
}

// implements the EventHandler request method
func (h *defaultHandler) OnData(req *Request) Response {
	return h.data(req)
}

// ----------------------------------->
// The requests

type Request struct {
	Reader *bufio.Reader
}

// Read the next line, on the request
func (req *Request) NextLine() (*Line, error) {

	line, err := req.Reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("Error while reading next line : %s", err)
	}

	return &Line{strings.TrimRight(line, "\r\n")}, nil
}

type Line struct {
	Value string
}

// Return this line as a command name (in upper case)
func (line *Line) AsCommand() Command {
	// parse the command line
	// TODO parse true command lines
	a := strings.Split(line.Value, " ")
	// return the Command object
	return Command{strings.ToLower(a[0]), a[1:]}
}

type Command struct {
	// the command name
	// always in lower case
	Name string
	// the arguments
	Args []string
}

// ----------------------------------->
// The responses

type State int

const (
	QUIT State = iota
	REQUEST
	DATA
)

//
// The response to a request
type Response interface {

	// write this response to the given writer
	WriteTo(writer *bufio.Writer)

	// returns the next expecting state
	Next() State
}

type defaultResponse struct {
	next State
}

func (resp *defaultResponse) Next() State {
	return resp.next
}

//
// A line response
// Will write a string line as response
type LineResponse struct {
	defaultResponse
	value string
}

// Creates a new LineResponse
func NewLineResponse(value string, next State) *LineResponse {
	return &LineResponse{defaultResponse{next}, value}
}

// Write this response
func (resp *LineResponse) WriteTo(writer *bufio.Writer) {
	// FIXME handle errors ?
	writer.WriteString(resp.value + "\n")
}
