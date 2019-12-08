package main

import (
	"fmt"
	"github.com/cagataygurturk/kubeport/pkg/chromium"
	"github.com/cagataygurturk/kubeport/pkg/kube"
	"github.com/phayes/freeport"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/mitchellh/go-homedir"
	"github.com/shirou/gopsutil/process"
)

var logger *log.Logger
var k *kube.Kube

func main() {

	home, _ := homedir.Dir()
	/* Initialize logger */
	f, err := os.OpenFile(fmt.Sprintf("%s/Library/Logs/Kubeport", home),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	logger = log.New(f, "prefix", log.LstdFlags)

	c := chromium.New(logger)

	k, err = kube.New(logger)
	if err != nil {
		c.Send(chromium.Message{Type: "responsee", Payload: err.Error()})
	}

	c.StartReading(func(msg *chromium.Message) {

		logger.Printf("Received message in handler: %s", msg)

		switch msg.Type {
		case "listNamespacesCommand":
			c.Send(getNamespaces())
			break

		case "listServicesCommand":
			c.Send(listServices(fmt.Sprintf("%v", msg.Payload)))
			break

		case "connectServiceCommand":
			c.Send(connectService(msg.Payload.(map[string]interface{})))
			break

		case "listActiveConnectionsCommand":
			c.Send(getActiveConnections())
			break
		case "killConnectionCommand":
			c.Send(killConnection(msg.Payload.(float64)))
			break
		}

	})

}

func killConnection(pid float64) chromium.Message {
	_ = syscall.Kill(int(pid), syscall.SIGKILL) // We don't have to be nice to kubectl, so SIGKILL is better
	return getActiveConnections()
}

func reSubMatchMap(r *regexp.Regexp, str string) map[string]string {
	match := r.FindStringSubmatch(str)
	subMatchMap := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 {
			subMatchMap[name] = match[i]
		}
	}

	return subMatchMap
}

func getActiveConnections() chromium.Message {
	logger.Println("Listing active connections")

	processes, err := process.Processes()
	if err != nil {
		logger.Println(err)
		return chromium.Message{
			Type: "error",
		}
	}

	r := regexp.MustCompile(`^kubectl port-forward service\/(?P<service>.*) (?P<localPort>.*):(?P<remotePort>.*) --namespace (?P<namespace>.*)$`)

	connections := make([]map[string]string, 0)
	for _, process := range processes {
		executableName, _ := process.Cmdline()
		if strings.Contains(executableName, "kubectl port-forward") {
			processPayload := reSubMatchMap(r, executableName)
			processPayload["pid"] = fmt.Sprintf("%d", process.Pid)
			connections = append(connections, processPayload)
		}
	}

	return chromium.Message{
		Type:    "listActiveConnectionsResponse",
		Payload: connections,
	}
}

func connectService(serviceConnectionCommandPayload map[string]interface{}) chromium.Message {
	logger.Printf("Connecting to a service: %v", serviceConnectionCommandPayload)

	namespace := serviceConnectionCommandPayload["namespace"].(string)
	serviceName := serviceConnectionCommandPayload["service"].(string)
	port := serviceConnectionCommandPayload["port"].(string)

	logger.Printf("Service connection values are parsed. Namespace: %s/%s:%v", namespace, serviceName, port)

	localPort, err := freeport.GetFreePort()
	if err != nil {
		logger.Println(err)
		return chromium.Message{
			Type: "error",
		}
	}

	kubectlPortForward := exec.Command("kubectl",
		"port-forward",
		fmt.Sprintf("service/%s", serviceName),
		fmt.Sprintf("%d:%s", localPort, port),
		"--namespace",
		namespace,
	)

	kubectlPortForward.Stdout = logger.Writer()
	kubectlPortForward.Stderr = logger.Writer()

	err = kubectlPortForward.Start()

	if err != nil {
		logger.Println(err)
		return chromium.Message{
			Type: "error",
		}
	}

	logger.Printf("Service connection is successful with PID %d. Returning the response.", kubectlPortForward.Process.Pid)

	return chromium.Message{
		Type: "connectServiceCommandResponse",
		Payload: map[string]string{
			"service":   fmt.Sprintf("%s:%s", serviceName, port),
			"localPort": fmt.Sprintf("%d", localPort),
		},
	}
}

func getNamespaces() chromium.Message {

	logger.Println("Listing namespaces")

	namespaces, err := k.ListNamespaces()
	if err != nil {
		//TODO send error message back
		logger.Println(err)
	}

	return chromium.Message{
		Type:    "listNamespacesResponse",
		Payload: namespaces.Items,
	}
}

func listServices(namespace string) chromium.Message {

	logger.Printf("Listing services in %s", namespace)

	services, err := k.ListServices(namespace)
	if err != nil {
		//TODO send error message back
		logger.Println(err)
	}

	return chromium.Message{
		Type:    "listServicesResponse",
		Payload: services.Items,
	}

}
