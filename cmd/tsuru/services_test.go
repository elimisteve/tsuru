// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tsuru

import (
	"bytes"
	"github.com/globocom/tsuru/cmd"
	. "launchpad.net/gocheck"
	"net/http"
	"strings"
)

func (s *S) TestServiceList(c *C) {
	var stdout, stderr bytes.Buffer
	output := `[{"service": "mysql", "instances": ["mysql01", "mysql02"]}, {"service": "oracle", "instances": []}]`
	expectedPrefix := `+---------+------------------+
| Service | Instances        |`
	lineMysql := "| mysql   | mysql01, mysql02 |"
	lineOracle := "| oracle  |                  |"
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &conditionalTransport{
		transport{
			msg:    output,
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			return req.URL.Path == "/services/instances"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&ServiceList{}).Run(&ctx, client)
	c.Assert(err, IsNil)
	table := stdout.String()
	c.Assert(table, Matches, "^"+expectedPrefix+".*")
	c.Assert(table, Matches, "^.*"+lineMysql+".*")
	c.Assert(table, Matches, "^.*"+lineOracle+".*")
}

func (s *S) TestServiceListWithEmptyResponse(c *C) {
	var stdout, stderr bytes.Buffer
	output := "[]"
	expected := ""
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &conditionalTransport{
		transport{
			msg:    output,
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			return req.URL.Path == "/services/instances"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&ServiceList{}).Run(&ctx, client)
	c.Assert(err, IsNil)
	c.Assert(stdout.String(), Equals, expected)
}

func (s *S) TestInfoServiceList(c *C) {
	expected := &cmd.Info{
		Name:    "service-list",
		Usage:   "service-list",
		Desc:    "Get all available services, and user's instances for this services",
		MinArgs: 0,
	}
	command := &ServiceList{}
	c.Assert(command.Info(), DeepEquals, expected)
}

func (s *S) TestServiceListShouldBeInfoer(c *C) {
	var _ cmd.Infoer = &ServiceList{}
}

func (s *S) TestServiceListShouldBeCommand(c *C) {
	var _ cmd.Command = &ServiceList{}
}

func (s *S) TestServiceBind(c *C) {
	*AppName = "g1"
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	ctx := cmd.Context{
		Args:   []string{"my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &conditionalTransport{
		transport{
			msg:    `["DATABASE_HOST","DATABASE_USER","DATABASE_PASSWORD"]`,
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			called = true
			return req.Method == "PUT" && req.URL.Path == "/services/instances/my-mysql/g1"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&ServiceBind{}).Run(&ctx, client)
	c.Assert(err, IsNil)
	c.Assert(called, Equals, true)
	expected := `Instance my-mysql successfully binded to the app g1.

The following environment variables are now available for use in your app:

- DATABASE_HOST
- DATABASE_USER
- DATABASE_PASSWORD

For more details, please check the documentation for the service, using service-doc command.
`
	c.Assert(stdout.String(), Equals, expected)
}

func (s *S) TestServiceBindWithoutFlag(c *C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	ctx := cmd.Context{
		Args:   []string{"my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &conditionalTransport{
		transport{
			msg:    `["DATABASE_HOST","DATABASE_USER","DATABASE_PASSWORD"]`,
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			called = true
			return req.Method == "PUT" && req.URL.Path == "/services/instances/my-mysql/ge"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &FakeGuesser{name: "ge"}
	err := (&ServiceBind{GuessingCommand{G: fake}}).Run(&ctx, client)
	c.Assert(err, IsNil)
	c.Assert(called, Equals, true)
	expected := `Instance my-mysql successfully binded to the app ge.

The following environment variables are now available for use in your app:

- DATABASE_HOST
- DATABASE_USER
- DATABASE_PASSWORD

For more details, please check the documentation for the service, using service-doc command.
`
	c.Assert(stdout.String(), Equals, expected)
}

func (s *S) TestServiceBindWithRequestFailure(c *C) {
	*AppName = "g1"
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &transport{
		msg:    "This user does not have access to this app.",
		status: http.StatusForbidden,
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&ServiceBind{}).Run(&ctx, client)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, trans.msg)
}

func (s *S) TestServiceBindInfo(c *C) {
	expected := &cmd.Info{
		Name:  "bind",
		Usage: "bind <instancename> [--app appname]",
		Desc: `bind a service instance to an app

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
	c.Assert((&ServiceBind{}).Info(), DeepEquals, expected)
}

func (s *S) TestServiceBindIsAnInfoer(c *C) {
	var _ cmd.Infoer = &ServiceBind{}
}

func (s *S) TestServiceBindIsACommand(c *C) {
	var _ cmd.Command = &ServiceBind{}
}

func (s *S) TestServiceUnbind(c *C) {
	*AppName = "pocket"
	var stdout, stderr bytes.Buffer
	var called bool
	ctx := cmd.Context{
		Args:   []string{"hand"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &conditionalTransport{
		transport{
			msg:    "",
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			called = true
			return req.Method == "DELETE" && req.URL.Path == "/services/instances/hand/pocket"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&ServiceUnbind{}).Run(&ctx, client)
	c.Assert(err, IsNil)
	c.Assert(called, Equals, true)
	c.Assert(stdout.String(), Equals, "Instance hand successfully unbinded from the app pocket.\n")
}

func (s *S) TestServiceUnbindWithoutFlag(c *C) {
	var stdout, stderr bytes.Buffer
	var called bool
	ctx := cmd.Context{
		Args:   []string{"hand"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &conditionalTransport{
		transport{
			msg:    "",
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			called = true
			return req.Method == "DELETE" && req.URL.Path == "/services/instances/hand/sleeve"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &FakeGuesser{name: "sleeve"}
	err := (&ServiceUnbind{GuessingCommand{G: fake}}).Run(&ctx, client)
	c.Assert(err, IsNil)
	c.Assert(called, Equals, true)
	c.Assert(stdout.String(), Equals, "Instance hand successfully unbinded from the app sleeve.\n")
}

func (s *S) TestServiceUnbindWithRequestFailure(c *C) {
	*AppName = "pocket"
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"hand"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &transport{
		msg:    "This app is not binded to this service.",
		status: http.StatusPreconditionFailed,
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&ServiceUnbind{}).Run(&ctx, client)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, trans.msg)
}

func (s *S) TestServiceUnbindInfo(c *C) {
	expected := &cmd.Info{
		Name:  "unbind",
		Usage: "unbind <instancename> [--app appname]",
		Desc: `unbind a service instance from an app

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
	c.Assert((&ServiceUnbind{}).Info(), DeepEquals, expected)
}

func (s *S) TestServiceUnbindIsAnInfoer(c *C) {
	var _ cmd.Infoer = &ServiceUnbind{}
}

func (s *S) TestServiceUnbindIsAComand(c *C) {
	var _ cmd.Command = &ServiceUnbind{}
}

func (s *S) TestServiceAddInfo(c *C) {
	usage := `service-add <servicename> <serviceinstancename>
e.g.:

    $ tsuru service-add mongodb tsuru_mongodb

Will add a new instance of the "mongodb" service, named "tsuru_mongodb".`
	expected := &cmd.Info{
		Name:    "service-add",
		Usage:   usage,
		Desc:    "Create a service instance to one or more apps make use of.",
		MinArgs: 2,
	}
	command := &ServiceAdd{}
	c.Assert(command.Info(), DeepEquals, expected)
}

func (s *S) TestServiceAddRun(c *C) {
	var stdout, stderr bytes.Buffer
	result := "Service successfully added.\n"
	args := []string{
		"my_app_db",
		"mysql",
	}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: result, status: http.StatusOK}}, nil, manager)
	err := (&ServiceAdd{}).Run(&context, client)
	c.Assert(err, IsNil)
	obtained := stdout.String()
	c.Assert(obtained, Equals, result)
}

func (s *S) TestServiceInstanceStatusInfo(c *C) {
	usg := `service-status <serviceinstancename>
e.g.:

    $ tsuru service-status my_mongodb
`
	expected := &cmd.Info{
		Name:    "service-status",
		Usage:   usg,
		Desc:    "Check status of a given service instance.",
		MinArgs: 1,
	}
	got := (&ServiceInstanceStatus{}).Info()
	c.Assert(got, DeepEquals, expected)
}

func (s *S) TestServiceInstanceStatusRun(c *C) {
	var stdout, stderr bytes.Buffer
	result := `Service instance "foo" is up`
	args := []string{"fooBar"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: result, status: http.StatusOK}}, nil, manager)
	err := (&ServiceInstanceStatus{}).Run(&context, client)
	c.Assert(err, IsNil)
	obtained := stdout.String()
	obtained = strings.Replace(obtained, "\n", "", -1)
	c.Assert(obtained, Equals, result)
}

func (s *S) TestServiceInfoInfo(c *C) {
	usg := `service-info <service>
e.g.:

    $ tsuru service-info mongodb
`
	expected := &cmd.Info{
		Name:    "service-info",
		Usage:   usg,
		Desc:    "List all instances of a service",
		MinArgs: 1,
	}
	got := (&ServiceInfo{}).Info()
	c.Assert(got, DeepEquals, expected)
}

func (s *S) TestServiceInfoRun(c *C) {
	var stdout, stderr bytes.Buffer
	result := `[{"Name":"mymongo", "Apps":["myapp"]}]`
	expected := `Info for "mongodb"
+-----------+-------+
| Instances | Apps  |
+-----------+-------+
| mymongo   | myapp |
+-----------+-------+
`
	args := []string{"mongodb"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: result, status: http.StatusOK}}, nil, manager)
	err := (&ServiceInfo{}).Run(&context, client)
	c.Assert(err, IsNil)
	obtained := stdout.String()
	c.Assert(obtained, Equals, expected)
}

