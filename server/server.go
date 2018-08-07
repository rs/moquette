package server

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/moquette/router"
)

type Server struct {
	conf   string
	sep    string
	client mqtt.Client
}

func New(mqttOpts *mqtt.ClientOptions, confDir, sep string) *Server {
	s := &Server{
		conf: confDir,
		sep:  sep,
	}

	mqttOpts.SetOnConnectHandler(func(c mqtt.Client) {
		if token := c.Subscribe("#", byte(0), s.messageHandler); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}
	})
	s.client = mqtt.NewClient(mqttOpts)

	return s
}

func (s *Server) Run(stop chan struct{}) error {
	if token := s.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	<-stop

	return nil
}

func (s *Server) publishHandler(r io.Reader) {
	proto := newProtoReader(r)
	for {
		ev, err := proto.Next()
		if err != nil {
			if _, ok := err.(*os.PathError); !ok && err != io.EOF {
				log.Printf("invalid input: %v", err)
			}
			break
		}
		// log.Print("PUB ", ev)
		s.client.Publish(ev.Topic, ev.QoS, false, ev.Payload)
	}
	log.Print("closing publish handler")
}

func (s *Server) messageHandler(_ mqtt.Client, msg mqtt.Message) {
	rt := router.Router{
		Dir: s.conf,
		Sep: s.sep,
	}
	topic := msg.Topic()
	cmd, err := rt.Match(topic)
	if err == router.ErrNotFound || cmd == "" {
		return
	}
	if err != nil {
		log.Print("can't route message: ", err)
		return
	}
	r, w, err := os.Pipe()
	if err != nil {
		log.Print("can't create pipe: ", err)
		return
	}
	defer r.Close()
	defer w.Close()
	go s.publishHandler(r)
	c := exec.Command(cmd, string(msg.Payload()))
	c.Dir = s.conf
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.ExtraFiles = []*os.File{w}
	c.Env = []string{
		fmt.Sprintf("MQTT_TOPIC=%s", msg.Topic()),
		fmt.Sprintf("MQTT_MSGID=%d", msg.MessageID()),
	}
	if err := c.Run(); err != nil {
		log.Printf("%s: %v", cmd, err)
	}
	log.Print("handler exec done")
}
