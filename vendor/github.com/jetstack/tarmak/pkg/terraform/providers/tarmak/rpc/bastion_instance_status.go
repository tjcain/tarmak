// Copyright Jetstack Ltd. See LICENSE for details.
package rpc

import (
	"fmt"
	"time"

	cluster "github.com/jetstack/tarmak/pkg/apis/cluster/v1alpha1"
)

var (
	BastionInstanceStatusCall = fmt.Sprintf("%s.BastionInstanceStatus", RPCName)
)

type BastionInstanceStatusArgs struct {
	Username string
	Hostname string
}

type BastionInstanceStatusReply struct {
	Status string
}

func (r *tarmakRPC) BastionInstanceStatus(args *BastionInstanceStatusArgs, result *BastionInstanceStatusReply) error {
	r.tarmak.Log().Debug("received rpc bastion status")

	if r.cluster.GetState() == cluster.StateDestroy {
		result.Status = "unknown"
		return nil
	}

	var err error
	for i := 1; i <= Retries; i++ {
		if err = r.cluster.Environment().VerifyBastionAvailable(); err != nil {
			r.tarmak.Log().Warn(err)
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("bastion instance is not ready: %s", err)
	}

	result.Status = "ready"
	return nil
}
