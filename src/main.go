package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nkanaev/yarr/src/platform"
	"github.com/nkanaev/yarr/src/server"
	"github.com/nkanaev/yarr/src/storage"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const APP = "yarr"

var Version string = "0.0"
var GitHash string = "unknown"

type Config struct {
	Addr     string `mapstructure:"addr"`
	AuthFile string `mapstructure:"auth-file"`
	Base     string `mapstructure:"base"`
	CertFile string `mapstructure:"cert-file"`
	KeyFile  string `mapstructure:"key-file"`
	DB       string `mapstructure:"db"`
	Open     bool   `mapstructure:"open"`
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	pflag.String("addr", "127.0.0.1:7070", "address to run server on")
	pflag.String("auth-file", "", "path to a file containing username:password")
	pflag.String("base", "", "base path of the service url")
	pflag.String("cert-file", "", "path to cert file for https")
	pflag.String("key-file", "", "path to key file for https")
	pflag.String("db", "", "storage file path")
	pflag.Bool("open", false, "open the server in browser")
	pflag.BoolP("help", "h", false, "")
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s v%s (%s):\n", os.Args[0], Version, GitHash)
		pflag.PrintDefaults()
	}
	pflag.Parse()

	ok, _ := pflag.CommandLine.GetBool("help")
	if ok {
		pflag.Usage()
		return
	}

	configPath, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Failed to get config dir: ", err)
	}
	storagePath := filepath.Join(configPath, APP)
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		log.Fatal("Failed to create app config dir: ", err)
	}

	v := viper.New()
	v.SetConfigName(APP)
	v.AddConfigPath(".")
	v.AddConfigPath(storagePath)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatal("Failed to get config file: ", err)
		}
	}
	v.SetEnvPrefix(APP)
	v.AutomaticEnv()
	v.BindPFlags(pflag.CommandLine)
	v.SetDefault("db", filepath.Join(storagePath, "storage.db"))

	config := &Config{}
	err = v.Unmarshal(config)
	if err != nil {
		log.Fatal("Failed to get config: ", err)
	}

	log.Printf("using config %#v", config)

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

	store, err := storage.New(config.DB)
	if err != nil {
		log.Fatal("Failed to initialise database: ", err)
	}

	srv := server.NewServer(store, config.Addr)

	if config.Base != "" {
		srv.BasePath = "/" + strings.Trim(config.Base, "/")
	}

	if config.CertFile != "" || config.KeyFile != "" {
		srv.CertFile = config.CertFile
		srv.KeyFile = config.KeyFile
	}

	if username != "" && password != "" {
		srv.Username = username
		srv.Password = password
	}

	log.Printf("starting server at %s", srv.GetAddr())
	if config.Open {
		platform.Open(srv.GetAddr())
	}
	platform.Start(srv)
}
