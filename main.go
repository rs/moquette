package main

import (
	"crypto/tls"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/moquette/server"
)

func main() {
	hostname, _ := os.Hostname()
	debug := flag.Bool("debug", false, "Turn on debugging")
	broker := flag.String("broker", "tcp://127.0.0.1:1883", "The full url of the mqtt broker to connect to ex: tcp://127.0.0.1:1883")
	clientID := flag.String("client-id", hostname+strconv.Itoa(time.Now().Second()), "A client id for the connection")
	username := flag.String("username", "", "A username to authenticate to the mqtt server")
	password := flag.String("password", "", "Password to match username")
	confDir := flag.String("conf", "/etc/moquette.d", "Path to the configuration director")
	sep := flag.String("sep", ":", "File name separator used for topic separator (/)")
	flag.Parse()

	if *debug {
		mqtt.DEBUG = log.New(os.Stderr, "", 0)
	}
	mqtt.ERROR = log.New(os.Stderr, "", 0)

	connOpts := mqtt.NewClientOptions().
		AddBroker(*broker).
		SetClientID(*clientID).
		SetCleanSession(true).
		SetKeepAlive(2 * time.Second)
	if *username != "" {
		connOpts.SetUsername(*username)
		if *password != "" {
			connOpts.SetPassword(*password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	connOpts.SetOnConnectHandler(func(_ mqtt.Client) {
		log.Print("Connected")
	})

	s := server.New(connOpts, *confDir, *sep)
	stop := make(chan struct{})
	if err := s.Run(stop); err != nil {
		panic(err)
	}
}
