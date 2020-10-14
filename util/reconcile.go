package util

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

// IsRequeued true if requeue is requested
func IsRequeued(result ctrl.Result, err error) bool {
	return err != nil || result.Requeue || result.RequeueAfter > 0
}
