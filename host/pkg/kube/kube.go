package kube

import (
	"flag"
	"github.com/mitchellh/go-homedir"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"path/filepath"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type Kube struct {
	logger    *log.Logger
	clientset *kubernetes.Clientset
}

func New(logger *log.Logger) (*Kube, error) {

	var kubeconfig *string

	home, _ := homedir.Dir()

	if home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	return &Kube{logger: logger, clientset: clientset}, nil
}

func (k *Kube) ListNamespaces() (*v1.NamespaceList, error) {
	return k.clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
}

func (k *Kube) ListServices(namespace string) (*v1.ServiceList, error) {
	return k.clientset.CoreV1().Services(namespace).List(metav1.ListOptions{})
}
