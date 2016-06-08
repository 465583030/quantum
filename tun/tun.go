package tun

import (
	"github.com/Supernomad/quantum/common"
	"github.com/Supernomad/quantum/logger"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

/*
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <linux/if.h>
#include <linux/if_tun.h>

#define IFREQ_SIZE sizeof(struct ifreq)
*/
import "C"

type Tun struct {
	Name string
	log  *logger.Logger
	file *os.File
}

func (tun *Tun) Close() error {
	return tun.file.Close()
}

func (tun *Tun) Read() (*common.Payload, bool) {
	buf := make([]byte, common.MaxPacketLength)
	n, err := tun.file.Read(buf[common.PacketStart:])

	if err != nil {
		tun.log.Warn("[TUN] Read Error:", err)
		return nil, false
	}

	return common.NewTunPayload(buf, n), true
}

func (tun *Tun) Write(payload *common.Payload) bool {
	_, err := tun.file.Write(payload.Packet)
	if err != nil {
		tun.log.Warn("[TUN] Write Error:", err)
		return false
	}
	return true
}

func New(ifPattern string, cidr string, log *logger.Logger) (*Tun, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	ifName, err := createTun(file, ifPattern)
	if err != nil {
		file.Close()
		return nil, err
	}
	realName := ifName[:strings.Index(ifName, "\000")]
	cmd := exec.Command("ip", "link", "set", "dev", realName, "up")
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	cmd = exec.Command("ip", "addr", "add", cidr, "dev", realName)
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	return &Tun{realName, log, file}, nil
}

type ifReq struct {
	Name  [C.IFNAMSIZ]byte
	Flags uint16
	pad   [C.IFREQ_SIZE - C.IFNAMSIZ - 2]byte
}

func createTun(file *os.File, ifPattern string) (string, error) {
	var req ifReq
	req.Flags = C.IFF_NO_PI | C.IFF_TUN

	copy(req.Name[:15], ifPattern)

	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&req)))
	if err != 0 {
		return "", err
	}
	return string(req.Name[:]), nil
}
