package repository

import (
	"github.com/timeredbull/tsuru/config"
	"github.com/timeredbull/tsuru/log"
	. "launchpad.net/gocheck"
	stdlog "log"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

type S struct {
	git         *repository
	gitRoot     string
	gitosisBare string
	gitosisRepo string
	mngr        *gitosisManager
	logFile     *os.File
	repoPath    string
}

var _ = Suite(&S{})

func (s *S) setUpGit(c *C) {
	s.repoPath = "/tmp/git" + time.Now().Format("20060102150405")
	err := os.MkdirAll(s.repoPath, 0755)
	c.Assert(err, IsNil)
	s.git = &repository{path: s.repoPath}
	_, err = s.git.run("init")
	c.Assert(err, IsNil)
}

func (s *S) tearDownGit(c *C) {
	err := os.RemoveAll(s.repoPath)
	c.Assert(err, IsNil)
}

func (s *S) SetUpSuite(c *C) {
	err := config.ReadConfigFile("../etc/tsuru.conf")
	c.Assert(err, IsNil)
	s.gitRoot, err = config.GetString("git:root")
	c.Assert(err, IsNil)
	s.gitosisBare, err = config.GetString("git:gitosis-bare")
	c.Assert(err, IsNil)
	s.gitosisRepo, err = config.GetString("git:gitosis-repo")
	err = os.RemoveAll(s.gitRoot)
	c.Assert(err, IsNil)
	err = os.MkdirAll(s.gitRoot, 0777)
	c.Assert(err, IsNil)
	err = exec.Command("git", "init", "--bare", s.gitosisBare).Run()
	c.Assert(err, IsNil)
	err = exec.Command("git", "clone", s.gitosisBare, s.gitosisRepo).Run()
	c.Assert(err, IsNil)
	s.logFile, err = os.Create("/tmp/tsuru-tests.log")
	c.Assert(err, IsNil)
	RunAgent()
	s.setUpGit(c)
	s.mngr, err = newGitosisManager()
	c.Assert(err, IsNil)
	log.Target = stdlog.New(s.logFile, "[tsuru-tests]", stdlog.LstdFlags|stdlog.Llongfile)
}

func (s *S) SetUpTest(c *C) {
	fpath := path.Join(s.gitosisRepo, "gitosis.conf")
	f, err := os.Create(fpath)
	c.Assert(err, IsNil)
	f.Close()
	err = s.mngr.git.commit("added gitosis test file")
	c.Assert(err, IsNil)
	err = s.mngr.git.push("origin", "master")
	c.Assert(err, IsNil)
}

func (s *S) TearDownSuite(c *C) {
	defer s.logFile.Close()
	defer s.tearDownGit(c)
	err := os.RemoveAll(s.gitRoot)
	c.Assert(err, IsNil)
}

func (s *S) TearDownTest(c *C) {
	_, err := s.mngr.git.run("rm", "gitosis.conf")
	c.Assert(err, IsNil)
	err = s.mngr.git.commit("removing test file")
	c.Assert(err, IsNil)
	err = s.mngr.git.push("origin", "master")
	c.Assert(err, IsNil)
}

func (s *S) lastBareCommit(c *C) string {
	bareOutput, err := exec.Command("git", "--git-dir="+s.gitosisBare, "log", "-1", "--pretty=format:%s").CombinedOutput()
	c.Assert(err, IsNil)
	return string(bareOutput)
}