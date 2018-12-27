package main

import (
	"crypto/tls"
	"log"
	"os"
	"strconv"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"moquette/server"
)

func main() {
	hostname, _ := os.Hostname()

	flag.Bool("debug", false, "Turn on debugging")
	flag.String("broker", "tcp://127.0.0.1:1883", "The full url of the mqtt broker to connect to ex: tcp://127.0.0.1:1883")
	flag.String("client-id", hostname+strconv.Itoa(time.Now().Second()), "A client id for the connection")
	flag.String("username", "", "A username to authenticate to the mqtt server")
	flag.String("password", "", "Password to match username")
	flag.String("conf", "/etc/moquette.d", "Path to the configuration directory")
	flag.String("sep", ":", "File name separator used for topic separator (/)")
	flag.String("configfile", "", "Configuration file name to read options from.")

	flag.Parse()
	viper.BindPFlags(flag.CommandLine)

	explicitConfig := viper.GetString("configfile")

	if explicitConfig != "" {
		viper.SetConfigFile(explicitConfig)
		err := viper.ReadInConfig()
		if err != nil {
			log.Fatal("Fatal error when loading config file given on command line:\n", err)
		}
	} else {
		viper.SetConfigName("moquette")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.local/config/moquette")
		viper.AddConfigPath("/etc/moquette/")
		viper.AddConfigPath("/etc/")

		// With implicit config files, it's permissible not to have them at all.
		err := viper.ReadInConfig()

		if err != nil {
			if _, cnf := err.(viper.ConfigFileNotFoundError); cnf {
				log.Print("No config file found, using defaults and command line only.")
			} else {
				log.Fatal("Fatal error when loading config file:\n", err)
			}
		}
	}

	debug := viper.GetBool("debug")
	broker := viper.GetString("broker")
	clientID := viper.GetString("client-id")
	username := viper.GetString("username")
	password := viper.GetString("password")
	confDir := viper.GetString("conf")
	sep := viper.GetString("sep")

	if debug {
		mqtt.DEBUG = log.New(os.Stderr, "", 0)
	}
	mqtt.ERROR = log.New(os.Stderr, "", 0)

	connOpts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetCleanSession(true).
		SetKeepAlive(2 * time.Second)
	if username != "" {
		connOpts.SetUsername(username)
		if password != "" {
			connOpts.SetPassword(password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	connOpts.SetOnConnectHandler(func(_ mqtt.Client) {
		log.Print("Connected")
	})

	s := server.New(connOpts, confDir, sep)
	stop := make(chan struct{})
	if err := s.Run(stop); err != nil {
		panic(err)
	}
}
