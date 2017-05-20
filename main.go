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
	"strconv"
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
	cfUsername            string
	cfPassword            string
	cfClientID            string
	cfClientSecret        string
	cfSkipSSLValidation   bool
	cfRefreshInterval     time.Duration
)

func init() {
	flag.StringVar(
		&gardenNetwork,
		"garden.network",
		getEnvString("GARDEN_NETWORK", "unix"),
		"network mode for garden server (tcp, unix) [GARDEN_NETWORK]",
	)

	flag.StringVar(
		&gardenAddr,
		"garden.addr",
		getEnvString("GARDEN_ADDR", "/tmp/garden.sock"),
		"network address for garden server [GARDEN_ADDR]",
	)

	flag.DurationVar(
		&gardenRefreshInterval,
		"garden.refresh-interval",
		getEnvDuration("GARDEN_REFRESH_INTERVAL", 3*time.Second),
		"interval to fetch for container updates from garden server [GARDEN_REFRESH_INTERVAL]",
	)

	flag.StringVar(
		&cfAPI,
		"cf.api-url",
		getEnvString("CF_API_URL", ""),
		"CF API endpoint to be used when looking up apps [CF_API_URL]",
	)

	flag.StringVar(
		&cfUsername,
		"cf.username",
		getEnvString("CF_USERNAME", ""),
		"username to be used when looking up apps in CF [CF_USERNAME]",
	)

	flag.StringVar(
		&cfPassword,
		"cf.password",
		getEnvString("CF_PASSWORD", ""),
		"password to be used when looking up apps in CF [CF_PASSWORD]",
	)

	flag.StringVar(
		&cfClientID,
		"cf.client-id",
		getEnvString("CF_CLIENT_ID", ""),
		"client ID to be used when looking up apps in CF [CF_CLIENT_ID]",
	)

	flag.StringVar(
		&cfClientSecret,
		"cf.client-secret",
		getEnvString("CF_CLIENT_SECRET", ""),
		"client secret to be used when looking up apps in CF [CF_CLIENT_SECRET]",
	)

	flag.BoolVar(
		&cfSkipSSLValidation,
		"cf.skip-ssl-verify",
		getEnvBool("CF_SKIP_SSL_VERIFY", false),
		"skip SSL validation when looking up apps in CF [CF_SKIP_SSL_VERIFY]",
	)

	flag.DurationVar(
		&cfRefreshInterval,
		"cf.refresh-interval",
		getEnvDuration("CF_REFRESH_INTERVAL", 3*time.Second),
		"interval to fetch for app updates from CF [CF_REFRESH_INTERVAL]",
	)

	flag.StringVar(
		&pluginsRoot,
		"plugins-root",
		getEnvString("PLUGINS_ROOT", "/var/run/scope/plugins"),
		"root directory for scope plugin sockets [PLUGINS_ROOT]",
	)

	flag.StringVar(
		&hostname,
		"hostname",
		getEnvString("HOSTNAME", ""),
		"hostname as reported by scope [HOSTNAME]",
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

	appDir := cf.NewAppDirectory(
		cfAPI,
		cfUsername,
		cfPassword,
		cfClientID,
		cfClientSecret,
		cfSkipSSLValidation,
		cfRefreshInterval,
	)
	defer appDir.Close()

	plugin := garden.NewPlugin(
		hostname,
		gardenNetwork,
		gardenAddr,
		gardenRefreshInterval,
		appDir.AppName,
	)
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

func getEnvString(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	return v
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}

	return d
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}

	return b
}
