package instrumentation

import (
	"fmt"
	"os"
	"time"
)

func SetupTraceServerClient(endpoint string) func() {
	setupDoneCh, cancel := InitSender(endpoint, 240*time.Second)
	go func() {
		// receive in another goroutine to avoid blocking
		<-setupDoneCh
	}()
	// <-setupDoneCh // EPPPPI-NOTE: don't wait for setup because trace server is deployed after controller-manager
	return cancel
}

func WriteResolvconf() {
	resolvconf := `search default.svc.cluster.local svc.cluster.local cluster.local
nameserver 10.96.0.10
options ndots:5
`
	fp, err := os.Create("/etc/resolv.conf")
	if err != nil {
		fmt.Println("failed to open resolv.conf: ", err)
		return
	}
	defer fp.Close()
	_, err = fp.Write([]byte(resolvconf))
	if err != nil {
		fmt.Println("failed to write resolv.conf: ", err)
		return
	}
	fmt.Println("sucessfully wrote resolv.conf")
}
