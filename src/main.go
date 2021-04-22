package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/nkanaev/yarr/src/platform"
	"github.com/nkanaev/yarr/src/server"
	"github.com/nkanaev/yarr/src/storage"
)

const APP = "yarr"

var Version string = "0.0"
var GitHash string = "unknown"

type Config struct {
	Address     string
	Database    string
	AuthFile    string
	CertFile    string
	KeyFile     string
	BasePath    string
	LogPath     string
	OpenBrowser bool
}

func main() {
	config := Config{}
	var ver bool
	flag.StringVar(&config.Address, "addr", "127.0.0.1:7070", "address to run server on")
	flag.StringVar(&config.AuthFile, "auth-file", "", "path to a file containing username:password")
	flag.StringVar(&config.BasePath, "base", "", "base path of the service url")
	flag.StringVar(&config.CertFile, "cert-file", "", "path to cert file for https")
	flag.StringVar(&config.KeyFile, "key-file", "", "path to key file for https")
	flag.StringVar(&config.Database, "db", "", "storage file path")
	flag.StringVar(&config.LogPath, "log", "", "log path")
	flag.BoolVar(&config.OpenBrowser, "open", false, "open the server in browser")
	flag.BoolVar(&ver, "version", false, "print application version")
	flag.Parse()

	if ver {
		fmt.Printf("v%s (%s)\n", Version, GitHash)
		return
	}

	err := envconfig.Process(APP, &config)
	if err != nil {
		log.Fatal("Failed to get config from env: ", err)
	}
	if config.LogPath != "" {
		f, err := os.OpenFile(config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(f)
	} else {
		log.SetOutput(os.Stdout)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	configPath, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Failed to get config dir: ", err)
	}

	if config.Database == "" {
		storagePath := filepath.Join(configPath, APP)
		if err := os.MkdirAll(storagePath, 0755); err != nil {
			log.Fatal("Failed to create app config dir: ", err)
		}
		config.Database = filepath.Join(storagePath, "storage.db")
	}

	log.Printf("using db file %s", config.Database)

	var username, password string
	if config.AuthFile != "" {
		f, err := os.Open(config.AuthFile)
		if err != nil {
			log.Fatal("Failed to open auth file: ", err)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, ":")
			if len(parts) != 2 {
				log.Fatalf("Invalid auth: %v (expected `username:password`)", line)
			}
			username = parts[0]
			password = parts[1]
			break
		}
	}

	if (config.CertFile != "" || config.KeyFile != "") && (config.CertFile == "" || config.KeyFile == "") {
		log.Fatalf("Both cert & key files are required")
	}

	store, err := storage.New(config.Database)
	if err != nil {
		log.Fatal("Failed to initialise database: ", err)
	}

	srv := server.NewServer(store, config.Address)

	if config.BasePath != "" {
		srv.BasePath = "/" + strings.Trim(config.BasePath, "/")
	}

	if config.CertFile != "" && config.KeyFile != "" {
		srv.CertFile = config.CertFile
		srv.KeyFile = config.KeyFile
	}

	if username != "" && password != "" {
		srv.Username = username
		srv.Password = password
	}

	log.Printf("starting server at %s", srv.GetAddr())
	if config.OpenBrowser {
		platform.Open(srv.GetAddr())
	}
	platform.Start(srv)
}
