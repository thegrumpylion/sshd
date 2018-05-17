package main

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// http://man7.org/linux/man-pages/man2/ioctl_tty.2.html

func ExecPTY(c *exec.Cmd) (pty *os.File, err error) {
	pty, tty, err := OpenPTY()
	if err != nil {
		return nil, err
	}
	defer tty.Close()
	c.Stdout = tty
	c.Stdin = tty
	c.Stderr = tty
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.Setctty = true
	c.SysProcAttr.Setsid = true
	err = c.Start()
	if err != nil {
		pty.Close()
		return nil, err
	}
	return pty, err
}

func OpenPTY() (pty, peer *os.File, err error) {
	pty, err = os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	// In case of error after this point, make sure we close the ptmx fd.
	defer func() {
		if err != nil {
			pty.Close()
		}
	}()

	if err := unlockpt(pty); err != nil {
		return nil, nil, err
	}

	peer, err = getPeer(pty)
	// named returns
	return
}

func getPeer(f *os.File) (*os.File, error) {
	pfd, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), unix.TIOCGPTPEER, uintptr(os.O_RDWR|syscall.O_NOCTTY))
	if err != 0 {
		return nil, err
	}
	pf := os.NewFile(pfd, "")
	if pf == nil {
		return nil, errors.New("Could not create a new file from fd")
	}
	return pf, nil
}

func unlockpt(f *os.File) error {
	var i int32
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&i))); err != 0 {
		return err
	}
	return nil
}
