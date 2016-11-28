package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/st3v/scope-garden/garden"
)

var (
	gardenNetwork string
	gardenAddr    string
	pluginsRoot   string
)

func init() {
	flag.StringVar(
		&gardenNetwork,
		"gardenNetwork",
		"unix",
		"network mode for garden server (tcp, unix)",
	)

	flag.StringVar(
		&gardenAddr,
		"gardenAddr",
		"/tmp/garden.sock",
		"network address for garden server",
	)

	flag.StringVar(
		&pluginsRoot,
		"pluginsRoot",
		"/var/run/scope/plugins",
		"root directory for scope plugin sockets",
	)
}

func main() {
	flag.Parse()

	hostID, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting on %s...\n", hostID)

	socket := filepath.Join(pluginsRoot, "garden", "garden.sock")

	listener, err := listen(socket)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		listener.Close()
		os.RemoveAll(filepath.Dir(socket))
	}()

	handleSignals()

	plugin := garden.NewPlugin(hostID, gardenNetwork, gardenAddr)

	http.HandleFunc("/report", plugin.Report)

	log.Fatal(http.Serve(listener, nil))
}

func listen(socket string) (net.Listener, error) {
	os.RemoveAll(filepath.Dir(socket))
	if err := os.MkdirAll(filepath.Dir(socket), 0700); err != nil {
		return nil, fmt.Errorf(
			"error creating directory %q: %v",
			filepath.Dir(socket),
			err,
		)
	}

	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("error listening on %q: %v", socket, err)
	}

	log.Printf("Listening on: unix://%s\n", socket)
	return listener, nil
}

func handleSignals() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interrupt
		os.Exit(0)
	}()
}
