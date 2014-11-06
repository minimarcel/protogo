package main

import (
	"log"
	"os"
	"os/signal"

	"protogo"
	"protogo/telnet"
)

func main() {
	log.Printf("Starting echo server.\n")

	// Binds SIGTERM signal on this channel
	closeChannel := make(chan os.Signal, 1)
	signal.Notify(closeChannel, os.Interrupt)

	server, err := protogo.Listen(8100, telnet.NewServer(welcome))
	if err != nil {
		log.Fatalf("Unable to start the server : %s", err.Error())
	}

	defer func() {
		log.Printf("Stopping echo server.\n")
		server.Stop()
	}()

	// Server CONNECTED !!
	log.Printf("Server Connected on %s\n", server.Address)

	log.Println("Ctrl-C/SIGTERM to quit.")
	<-closeChannel
}

// ----------------------------------->

func welcome() (telnet.Response, telnet.EventHandler) {
	return telnet.NewLineResponse("Welcome!!!", telnet.REQUEST), telnet.EventHandlerFrom(echo, nil)
}

func echo(line telnet.Line) telnet.Response {

	switch line.AsCommand().Name {
	case "":
		return telnet.NewLineResponse("Please, say something!", telnet.REQUEST)
	case "quit", "exit", "bye":
		return telnet.NewLineResponse("Bye!", telnet.QUIT)
	}

	return telnet.NewLineResponse("You just said: "+line.Value, telnet.REQUEST)
}
