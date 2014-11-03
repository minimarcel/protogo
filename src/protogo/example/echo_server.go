package main

import (
    "fmt"
    "protogo"
    "protogo/telnet"
)

func main() {

    fmt.Printf("Start echo server.\n")

    server,err := protogo.Listen(8100, telnet.NewServer(welcome))
    if err != nil {
        fmt.Println("Unable to start the server %s", err);
        return
    }

    defer server.Stop()

    // Server CONNECTED !!
    fmt.Printf("Server Connected on %s\n", server.Address)

    // FIXME how to not wait for an input !?
    fmt.Println("Type a key to quit ...")
    fmt.Scanln()
    fmt.Println("Stopped")
}

// -----------------------------------> 

func welcome() (telnet.Response, telnet.EventHandler) {
    return telnet.NewLineResponse("Welcome!!!", telnet.REQUEST), telnet.EventHandlerFrom(echo, nil)
}

func echo(line telnet.Line) telnet.Response {

    switch(line.AsCommand().Name) {
        case "" : return telnet.NewLineResponse("Please, say something!", telnet.REQUEST)
        case "quit", "exit", "bye" : return telnet.NewLineResponse("Bye!", telnet.QUIT)
    }

    return telnet.NewLineResponse("You just said: " + line.Value, telnet.REQUEST)
}
