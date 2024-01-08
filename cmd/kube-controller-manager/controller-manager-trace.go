package main

import (
	"fmt"
	"os"

	k8scpdtinst "github.com/eppppi/k8s-cp-dt/instrumentation"
)

func setupTraceServer(endpoint string) {
	_, cancel := k8scpdtinst.InitSender(endpoint)
	// <-setupDoneCh // EPPPPI-NOTE: don't wait for setup because trace server is deployed after controller-manager
	defer cancel()
}

func writeResolvconf() {
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
	fmt.Println("success to write resolv.conf")
}
