package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // Import metav1
	"k8s.io/client-go/rest"                       // Import the rest package for in-cluster config
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
	// Initialize the in-cluster Kubernetes client configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Error building in-cluster Kubernetes config: %v", err)
		panic(nil)
	}

	// Initialize the Metrics Server client
	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Metrics Server client: %v", err)
		panic(nil)
	}

	// Get the node name from the environment variable
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatal("Environment variable NODE_NAME is not set")
		panic(nil)
	}

	// Read contents of a file specified by an environment variable
	filePath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if filePath == "" {
		log.Fatal("Environment variable SSH_PRIVATE_KEY_PATH is not set")
		panic(nil)
	}

	// Read contents of a file specified by an environment variable
	sshUser := os.Getenv("SSH_USER_NAME")
	if sshUser == "" {
		log.Fatal("Environment variable SSH_USER_NAME is not set")
		panic(nil)
	}

	sshHost := os.Getenv("SSH_HOST")
	if sshHost == "" {
		log.Fatal("Environment variable SSH_HOST is not set")
		panic(nil)
	}

	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
		panic(nil)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(fileContents)
	if err != nil {
		log.Fatalf("Unable to parse private key: %v", err)
		panic(nil)
	}

	shutdownThreshold := os.Getenv("SHUTDOWN_THRESHOLD")
	if shutdownThreshold == "" {
		log.Fatal("Environment variable SHUTDOWN_THRESHOLD is not set")
		panic(nil)
	}

	shutdownThresholdInt, err := strconv.ParseInt(shutdownThreshold, 10, 64)
	if err != nil {
		log.Fatalf("%v", err)
		panic(nil)
	}

	shutdownDelay := os.Getenv("SHUTDOWN_DELAY")
	if shutdownDelay == "" {
		log.Fatal("Environment variable SHUTDOWN_DELAY is not set")
		panic(nil)
	}

	shutdownDelayInt, err := strconv.Atoi(shutdownDelay)
	if err != nil {
		log.Fatalf("%v", err)
		panic(nil)
	}

	// Create a timer that triggers every minute
	timer := time.NewTicker(1 * time.Minute)

	// Initialize the consecutive checks counter
	checksBelowThreshold := 0
	// Define a function to check CPU usage and SSH into the node
	checkAndSSH := func() {
		// Query node metrics (CPU usage) for the specified node
		nodeMetricsClient := metricsClient.MetricsV1beta1().NodeMetricses()
		nodeMetrics, err := nodeMetricsClient.Get(context.TODO(), nodeName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Error getting node metrics for node %s: %v", nodeName, err)
			return
		}

		cpuUsage := nodeMetrics.Usage["cpu"]
		cpuMilliValue := cpuUsage.MilliValue()
		log.Printf("Node %s CPU Usage: %dm\n", nodeName, cpuMilliValue)

		// Check if CPU usage is below the threshold
		if cpuMilliValue <= shutdownThresholdInt {
			// Increment the consecutive checks counter
			checksBelowThreshold++
		} else {
			// Reset the consecutive checks counter
			checksBelowThreshold = 0
		}

		// If CPU usage has been below the threshold for 10 consecutive checks, take action
		if checksBelowThreshold >= shutdownDelayInt {
			// Send SSH commands to a Linux VM (example)
			sshConfig := &ssh.ClientConfig{
				User: sshUser,
				Auth: []ssh.AuthMethod{
					// Use the PublicKeys method for remote authentication.
					ssh.PublicKeys(signer),
				},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         0,
			}
			sshClient, err := ssh.Dial("tcp", sshHost, sshConfig)
			if err != nil {
				log.Printf("Error establishing SSH connection: %v", err)
				return
			}
			defer sshClient.Close()

			session, err := sshClient.NewSession()
			if err != nil {
				log.Printf("Error creating SSH session: %v", err)
				return
			}
			defer session.Close()

			// Execute an SSH command (example: echo "Hello, world!")
			output, err := session.CombinedOutput("sudo /usr/sbin/shutdown now")
			if err != nil {
				log.Printf("Error executing SSH command: %v", err)
				panic(nil)
			}
			fmt.Printf("%s", output)
			checksBelowThreshold = 0
		}
	}

	// Execute the function immediately and then at every minute interval
	checkAndSSH()
	for range timer.C {
		checkAndSSH()
	}

	stopCh := make(chan struct{})
	// Run the application indefinitely (or until manually terminated)
	<-stopCh
}
