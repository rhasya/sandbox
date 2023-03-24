package sandbox_utils

import (
	"log"
	"os"
	"path/filepath"
	"syscall"
)

func InitNamespace(newRoot string) error {
	log.Printf("Init InitNamespace(%s) starting...\n", newRoot)

	// pivotRoot
	if err := pivotRoot(newRoot); err != nil {
		log.Printf("pivotRoot(%s) failed, err: %s\n", newRoot, err.Error())
		return err
	}

	// set Hostname
	if err := syscall.Sethostname([]byte("snowbox")); err != nil {
		log.Printf("syscall.Sethostname(snowbox) failed, err: %s\n", err.Error())
		return err
	}

	log.Printf("Init InitNamespace(%s) done\n", newRoot)

	return nil
}

func pivotRoot(newRoot string) error {
	// reference : https://manpages.ubuntu.com/manpages/impish/man2/pivot_root.2.html
	oldRoot := filepath.Join(newRoot, "/.old-root")

	if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		log.Printf("syscall.Mount(%s, %s, NULL, syscall.MS_BIND|syscall.MS_REC, NULL) failed", newRoot, newRoot)
		return err
	}

	if err := os.MkdirAll(oldRoot, 0777); err != nil {
		log.Printf("os.MkdirAll(%s, 0777) failed", oldRoot)
		return err
	}

	if err := syscall.PivotRoot(newRoot, oldRoot); err != nil {
		log.Printf("syscall.PivotRoot(%s, %s) failed", newRoot, oldRoot)
		return err
	}

	if err := os.Chdir("/"); err != nil {
		log.Printf("os.Chdir(/) failed")
		return err
	}

	oldRoot = "/.old-root"
	if err := syscall.Unmount(oldRoot, syscall.MNT_DETACH); err != nil {
		log.Printf("syscall.Unmount(%s, syscall.MNT_DETACH) failed", oldRoot)
		return err
	}

	if err := syscall.Rmdir(oldRoot); err != nil {
		log.Printf("syscall.Rmdir(%s) failed", oldRoot)
		return err
	}

	// mount proc to run other process
	if e := syscall.Mount("/proc", "/proc", "proc", 0, ""); e != nil {
		log.Fatal("Mount proc: " + e.Error())
	}

	return nil
}
