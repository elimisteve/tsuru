// Copyright 2013 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package queue implements all the queue handling with tsuru. It abstract
// which queue server is being used, how the message gets marshaled in to the
// wire and how it's read.
//
// It provides three functions: Put, Get and Delete, which puts, gets and
// deletes a message from the queue.
package queue

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/globocom/config"
	"github.com/kr/beanstalk"
	"io"
	"regexp"
	"time"
)

var (
	conn           *beanstalk.Conn
	timeoutRegexp  = regexp.MustCompile(`TIMED_OUT$`)
	notFoundRegexp = regexp.MustCompile(`not found$`)
)

// Message represents the message stored in the queue.
//
// A message is specified by an action and a slice of strings, representing
// arguments to the action.
//
// For example, the action "regenerate apprc" could receive one argument: the
// name of the app for which the apprc file will be regenerate.
type Message struct {
	Action string
	Args   []string
	id     uint64
}

// Put sends a new message to the queue.
func Put(msg *Message) error {
	conn, err := connection()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = gob.NewEncoder(&buf).Encode(msg)
	if err != nil {
		return err
	}
	id, err := conn.Put(buf.Bytes(), 1, 0, 60e9)
	msg.id = id
	return err
}

// Get retrieves a message from the queue.
func Get(timeout time.Duration) (*Message, error) {
	conn, err := connection()
	if err != nil {
		return nil, err
	}
	id, body, err := conn.Reserve(timeout)
	if err != nil {
		if timeoutRegexp.MatchString(err.Error()) {
			return nil, fmt.Errorf("Timed out waiting for message after %s.", timeout)
		}
		return nil, err
	}
	r := bytes.NewReader(body)
	var msg Message
	if err = gob.NewDecoder(r).Decode(&msg); err != nil && err != io.EOF {
		conn.Delete(id)
		return nil, fmt.Errorf("Invalid message: %q", body)
	}
	msg.id = id
	return &msg, nil
}

// Delete deletes a message from the queue. For deletion, the given message
// must be one returned by Get, or added by Put. This function uses internal
// state of the message to delete it (a message can not be deleted by its
// content).
func Delete(msg *Message) error {
	conn, err := connection()
	if err != nil {
		return err
	}
	if msg.id == 0 {
		return errors.New("Unknown message.")
	}
	if err = conn.Delete(msg.id); err != nil && notFoundRegexp.MatchString(err.Error()) {
		return errors.New("Message not found.")
	}
	return err
}

func connection() (*beanstalk.Conn, error) {
	var (
		addr string
		err  error
	)
	if conn == nil {
		addr, err = config.GetString("queue-server")
		if err != nil {
			return nil, errors.New(`"queue-server" is not defined in config file.`)
		}
		conn, err = beanstalk.Dial("tcp", addr)
	}
	return conn, err
}
