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

	var cmd *exec.Cmd
	if lang == "java" {
		cmd = exec.Command("/usr/lib/jvm/zulu11/bin/java", "Solution")
	} else if lang == "python" {
		cmd = exec.Command("python", "solution.py")
	} else {
		log.Fatal("Language error")
	}

	// prepare command
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

	e := 0

	// run source
	time.AfterFunc(time.Duration(timeLimit)*time.Millisecond, func() {
		_ = syscall.Kill(cmd.Process.Pid, syscall.SIGINT)
	})

	startTime := time.Now().UnixMicro() / 1000
	if err := cmd.Run(); err != nil {
		log.Println("error msg: " + err.Error())
		if strings.HasSuffix(err.Error(), "status 130") {
			e = 1
		} else if strings.HasSuffix(err.Error(), "status 137") || strings.HasSuffix(err.Error(), "killed") {
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

func main() {
	lang := os.Args[1]

	var outbuf, errbuf bytes.Buffer

	// add to cgroup
	initCGroup(os.Getpid())

	// clone target function
	// https://manpages.ubuntu.com/manpages/focal/en/man2/clone.2.html
	// reexec is implementation of clone (maybe)
	cmd := reexec.Command("runner", lang, "1500")
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
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
	defaultPath := "/sys/fs/cgroup/memory/snowbox/"

	if e := os.WriteFile(defaultPath+"memory.limit_in_bytes", []byte("256m"), 0644); e != nil {
		log.Fatal("Write memory.limit_in_bytes: " + e.Error())
	}
	if e := os.WriteFile(defaultPath+"memory.swappiness", []byte("0"), 0644); e != nil {
		log.Fatal("Write memory.swappiness: " + e.Error())
	}
	// write pid
	if e := os.WriteFile(defaultPath+"tasks", []byte(pidStr), 0644); e != nil {
		log.Fatal("Write tasks: " + e.Error())
	}
}
