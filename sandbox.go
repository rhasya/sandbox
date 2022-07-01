package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/rhasya/sandbox/ns"
)

type Result struct {
	E    int   `json:"e"`
	Time int64 `json:"time"`
	Mem  int64 `json:"mem"`
}

func init() {
	reexec.Register("runner", runner)

	// clone
	if reexec.Init() {
		os.Exit(0)
	}
}

func runner() {
	log.Println("Init namespace")
	ns.InitNamespace("/tmp/snowbox")

	lang := os.Args[1]
	timeLimit, _ := strconv.Atoi(os.Args[2])

	var errbuf bytes.Buffer
	infile, _ := os.OpenFile("/tmp/input.txt", os.O_RDONLY, 0644)
	outfile, _ := os.OpenFile("/tmp/output.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	defer func() {
		_ = infile.Close()
		_ = outfile.Close()
	}()

	var cmdString string
	if lang == "java" {
		cmdString = "java Solution"
	} else if lang == "python" {
		cmdString = "python solution.py"
	} else {
		log.Fatal("Language error")
	}

	// prepare command
	cmd := exec.Command("sh", "-c", cmdString)
	cmd.Stdin = infile
	cmd.Stdout = outfile
	cmd.Stderr = &errbuf
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Env = []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin:/usr/lib/jvm/zulu11/bin",
		"JAVA_HOME=/usr/lib/jvm/zulu11",
	}

	// TLE terminator
	time.AfterFunc(time.Duration(timeLimit)*time.Millisecond, func() {
		_ = syscall.Kill(cmd.Process.Pid, syscall.SIGABRT)
	})

	e := 0

	// run source
	startTime := time.Now().UnixMicro() / 1000
	if err := cmd.Run(); err != nil {
		log.Println(errbuf.String())
		log.Println(err)
		if strings.HasSuffix(err.Error(), "signal: aborted") {
			e = 1
		} else if strings.HasSuffix(err.Error(), "signal: killed") {
			e = 2
		} else {
			e = 3
		}
	}
	endTime := time.Now().UnixMicro() / 1000

	r := Result{e, endTime - startTime,
		cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss}

	j, _ := json.Marshal(r)
	_, _ = os.Stdout.Write(j)
}

// before you run do this process
// 1. make directory : /tmp/snowbox
// 2. get python image from docker : docker export $(docker create python:3-slim) | tar -C /tmp/snowbox -xzv -
// 3. download and install java11 : /usr/lib/jvm/zulu11
func main() {
	lang := os.Args[1]

	var outbuf bytes.Buffer

	// add to cgroup
	initCGroup(os.Getpid())
	log.Printf("[%d] current namespace. parent=%d\n", os.Getpid(), os.Getppid())

	// clone target function
	// https://manpages.ubuntu.com/manpages/focal/en/man2/clone.2.html
	// reexec is implementation of clone (maybe)
	cmd := reexec.Command("runner", lang, "1500")
	cmd.Stdout = &outbuf
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
		log.Fatal(err)
	}

	var r Result
	_ = json.Unmarshal(outbuf.Bytes(), &r)
	if r.E == 1 {
		fmt.Println("Time Limit Exceeded.")
	} else if r.E == 2 {
		fmt.Println("Memory Limit Exceeded.")
	} else if r.E == 3 {
		fmt.Println("Runtime error.")
	} else {
		fmt.Printf("%d ms / %d Kbytes\n", r.Time, r.Mem)
	}
}

func initCGroup(pid int) {
	pidStr := strconv.Itoa(pid)
	defaultPath := "/sys/fs/cgroup/memory/sandg/"

	// write memory size
	prevSizeStr, e := os.ReadFile(defaultPath + "memory.limit_in_bytes")
	if e != nil {
		log.Fatal("Read prev size: " + e.Error())
	}
	prevSize, _ := strconv.Atoi(string(prevSizeStr))
	var updateOrder []string
	if prevSize > 256*1024*1024 {
		// bigger
		updateOrder = []string{"memory.kmem.limit_in_bytes", "memory.memsw.limit_in_bytes", "memory.limit_in_bytes"}
	} else {
		// smaller
		updateOrder = []string{"memory.kmem.limit_in_bytes", "memory.limit_in_bytes", "memory.memsw.limit_in_bytes"}
	}

	for _, f := range updateOrder {
		if e := os.WriteFile(defaultPath+f, []byte("256m"), 0644); e != nil {
			log.Fatal("Write " + f + ": " + e.Error())
		}
	}

	// write pid
	if e := os.WriteFile(defaultPath+"tasks", []byte(pidStr), 0644); e != nil {
		log.Fatal("Write tasks: " + e.Error())
	}
}
