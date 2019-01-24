package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type protoReader struct {
	buf *bufio.Reader
}

type command interface {
	String() string
}

type event struct {
	Topic   string
	QoS     byte
	Payload []byte
}

func (ev event) String() string {
	return fmt.Sprintf("PUB %s %v: %s", ev.Topic, ev.QoS, ev.Payload)
}

type kill struct {
	Topic string
}

func (k kill) String() string {
	return fmt.Sprintf("KILL %s", k.Topic)
}

func newProtoReader(r io.Reader) protoReader {
	return protoReader{bufio.NewReader(r)}
}

func (p protoReader) Next() (command, error) {
	cmd, err := p.buf.ReadString(' ')
	if err != nil {
		return nil, err
	}
	cmd = strings.ToUpper(strings.TrimSpace(cmd))
	switch cmd {
	case "PUB":
		return p.parsePub()
	case "KILL":
		return p.parseKill()
	default:
		return nil, fmt.Errorf("invalid command (%s)", cmd)
	}
}

func (p protoReader) parsePub() (ev event, err error) {
	if ev.Topic, err = p.buf.ReadString(' '); err != nil {
		return ev, fmt.Errorf("can't parse topic: %v", err)
	}
	ev.Topic = strings.TrimSpace(ev.Topic)
	if _, err := fmt.Fscanf(p.buf, "%d", &ev.QoS); err != nil {
		return ev, fmt.Errorf("can't parse QoS: %v", err)
	}
	if ev.QoS < 0 || ev.QoS > 3 {
		return ev, fmt.Errorf("invalid QoS: %v", ev.QoS)
	}
	if b, err := p.buf.ReadByte(); err != nil {
		return ev, err
	} else if b != ' ' {
		return ev, fmt.Errorf("expect space, got: %q", string(b))
	}
	var l int
	if _, err := fmt.Fscanf(p.buf, "%d", &l); err != nil {
		return ev, fmt.Errorf("can't parse payload length: %v", err)
	}
	if b, err := p.buf.ReadByte(); err != nil {
		return ev, err
	} else if b != '\n' {
		return ev, fmt.Errorf("expect EOL, got: %q", string(b))
	}
	ev.Payload = make([]byte, l)
	if n, err := p.buf.Read(ev.Payload); err != nil {
		return ev, fmt.Errorf("can't parse payload: %v", err)
	} else if n < l {
		return ev, fmt.Errorf("payload too short: expected %d, got %d", l, n)
	}
	if b, _ := p.buf.Peek(1); len(b) == 1 && b[0] == '\n' {
		// Clean optional return after payload
		p.buf.Discard(1)
	}
	return
}

func (p protoReader) parseKill() (k kill, err error) {
	if k.Topic, err = p.buf.ReadString('\n'); err != nil {
		return k, fmt.Errorf("can't parse topic: %v", err)
	}
	k.Topic = strings.TrimSpace(k.Topic)
	return
}
