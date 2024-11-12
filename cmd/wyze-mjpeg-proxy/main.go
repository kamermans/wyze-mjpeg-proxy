package main

import (
	"fmt"
	"net/http"
	"os"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var (
	configFile  = ""
	verbose     = false
	veryVerbose = false

	cfg Config
)

type StreamConfig struct {
	Name       string `yaml:"name"`
	Source     string `yaml:"source"`
	Resolution string `yaml:"resolution"`
	Quality    int    `yaml:"quality"`
	Framerate  int    `yaml:"framerate"`
}

type Config struct {
	Verbosity int            `yaml:"verbosity"`
	Addr      string         `yaml:"addr"`
	Port      int            `yaml:"port"`
	Streams   []StreamConfig `yaml:"streams"`
}

func init() {
	flag.StringVarP(&configFile, "config", "c", configFile, "Config file")
	flag.Parse()

	if configFile != "" {
		fmt.Printf("Loading config from %s\n", configFile)

		// Load config
		configBytes, err := os.ReadFile(configFile)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Parse config
		err = yaml.Unmarshal(configBytes, &cfg)
		if err != nil {
			fmt.Printf("Error parsing config: %v\n", err)
			os.Exit(1)
		}

		if cfg.Verbosity >= 1 {
			verbose = true
		}

		if cfg.Verbosity >= 2 {
			veryVerbose = true
		}
	}
}

func main() {

	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("Listening on %s\n", addr)

	for _, stream := range cfg.Streams {
		fmt.Printf("Setting up stream: %s\n", stream.Name)

		// Setup stream
		s := NewStreamer(stream)

		http.Handle(fmt.Sprintf("/%s/stream.mjpg", stream.Name), accessLog(http.HandlerFunc(s.StreamHandler)))
		http.Handle(fmt.Sprintf("/%s/image.jpg", stream.Name), accessLog(http.HandlerFunc(s.ImageHandler)))
		s.Start()
	}

	http.ListenAndServe(addr, nil)
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if verbose {
		fmt.Printf("%s %v %v %q\n", r.RemoteAddr, r.Method, r.URL.Path, r.UserAgent())
		// }
		next.ServeHTTP(w, r)
	})
}
