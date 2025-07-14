package main

import (
	"fmt"
	"golang.org/x/sys/windows/svc"
	"log"
	"os/exec"
)

type proxyService struct{}

func (m *proxyService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	s <- svc.Status{State: svc.StartPending}
	go startProxy()
	s <- svc.Status{State: svc.Running, Accepts: accepted}

	for req := range r {
		switch req.Cmd {
		case svc.Interrogate:
			s <- req.CurrentStatus
		case svc.Stop, svc.Shutdown:
			s <- svc.Status{State: svc.StopPending}
			stopProxy()
			return false, 0
		default:
			log.Printf("unexpected control request: %v", req)
		}
	}
	return false, 0
}

func runCmd(cmd string) {
	fmt.Println("Executing:", cmd)
	out, err := execCommand("cmd", "/C", cmd)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println(string(out))
}

func execCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}
