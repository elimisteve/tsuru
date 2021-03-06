// Copyright 2013 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import (
	"bytes"
	"encoding/gob"
	"github.com/globocom/config"
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type S struct{}

var _ = Suite(&S{})

func (s *S) SetUpSuite(c *C) {
	config.Set("queue-server", "127.0.0.1:11300")

	// Cleaning the queue. All tests must clean its mess, but we can't
	// guarante the state of the queue before running them.
	cn, err := connection()
	c.Assert(err, IsNil)
	var id uint64
	for err == nil {
		if id, _, err = cn.Reserve(1e6); err == nil {
			err = cn.Delete(id)
		}
	}
}

func (s *S) SetUpTest(c *C) {
	conn = nil
}

func (s *S) TestConnection(c *C) {
	cn, err := connection()
	c.Assert(err, IsNil)
	defer cn.Close()
	tubes, err := cn.ListTubes()
	c.Assert(err, IsNil)
	c.Assert(tubes, DeepEquals, []string{"default"})
}

func (s *S) TestConnectionQueueServerUndefined(c *C) {
	old, _ := config.Get("queue-server")
	config.Unset("queue-server")
	defer config.Set("queue-server", old)
	conn, err := connection()
	c.Assert(conn, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, `"queue-server" is not defined in config file.`)
}

func (s *S) TestConnectDoubleCall(c *C) {
	cn1, err := connection()
	c.Assert(err, IsNil)
	defer cn1.Close()
	c.Assert(cn1, Equals, conn)
	cn2, err := connection()
	c.Assert(err, IsNil)
	c.Assert(cn2, Equals, cn1)
}

func (s *S) TestPut(c *C) {
	msg := Message{
		Action: "regenerate-apprc",
		Args:   []string{"myapp"},
	}
	err := Put(&msg)
	c.Assert(err, IsNil)
	c.Assert(msg.id, Not(Equals), 0)
	defer conn.Delete(msg.id)
	id, body, err := conn.Reserve(1e6)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, msg.id)
	var got Message
	buf := bytes.NewBuffer(body)
	err = gob.NewDecoder(buf).Decode(&got)
	c.Assert(err, IsNil)
	got.id = msg.id
	c.Assert(got, DeepEquals, msg)
}

func (s *S) TestPutConnectionFailure(c *C) {
	old, _ := config.Get("queue-server")
	defer config.Set("queue-server", old)
	config.Unset("queue-server")
	msg := Message{Action: "regenerate-apprc"}
	err := Put(&msg)
	c.Assert(err, NotNil)
}

func (s *S) TestGet(c *C) {
	msg := Message{
		Action: "regenerate-apprc",
		Args:   []string{"myapprc"},
	}
	err := Put(&msg)
	c.Assert(err, IsNil)
	defer conn.Delete(msg.id)
	got, err := Get(1e6)
	c.Assert(err, IsNil)
	c.Assert(*got, DeepEquals, msg)
}

func (s *S) TestGetConnectionError(c *C) {
	old, _ := config.Get("queue-server")
	defer config.Set("queue-server", old)
	config.Unset("queue-server")
	msg, err := Get(1e6)
	c.Assert(msg, IsNil)
	c.Assert(err, NotNil)
}

func (s *S) TestGetFromEmptyQueue(c *C) {
	msg, err := Get(1e6)
	c.Assert(msg, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Timed out waiting for message after 1ms.")
}

func (s *S) TestGetInvalidMessage(c *C) {
	conn, err := connection()
	c.Assert(err, IsNil)
	id, err := conn.Put([]byte("hello world"), 1, 0, 10e9)
	defer conn.Delete(id) // sanity
	msg, err := Get(1e6)
	c.Assert(msg, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, `Invalid message: "hello world"`)
	_, _, err = conn.Reserve(1e6)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^.*TIMED_OUT$")
}

func (s *S) TestDelete(c *C) {
	msg := Message{
		Action: "create-app",
		Args:   []string{"something"},
	}
	err := Put(&msg)
	c.Assert(err, IsNil)
	defer conn.Delete(msg.id)
	err = Delete(&msg)
	c.Assert(err, IsNil)
}

func (s *S) TestDeleteConnectionError(c *C) {
	old, _ := config.Get("queue-server")
	defer config.Set("queue-server", old)
	config.Unset("queue-server")
	err := Delete(nil)
	c.Assert(err, NotNil)
}

func (s *S) TestDeleteUnknownMessage(c *C) {
	msg := Message{
		Action: "create-app",
		Args:   []string{"something"},
		id:     837826742,
	}
	err := Delete(&msg)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Message not found.")
}

func (s *S) TestDeleteMessageWithoutId(c *C) {
	msg := Message{
		Action: "create-app",
		Args:   []string{"something"},
	}
	err := Delete(&msg)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Unknown message.")
}
