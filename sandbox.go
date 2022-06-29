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
)

type Result struct {
	E    int   `json:"e"`
	Time int64 `json:"time"`
	Mem  int64 `json:"mem"`
}

func init() {
	reexec.Register("runner", tempInit)

	// it call folk
	// first time => return false
	// second time => return true
	if reexec.Init() {
		os.Exit(0)
	}
}

func tempInit() {
	// This is another file
	solution := os.Args[1]
	timeLimit, _ := strconv.Atoi(os.Args[2])

	var errbuf bytes.Buffer
	infile, _ := os.OpenFile("input.txt", os.O_RDONLY, 0744)
	outfile, _ := os.OpenFile("output.txt", os.O_WRONLY|os.O_CREATE, 0755)

	defer infile.Close()
	defer outfile.Close()

	cmd := exec.Command(solution)
	cmd.Stdin = infile
	cmd.Stdout = outfile
	cmd.Stderr = &errbuf
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// TLE terminator
	time.AfterFunc(time.Duration(timeLimit)*time.Millisecond, func() {
		_ = syscall.Kill(cmd.Process.Pid, syscall.SIGABRT)
	})

	e := 0

	// run source
	startTime := time.Now().UnixMicro() / 1000
	if err := cmd.Run(); err != nil {
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

func main() {

	var outbuf bytes.Buffer

	// add to cgroup
	initCGroup(os.Getpid())

	cmd := reexec.Command("runner", "./temp/a", "5000")
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
	defaultPath := "/sys/fs/cgroup/memory/sboxg/"

	// write to mem limit to file
	e1 := os.WriteFile(defaultPath+"memory.kmem.limit_in_bytes", []byte("256m"), 0644)
	if e1 != nil {
		log.Fatal("e1 " + e1.Error())
	}
	e2 := os.WriteFile(defaultPath+"memory.memsw.limit_in_bytes", []byte("256m"), 0644)
	if e2 != nil {
		log.Fatal("e2 " + e2.Error())
	}
	e3 := os.WriteFile(defaultPath+"memory.limit_in_bytes", []byte("256m"), 0644)
	if e3 != nil {
		log.Fatal("e3 " + e2.Error())
	}
	e4 := os.WriteFile(defaultPath+"tasks", []byte(pidStr), 0644)
	if e4 != nil {
		log.Fatal("e4 " + e4.Error())
	}
}
