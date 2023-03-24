package sandbox_utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

const (
	cgCpuPathPrefix = "/sys/fs/cgroup/cpu/snowbox"
	cgPidPathPrefix = "/sys/fs/cgroup/pid/snowbox"
	cgMemPathPrefix = "/sys/fs/cgroup/memory/snowbox"
)

func InitCGroup(pid int, memory string) error {
	pidStr := strconv.Itoa(pid)
	log.Printf("Init CGroup(%s, %s) starting...\n", pidStr, memory)

	dirs := []string{
		"/sys/fs/cgroup/cpu/snowbox",
		"/sys/fs/cgroup/pid/snowbox",
		"/sys/fs/cgroup/memory/snowbox",
	}

	// makedir
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Printf("os.MkdirAll(%s, os.ModePerm) failed, err: %s\n", dir, err.Error())
			return err
		}
	}

	if err := cpuCGroup(pidStr); err != nil {
		log.Printf("cpuCGroup(%s) failed, err: %s\n", pidStr, err.Error())
		return err
	}

	if err := pidCGroup(pidStr); err != nil {
		log.Printf("pidCGroup(%s) failed, err: %s\n", pidStr, err.Error())
		return err
	}

	if err := memCGroup(pidStr, memory); err != nil {
		log.Printf("memCGroup(%s, %s) failed, err: %s\n", pidStr, memory, err.Error())
		return err
	}

	log.Printf("Init CGroup(%s, %s) done\n", pidStr, memory)

	return nil
}

func cpuCGroup(pid string) error {
	mapping := map[string]string{
		"tasks":            pid,
		"cpu.cfs_quota_us": "10000",
	}

	for key, value := range mapping {
		path := filepath.Join(cgCpuPathPrefix, key)
		if err := ioutil.WriteFile(path, []byte(value), 0644); err != nil {
			log.Printf("Writing [%s] to file: %s failed\n", value, path)
			return err
		}
		c, _ := ioutil.ReadFile(path)
		log.Printf("Content of %s is: %s", path, c)
	}
	return nil
}

func pidCGroup(pid string) error {
	mapping := map[string]string{
		"cgroup.procs": pid,
		"pids.max":     "64",
	}

	for key, value := range mapping {
		path := filepath.Join(cgPidPathPrefix, key)
		if err := ioutil.WriteFile(path, []byte(value), 0644); err != nil {
			log.Printf("Writing [%s] to file: %s failed\n", value, path)
			return err
		}
		c, _ := ioutil.ReadFile(path)
		log.Printf("Content of %s is: %s", path, c)
	}
	return nil
}

func memCGroup(pid, memory string) error {
	mapping := map[string]string{
		"memory.kmem.limit_in_bytes":  "64m",
		"tasks":                       pid,
		"memory.limit_in_bytes":       fmt.Sprintf("%sk", memory),
		"memory.memsw.limit_in_bytes": fmt.Sprintf("%sk", memory),
	}

	for key, value := range mapping {
		path := filepath.Join(cgMemPathPrefix, key)
		if err := ioutil.WriteFile(path, []byte(value), 0644); err != nil {
			log.Printf("Writing [%s] to file: %s failed\n", value, path)
			return err
		}
		c, _ := ioutil.ReadFile(path)
		log.Printf("Content of %s is: %s", path, c)
	}
	return nil
}
