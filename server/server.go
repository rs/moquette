package server

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/moquette/router"
)

type Server struct {
	conf   string
	sep    string
	client mqtt.Client
	procs  map[*os.Process]string // proc -> topic
	mu     sync.RWMutex
}

func New(mqttOpts *mqtt.ClientOptions, confDir, sep string) *Server {
	s := &Server{
		conf:  confDir,
		sep:   sep,
		procs: map[*os.Process]string{},
	}

	messageHandler := func(_ mqtt.Client, msg mqtt.Message) {
		go s.handleMessage(msg)
	}
	mqttOpts.SetOnConnectHandler(func(c mqtt.Client) {
		if token := c.Subscribe("#", byte(0), messageHandler); token.Wait() && token.Error() != nil {
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

func (s *Server) inputHandler(p *os.Process, r io.Reader) {
	proto := newProtoReader(r)
	for {
		cmd, err := proto.Next()
		if err != nil {
			if _, ok := err.(*os.PathError); !ok && err != io.EOF {
				log.Printf("invalid input: %v", err)
			}
			break
		}
		log.Print(cmd)
		switch t := cmd.(type) {
		case event:
			s.client.Publish(t.Topic, t.QoS, false, t.Payload)
		case kill:
			s.kill(t.Topic, p)
		}
	}
}

func (s *Server) handleMessage(msg mqtt.Message) {
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
	p := string(msg.Payload())
	c := exec.Command(cmd, p)
	c.Dir = s.conf
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.ExtraFiles = []*os.File{w}
	c.Env = []string{
		fmt.Sprintf("MQTT_TOPIC=%s", msg.Topic()),
		fmt.Sprintf("MQTT_MSGID=%d", msg.MessageID()),
	}
	if err := c.Start(); err != nil {
		log.Printf("%s: %v", cmd, err)
	}
	go s.inputHandler(c.Process, r)
	log.Printf("executing %s %s (pid: %d)", cmd, p, c.Process.Pid)
	s.addProc(c.Process, topic)
	defer s.removeProc(c.Process)
	if err := c.Wait(); err != nil {
		log.Printf("%s: %v", cmd, err)
	}
}

func (s *Server) addProc(p *os.Process, topic string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.procs[p] = topic
}

func (s *Server) removeProc(p *os.Process) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.procs, p)
}

func (s *Server) kill(topic string, except *os.Process) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for p, t := range s.procs {
		if t == topic && (except == nil || except.Pid != p.Pid) {
			p.Kill()
		}
	}
}
