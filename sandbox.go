package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
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
	reexec.Register("temp", tempInit)

	// it call folk
	// first time => return false
	// second time => return true
	if reexec.Init() {
		os.Exit(0)
	}
}

func tempInit() {
	// This is another file
	arg1 := os.Args[1]

	var errbuf bytes.Buffer
	outfile, _ := os.OpenFile("output.txt", os.O_WRONLY|os.O_CREATE, 0755)

	cmd := exec.Command("python3.10", arg1)
	cmd.Stdout = outfile
	cmd.Stderr = &errbuf

	// TLE terminator
	time.AfterFunc(time.Duration(2000)*time.Millisecond, func() {
		_ = syscall.Kill(cmd.Process.Pid, syscall.SIGABRT)
	})

	e := 0

	// run source
	startTime := time.Now().UnixMicro() / 1000
	if err := cmd.Run(); err != nil {
		if strings.HasSuffix(err.Error(), "signal: aborted") {
			e = 1
		} else {
			e = 2
		}
	}
	endTime := time.Now().UnixMicro() / 1000

	r := Result{e, endTime - startTime,
		cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss}

	j, _ := json.Marshal(r)
	_, _ = os.Stdout.Write(j)

	outfile.Close()
}

func main() {

	var outbuf bytes.Buffer

	cmd := reexec.Command("temp", "a.py")
	cmd.Stdout = &outbuf
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	var r Result
	_ = json.Unmarshal(outbuf.Bytes(), &r)
	if r.E == 1 {
		fmt.Println("Time Limit Exceeded.")
	} else if r.E == 2 {
		fmt.Println("Runtime error.")
	} else {
		fmt.Printf("%d ms / %d Kbytes\n", r.Time, r.Mem)
	}
}
