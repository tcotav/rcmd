package rcmd

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"
)

func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return ssh.PublicKeys(key)
}

func GetSshClient(user string, keyfile string, targetUrl string) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{PublicKeyFile(keyfile)},
	}
	// new connection
	return ssh.Dial("tcp", targetUrl, sshConfig)
}

type RunResult struct {
	Summary  map[string]int
	HostList []HostCmdReturn
}

type HostCmdReturn struct {
	Host      string
	Stdout    string
	Stderr    string
	Timestamp string
}

//
type HostCmdRequest struct {
	Host       string
	SshUser    string
	SshKeyfile string
	Command    string
}

// let's put in a method to make pretty output
func (hcr HostCmdReturn) Dump() []string {
	var retList []string
	retList = append(retList, hcr.Host)
	for _, s := range strings.Split(hcr.Stdout, "\n") {
		retList = append(retList, s)
	}
	return retList
}

func SshSession(user string, keyFile string, targetUrl string, cmd string) *HostCmdReturn {
	t := fmt.Sprintf("%s", time.Now())
	connection, err := GetSshClient(user, keyFile, targetUrl)
	if err != nil {
		r := &HostCmdReturn{Host: targetUrl, Stderr: fmt.Sprintf("local: %s", err.Error()), Timestamp: t}
		return r
	}

	// client can be used across multiple sessions
	session, err := connection.NewSession()
	if err != nil {
		r := &HostCmdReturn{Host: targetUrl, Stderr: fmt.Sprintf("local: %s", err.Error()), Timestamp: t}
		return r
	}
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Run(cmd)
	if err != nil {
		r := &HostCmdReturn{Host: targetUrl, Stderr: fmt.Sprintf("local: %s", err.Error()), Timestamp: t}
		return r
	}

	r := &HostCmdReturn{Host: targetUrl, Stdout: stdoutBuf.String(), Stderr: stderrBuf.String(), Timestamp: t}
	return r
}

func GetHostMatches(re string, filename string) ([]string, error) {
	regex, err := regexp.Compile(re)
	if err != nil {
		return nil, err
	}
	fh, err := os.Open(filename)
	f := bufio.NewReader(fh)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	retList := make([]string, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		if regex.MatchString(s) {
			retList = append(retList, s)
		}
	}

	return retList, nil
}

func worker(id int, jobs <-chan HostCmdRequest, results chan<- HostCmdReturn) {
	for j := range jobs {
		r := SshSession(j.SshUser, j.SshKeyfile, j.Host, j.Command)
		results <- *r
	}
}

func ProcessListBase(hostList []string, numWorkers int, ssh_user string, ssh_keyfile string, cmd string) RunResult {
	return ProcessList(hostList, numWorkers, ssh_user, ssh_keyfile, cmd, false)
}

func ProcessList(hostList []string, numWorkers int, ssh_user string, ssh_keyfile string, cmd string, errOnly bool) RunResult {
	numHosts := len(hostList)
	jobs := make(chan HostCmdRequest, numHosts)
	results := make(chan HostCmdReturn, numHosts)

	if numWorkers > numHosts {
		numWorkers = numHosts
	}

	for wid := 0; wid < numWorkers; wid++ {
		go worker(wid, jobs, results)
	}

	r, _ := regexp.Compile("(:[0-9]+)$")
	for i := 0; i < numHosts; i++ {
		useHost := hostList[i]
		if !r.MatchString(useHost) {
			useHost = fmt.Sprintf("%s:22", useHost)
		}
		j := HostCmdRequest{SshUser: ssh_user, SshKeyfile: ssh_keyfile, Host: useHost, Command: cmd}
		jobs <- j
	}

	var errCount, successCount int
	retList := make([]HostCmdReturn, 0)
	for i := 0; i < numHosts; i++ {
		res := <-results
		if res.Stderr == "" {
			successCount++
		} else {
			errCount++
		}

		if errOnly {
			// test if Stderr contains anything
			// skip to next if empty
			if res.Stderr == "" {
				continue
			}
		}
		retList = append(retList, res)
	}

	summary := make(map[string]int)
	summary["total"] = successCount + errCount
	summary["success"] = successCount
	summary["failures"] = errCount

	return RunResult{Summary: summary, HostList: retList}
}
