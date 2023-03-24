package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/rhasya/sandbox/sandbox_utils"
)

type Result struct {
	Status int    `json:"status"`
	Error  string `json:"error,omitempty"`
	Time   int64  `json:"time,omitempty"`
	Memory int64  `json:"memory,omitempty"`
	Output string `json:"output,omitempty"`
}

const (
	StatusAc = iota
	StatusRe
	StatusTle
	StatusMem
)

func init() {
	reexec.Register("runner", runner)

	// clone
	if reexec.Init() {
		os.Exit(0)
	}
}

func runner() {
	basedir := os.Args[1]
	target := os.Args[2]
	timeout, _ := strconv.Atoi(os.Args[3])
	memory := os.Args[4]

	if err := sandbox_utils.InitNamespace(basedir); err != nil {
		result, _ := json.Marshal(Result{Status: StatusRe, Error: "Runtime Error"})
		_, _ = os.Stdout.Write(result)
		return
	}

	var o, e bytes.Buffer
	var cmd *exec.Cmd

	// prepare command
	if target == "java" {
		cmd = exec.Command("java", fmt.Sprintf("-Xmx%sk", memory), "Solution")
	} else if target == "cpp" {
		cmd = exec.Command("/solution")
	} else {
		result, _ := json.Marshal(Result{Status: StatusRe, Error: "Wrong target"})
		_, _ = os.Stdout.Write(result)
		return
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = &o
	cmd.Stderr = &e
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Env = []string{
		"PS1=[snow] # ",
	}

	// TLE terminator
	tle := false
	time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		_ = syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)
		tle = true
	})

	// Start
	startTime := time.Now().UnixNano() / 1000000
	if err := cmd.Run(); err != nil {
		log.Printf("err: %s\n", err.Error())

		if tle {
			// Time Limit Exceeded
			result, _ := json.Marshal(Result{Status: StatusTle, Error: "Time Limit Exceeded"})
			_, _ = os.Stdout.Write(result)
		} else {
			// Something Wrong (maybe memory error)
			result, _ := json.Marshal(Result{Status: StatusRe, Error: "Runtime Error"})
			_, _ = os.Stdout.Write(result)
		}

		return
	}
	endTime := time.Now().UnixNano() / 1000000

	// Runtime Error
	if e.Len() > 0 {
		log.Printf("stderr: %s\n", e.String())

		result, _ := json.Marshal(Result{Status: StatusRe, Error: "Runtime Error"})
		_, _ = os.Stdout.Write(result)
		return
	}

	// Print Result
	timeCost := endTime - startTime
	if timeCost == 0 {
		timeCost = 1
	}
	memoryCost := cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss / 1024
	result, _ := json.Marshal(Result{Status: StatusAc, Time: timeCost, Memory: memoryCost, Output: strings.TrimSpace(o.String())})
	_, _ = os.Stdout.Write(result)
}

func main() {
	basedir := flag.String("basedir", "/tmp/snow", "base directory of sandbox")
	target := flag.String("target", "cpp", "target language (cpp / java)")
	timeout := flag.String("timeout", "2000", "timeout duration in ms")
	memory := flag.String("memory", "262144", "memory usage limit in kb")
	flag.Parse()

	if err := sandbox_utils.InitCGroup(os.Getpid(), *memory); err != nil {
		result, _ := json.Marshal(Result{Status: StatusRe, Error: "Runtime Error"})
		os.Stdout.Write(result)
		os.Exit(0)
	}

	// clone target function
	// https://manpages.ubuntu.com/manpages/focal/en/man2/clone.2.html
	// reexec is implementation of clone (maybe)
	cmd := reexec.Command("runner", *basedir, *target, *timeout, *memory)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWUTS,
		UidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      os.Getuid(),
			Size:        1,
		}},
		GidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      os.Getgid(),
			Size:        1,
		}},
	}

	if err := cmd.Run(); err != nil {
		result, _ := json.Marshal(Result{Status: StatusRe, Error: "Runtime Error"})
		os.Stdout.Write(result)
		log.Println(err.Error())
	}
}
