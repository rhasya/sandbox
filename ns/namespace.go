package ns

import (
	"log"
	"os"
	"syscall"
)

func InitNamespace(newRoot string) {
	pivotRoot(newRoot)

	// set Hostname
	if e := syscall.Sethostname([]byte("snowbox")); e != nil {
		log.Fatal("SetHostname: " + e.Error())
	}
}

func pivotRoot(newRoot string) {
	// reference : https://manpages.ubuntu.com/manpages/impish/man2/pivot_root.2.html
	oldPath := "/oldrootfs"

	if e := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); e != nil {
		log.Fatal("Mount2: " + e.Error())
	}
	if e := syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); e != nil {
		log.Fatal("Mount2: " + e.Error())
	}
	if e := os.MkdirAll(newRoot+oldPath, 0777); e != nil {
		log.Fatal("Mkdir: " + e.Error())
	}
	if e := syscall.PivotRoot(newRoot, newRoot+oldPath); e != nil {
		log.Fatal("PivotRoot: " + e.Error())
	}
	if e := os.Chdir("/"); e != nil {
		log.Fatal("Chdir: " + e.Error())
	}
	if e := syscall.Unmount("/oldrootfs", syscall.MNT_DETACH); e != nil {
		log.Fatal("Unmount: " + e.Error())
	}
	if e := syscall.Rmdir("/oldrootfs"); e != nil {
		log.Fatal("Rmdir" + e.Error())
	}

	// mount proc to run other process
	if e := syscall.Mount("/proc", "/proc", "proc", 0, ""); e != nil {
		log.Fatal("Mount proc: " + e.Error())
	}
}
