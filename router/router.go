package router

import (
	"errors"
	"io/ioutil"
	"path"
	"strings"

	"golang.org/x/sys/unix"
)

var ErrNotFound = errors.New("not found")

type Router struct {
	Dir string
	Sep string
}

func (r Router) Match(topic string) (string, error) {
	files, err := ioutil.ReadDir(r.Dir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if !file.Mode().IsRegular() || file.Mode().Perm()&unix.S_IXUSR == 0 {
			// Skip non-executable files
			continue
		}

		route := strings.Replace(file.Name(), r.Sep, "/", -1)
		if route == topic || routeIncludesTopic(route, topic) {
			return path.Join(r.Dir, file.Name()), nil
		}
	}

	return "", ErrNotFound
}

func routeIncludesTopic(route, topic string) bool {
	return match(strings.Split(route, "/"), strings.Split(topic, "/"))
}

// match takes a slice of strings which represent the route being tested having been split on '/'
// separators, and a slice of strings representing the topic string in the published message, similarly
// split.
// The function determines if the topic string matches the route according to the MQTT topic rules
// and returns a boolean of the outcome
func match(route []string, topic []string) bool {
	if len(route) == 0 {
		if len(topic) == 0 {
			return true
		}
		return false
	}

	if len(topic) == 0 {
		if route[0] == "#" {
			return true
		}
		return false
	}

	if route[0] == "#" {
		return true
	}

	if (route[0] == "+") || (route[0] == topic[0]) {
		return match(route[1:], topic[1:])
	}
	return false
}