func (s *S) TestServiceDocInfo(c *C) {
	i := (&ServiceDoc{}).Info()
	expected := &cmd.Info{
		Name:    "service-doc",
		Usage:   "service-doc <servicename>",
		Desc:    "Show documentation of a service",
		MinArgs: 1,
	}
	c.Assert(i, DeepEquals, expected)
}

func (s *S) TestServiceDocRun(c *C) {
	var stdout, stderr bytes.Buffer
	result := `This is a test doc for a test service.
Service test is foo bar.
`
	expected := result
	ctx := cmd.Context{
		Args:   []string{"foo"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: result, status: http.StatusOK}}, nil, manager)
	err := (&ServiceDoc{}).Run(&ctx, client)
	c.Assert(err, IsNil)
	obtained := stdout.String()
	c.Assert(obtained, Equals, expected)
}

func (s *S) TestServiceRemoveInfo(c *C) {
	i := (&ServiceRemove{}).Info()
	expected := &cmd.Info{
		Name:    "service-remove",
		Usage:   "service-remove <serviceinstancename>",
		Desc:    "Removes a service instance",
		MinArgs: 1,
	}
	c.Assert(i, DeepEquals, expected)
}

func (s *S) TestServiceRemoveRun(c *C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"some-service-instance"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	result := "service instance successfuly removed"
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: result, status: http.StatusOK}}, nil, manager)
	err := (&ServiceRemove{}).Run(&ctx, client)
	c.Assert(err, IsNil)
	obtained := stdout.String()
	c.Assert(obtained, Equals, result+"\n")
}
