// Copyright 2013 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tsuru

import (
	"errors"
	"io"
	. "launchpad.net/gocheck"
	"os"
	"os/exec"
	"path"
	"syscall"
)

func writeConfig(sourceFile string, c *C) string {
	srcConfig, err := os.Open(sourceFile)
	c.Assert(err, IsNil)
	defer srcConfig.Close()
	p := path.Join(os.TempDir(), "guesser-tests")
	err = os.MkdirAll(p, 0700)
	c.Assert(err, IsNil)
	cmd := exec.Command("git", "init")
	cmd.Dir = p
	c.Assert(err, IsNil)
	err = cmd.Run()
	c.Assert(err, IsNil)
	dstConfig, err := os.OpenFile(path.Join(p, ".git", "config"), syscall.O_WRONLY|syscall.O_TRUNC|syscall.O_CREAT|syscall.O_CLOEXEC, 0644)
	c.Assert(err, IsNil)
	defer dstConfig.Close()
	_, err = io.Copy(dstConfig, srcConfig)
	c.Assert(err, IsNil)
	return p
}

func (s *S) TestGitGuesser(c *C) {
	p := writeConfig("testdata/gitconfig-ok", c)
	defer os.RemoveAll(p)
	dirPath := path.Join(p, "somepath")
	err := os.MkdirAll(dirPath, 0700) // Will be removed when p is removed.
	c.Assert(err, IsNil)
	g := GitGuesser{}
	name, err := g.GuessName(p) // repository root
	c.Assert(err, IsNil)
	c.Assert(name, Equals, "gopher")
	name, err = g.GuessName(dirPath) // subdirectory
	c.Assert(err, IsNil)
	c.Assert(name, Equals, "gopher")
}

// This test may fail if you have a git repository in /tmp. By the way, if you
// do have a repository in the temporary file hierarchy, please kill yourself.
func (s *S) TestGitGuesserWhenTheDirectoryIsNotAGitRepository(c *C) {
	p := path.Join(os.TempDir(), "guesser-tests")
	err := os.MkdirAll(p, 0700)
	c.Assert(err, IsNil)
	defer os.RemoveAll(p)
	name, err := GitGuesser{}.GuessName(p)
	c.Assert(name, Equals, "")
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^Git repository not found:.*")
}

func (s *S) TestGitGuesserWithoutTsuruRemote(c *C) {
	p := writeConfig("testdata/gitconfig-without-tsuru-remote", c)
	defer os.RemoveAll(p)
	name, err := GitGuesser{}.GuessName(p)
	c.Assert(name, Equals, "")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "tsuru remote not declared.")
}

func (s *S) TestGitGuesserWithTsuruRemoteNotMatchingTsuruPattern(c *C) {
	p := writeConfig("testdata/gitconfig-not-matching", c)
	defer os.RemoveAll(p)
	name, err := GitGuesser{}.GuessName(p)
	c.Assert(name, Equals, "")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, `"tsuru" remote did not match the pattern. Want something like git@<host>:<app-name>.git, got me@myhost.com:gopher.git`)
}

func (s *S) TestGuessingCommandGuesserNil(c *C) {
	g := GuessingCommand{G: nil}
	c.Assert(g.guesser(), FitsTypeOf, GitGuesser{})
}

func (s *S) TestGuessingCommandGuesserNonNil(c *C) {
	fake := &FakeGuesser{}
	g := GuessingCommand{G: fake}
	c.Assert(g.guesser(), DeepEquals, fake)
}

func (s *S) TestGuessingCommandWithFlagDefined(c *C) {
	*AppName = "myapp"
	fake := &FakeGuesser{name: "other-app"}
	g := GuessingCommand{G: fake}
	name, err := g.Guess()
	c.Assert(err, IsNil)
	c.Assert(name, Equals, "myapp")
	pwd, err := os.Getwd()
	c.Assert(err, IsNil)
	c.Assert(fake.HasGuess(pwd), Equals, false)
}

func (s *S) TestGuessingCommandWithoutFlagDefined(c *C) {
	AppName = nil
	fake := &FakeGuesser{name: "other-app"}
	g := GuessingCommand{G: fake}
	name, err := g.Guess()
	c.Assert(err, IsNil)
	c.Assert(name, Equals, "other-app")
	pwd, err := os.Getwd()
	c.Assert(err, IsNil)
	c.Assert(fake.HasGuess(pwd), Equals, true)
}

func (s *S) TestGuessingCommandFailToGuess(c *C) {
	fake := &FailingFakeGuesser{}
	g := GuessingCommand{G: fake}
	name, err := g.Guess()
	c.Assert(name, Equals, "")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, `tsuru wasn't able to guess the name of the app.

Use the --app flag to specify the name of the app.`)
	pwd, err := os.Getwd()
	c.Assert(err, IsNil)
	c.Assert(fake.HasGuess(pwd), Equals, true)
}

type FakeGuesser struct {
	guesses []string
	name    string
}

func (f *FakeGuesser) HasGuess(path string) bool {
	for _, g := range f.guesses {
		if g == path {
			return true
		}
	}
	return false
}

func (f *FakeGuesser) GuessName(path string) (string, error) {
	f.guesses = append(f.guesses, path)
	return f.name, nil
}

type FailingFakeGuesser struct {
	FakeGuesser
	message string
}

func (f *FailingFakeGuesser) GuessName(path string) (string, error) {
	f.FakeGuesser.GuessName(path)
	return "", errors.New(f.message)
}
