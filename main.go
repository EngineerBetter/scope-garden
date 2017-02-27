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
	"time"

	"github.com/st3v/scope-garden/cf"
	"github.com/st3v/scope-garden/garden"
)

var (
	gardenNetwork         string
	gardenAddr            string
	gardenRefreshInterval time.Duration
	pluginsRoot           string
	hostname              string
	cfAPI                 string
	cfClientID            string
	cfClientSecret        string
	cfSkipSSLValidation   bool
	cfRefreshInterval     time.Duration
)

func init() {
	flag.StringVar(
		&gardenNetwork,
		"garden.network",
		"unix",
		"network mode for garden server (tcp, unix)",
	)

	flag.StringVar(
		&gardenAddr,
		"garden.addr",
		"/tmp/garden.sock",
		"network address for garden server",
	)

	flag.DurationVar(
		&gardenRefreshInterval,
		"garden.refresh-interval",
		3*time.Second,
		"interval to fetch for container updates from garden server",
	)

	flag.StringVar(
		&cfAPI,
		"cf.api-url",
		"",
		"CF API endpoint to be used when looking up apps, optional",
	)

	flag.StringVar(
		&cfClientID,
		"cf.client-id",
		"",
		"client ID to be used when looking up apps in CF, optional",
	)

	flag.StringVar(
		&cfClientSecret,
		"cf.client-secret",
		"",
		"client secret to be used when looking up apps in CF, optional",
	)

	flag.BoolVar(
		&cfSkipSSLValidation,
		"cf.skip-ssl-verify",
		false,
		"skip SSL validation when looking up apps in CF, optional",
	)

	flag.DurationVar(
		&cfRefreshInterval,
		"cf.refresh-interval",
		3*time.Second,
		"interval to fetch for app updates from CF",
	)

	flag.StringVar(
		&pluginsRoot,
		"plugins-root",
		"/var/run/scope/plugins",
		"root directory for scope plugin sockets",
	)

	flag.StringVar(
		&hostname,
		"hostname",
		"",
		"hostname as reported by scope",
	)
}

func main() {
	flag.Parse()

	if hostname == "" {
		var err error
		hostname, err = os.Hostname()
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("Starting on %s...\n", hostname)

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

	appDir := cf.NewAppDirectory(cfAPI, cfClientID, cfClientSecret, cfSkipSSLValidation, cfRefreshInterval)
	defer appDir.Close()

	plugin := garden.NewPlugin(hostname, gardenNetwork, gardenAddr, gardenRefreshInterval, appDir.AppName)
	defer plugin.Close()

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

	log.Printf("Listening on unix://%s\n", socket)
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
