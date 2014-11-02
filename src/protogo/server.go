package protogo

import (
    "net"
    "strconv"
    "fmt"
    "sync"
    "errors"
)

// -----------------------------------> 

// handle connections
// FIXME common handler or only server side ?
type ServerHandler interface {

    //
    // Will be invoked when a connection has been accepted
    //
    // @param   type    the message type
    // @param   reader  the message reader
    OnAccepted(conn net.Conn)
}

// the protogo Server
// will accept connections and handle requests
type Server struct {

    // on what is this server connected
    Address     string
    Port        int

    // accept new connections
    listener    net.Listener

    // determines whether this server is stopped
    stopped     bool

    // the opened connections list, 
    // and the max accepted connections
    conns       map[*serverConn]struct{}
    maxConns    int

    // protects conns, quit ...
    mutex       sync.Mutex

    // handle accepted connections
    handler     ServerHandler
}

// wrapp a net.Conn
type serverConn struct {

    // embeded struct
    net.Conn
    // the parent server
    parent *Server
}

// -----------------------------------> 

//
// Start this server by listning on the given port
//
func Listen(port int, handler ServerHandler) (*Server, error) {

    if handler == nil {
        return nil, errors.New("Can't create a server with an handler undefined")
    }

    if port <= 0 {
        return nil, fmt.Errorf("Can't create a server with a port negative or zero : %d", port)
    }

    // listen on the port
    ln,err := net.Listen("tcp", ":" + strconv.Itoa(port))
    if err != nil {
        // return this error
        return nil, fmt.Errorf("Unable to listen on port %d : %s", port, err)
    }

    // creates the server
    s := &Server{
        Address     : ln.Addr().String(),
        Port        : port,
        listener    : ln,
        stopped     : false,
        maxConns    : 100, // TODO options
        conns       : make(map[*serverConn]struct{}),
        handler     : handler,
    }

    // start serving
    go s.serve()

    return s,nil
}

//
// Stop this server
//
func (srv *Server) Stop() {

    // FIXME should we close all connections, 
    // or the listener.close will do it !?
    srv.stopped = true
    srv.listener.Close()
}

// -----------------------------------> 

// Accept connections
func (srv *Server) serve() {

    fmt.Println("Start serving requests")
    defer srv.Stop()

    for {

        conn, err := srv.listener.Accept()
        if err != nil {
            fmt.Println("Unable to accept a connection : ", err)
            continue
        }

        // wrapp the connection
        srvConn := srv.wrappConn(conn)
        if srvConn == nil {
           // no more connections are accepted
           // continue accepting util a slot got free
           continue
        }

        // handle the connection and continue accepting
        go srv.handleConn(srvConn)
    }
}

// wrap the connection to control the number of opened connections
func (srv *Server) wrappConn(conn net.Conn) *serverConn {

    // obtain a lock
    srv.mutex.Lock()
    defer srv.mutex.Unlock()

    // if the server is stopped, close all connections
    if srv.stopped {
        conn.Close()
        return nil
    }

    // some place lefts?
    if len(srv.conns) >= srv.maxConns {
        fmt.Printf("Refuse connection from %s, too much opened connections\n", conn.RemoteAddr().String())
        conn.Close()
        return nil
    }

    // creates the server conn
    srvConn := &serverConn{conn, srv}
    srv.conns[srvConn] = struct{}{}

    return srvConn
}

// handle this connection
func (srv *Server) handleConn(conn net.Conn) {

    fmt.Println("Accept connection from ", conn.RemoteAddr().String())
    srv.handler.OnAccepted(conn)
}

// -----------------------------------> 

//
// Release the connection on closing 
// (override the net.Conn close method)
//
func (conn *serverConn) Close() error {

    fmt.Println("Connection closed for ", conn.Conn.RemoteAddr().String())
    // release ourselves
    delete(conn.parent.conns, conn)
    // call the super
    return conn.Conn.Close()
}
