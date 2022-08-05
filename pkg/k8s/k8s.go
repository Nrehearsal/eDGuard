package k8s

import (
	"k8s.io/client-go/kubernetes"
	"log"
	ctrl "sigs.k8s.io/controller-runtime"
)

var k8sClient kubernetes.Interface
var err error

func init() {
	k8sClient, err = kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		log.Fatal(err, "setup k8s client failed")
	}
}

func GetK8SClient() kubernetes.Interface {
	return k8sClient
}
