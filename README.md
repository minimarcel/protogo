# Protogo

The goal of this project is to experiment in _golang_ the network package, and to implement different kinds of **Clients** and **Servers** in different protocols.


# Server

Actually, this is the only implemented part (for now).

Starting a new **Server** is pretty easy, just specify a `port` and give a `protogo.ServerHandler`, by using the method `protogo.Listen`.

```
server,err := protogo.Listen(port, &MyServerHandler{})
if err != nil {
	// Handle errors here
	return;
}
defer server.Stop()
```

The `protogo.ServerHandler` will handle the newly opened connections :

```
type ServerHandler interface {
	 // Will be invoked when a connection has been accepted
    OnAccepted(net.Conn)
}
```


## Telnet

The `telnet` package provides a **Telnet** implementation of the `prootgo.ServerHandler`, that will help you to create **Telnet Servers**.


### Starting a Telnet Server

You just have to start a **Server**, like we did before, by given the **Telnet** `ServerHandler`.

```
server,err := protogo.Listen(port, telnet.NewServer(welcome))
```

The `SeverHandler` is created by invoking `telnet.NewServer`, that accept only one argument, the `telnet.WelcomeHandler` in charge of warmly welcoming all new connections.
(This part is described bellow.)


### Telnet Requests

This implementation of a **Telnet Server** considere that you will encounter two types of requests : 

- A single `Line` request, representing commonly a `Command`
- A bunch of `Data`


### Telnet Responses

To each `Request`, you must give a `telnet.Response` that will be written to the connection outputstream, but most important, this `telnet.Response` contains the next `telnet.State` the Server should expect for : 

- `QUIT`: the server should quit by closing the connection
- `REQUEST` : the server should expect as next Request a `Line`
- `DATA` : the server should expect as next request a bunch of `Data`


```
type Response interface { 
    // write this response to the given writer
    WriteTo(writer *bufio.Writer)
    // returns the next expecting state
    Next() State
}
```


### Telnet WelcomeHandler

The `telnet.WelcomeHandler` is the entrypoint of each new connections, and must return a `telnet.Response` and the `EventHandler` that will handle all events attached to this connection.

Note that the `telnet.WelcomeHandler` is just a _function_, nothing more : 

```
type WelcomeHandler func() (Response,EventHandler)
```


### Telnet EventHandler

As described bellow, this handler is created by the `telnet.WelcomeHandler` on each new connection, and is in charge to handle all events (requests) for one connection, meaning the two kinds of requests : `Line` and `Data`.

```
type EventHandler interface {
    // handle line requests
    OnRequest(Line) Response
    // handle data request
    OnData(*Request) Response
}
```

If your **Telnet Server** doesn't need to have an instanciated struct at each new connection, you can just use `telnet.EventHandlerFrom(request, data)` that will create an `telnet.EventHandler` for you.

The two arguments `request` and `data` are two simple _functions_ : 

```
type RequestHandler func(Line)      (Response)
type DataHandler    func(*Request)  (Response)
```

### Examples

Two examples implementing a `Telnet Server` are provided :

- a simple **EchoServer**
- the implementation of the _SMTP_ protocol, with the **SMTPServer**
