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

	dryRun := os.Getenv("DRY_RUN_MODE")
	if dryRun == "" {
		dryRun = "false"
		log.Printf("Environment variable DRY_RUN_MODE is not set. Using false as default.")
	} else if dryRun == "true" {
		log.Printf("Running in dry run mode. Not performing shutdown.")
	}

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatal("Environment variable NODE_NAME is not set")
		panic(nil)
	}

	filePath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if filePath == "" {
		log.Fatal("Environment variable SSH_PRIVATE_KEY_PATH is not set")
		panic(nil)
	}

	sshUser := os.Getenv("SSH_USER_NAME")
	if sshUser == "" {
		log.Fatal("Environment variable SSH_USER_NAME is not set")
		panic(nil)
	}

	sshPort := os.Getenv("SSH_PORT")
	if sshPort == "" {
		sshPort = "22"
		log.Printf("Environment variable SSH_PORT is not set. Using 22 as default.")
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

	MemoryThreshold := os.Getenv("MEMORY_THRESHOLD")
	CPUthreshold := os.Getenv("CPU_THRESHOLD")

	if CPUthreshold == "" && MemoryThreshold == "" {
		log.Print("No shutdown threshold set.")
		log.Fatal("At least one of CPU_THRESHOLD or MEMORY_THRESHOLD must be set!")
		panic(nil)
	}

	shutdownDelayMinutes := os.Getenv("SHUTDOWN_DELAY_MINUTES")
	if shutdownDelayMinutes == "" {
		log.Fatal("Environment variable SHUTDOWN_DELAY_MINUTES is not set")
		panic(nil)
	}

	shutdownDelayInt, err := strconv.Atoi(shutdownDelayMinutes)
	if err != nil {
		log.Fatalf("%v", err)
		panic(nil)
	}

	// Create a timer that triggers every minute
	timer := time.NewTicker(1 * time.Minute)

	checksBelowThreshold := 0
	checkAndSSH := func() {
		nodeMetricsClient := metricsClient.MetricsV1beta1().NodeMetricses()
		nodeMetrics, err := nodeMetricsClient.Get(context.TODO(), nodeName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Error getting node metrics for node %s: %v", nodeName, err)
			return
		}

		if MemoryThreshold != "" {
			MemoryThresholdInt, err := strconv.ParseInt(MemoryThreshold, 0, 64)
			if err != nil {
				log.Fatalf("%v", err)
				panic(nil)
			}
			memoryUsage := nodeMetrics.Usage["memory"]
			memoryValue := memoryUsage.Value() / 1024 / 1024
			log.Printf("Node %s memory usage: %dMB", nodeName, memoryValue)
			if memoryValue <= MemoryThresholdInt {
				checksBelowThreshold++
				remainingMinutes := shutdownDelayInt - checksBelowThreshold
				log.Printf("Memory usage below configured threshold. Shutting down in %d minutes.", remainingMinutes)
			} else {
				log.Printf("Memory usage above configured threshold. Resetting shutdown delay back to %d minutes", shutdownDelayInt)
				checksBelowThreshold = 0
			}
		}

		if CPUthreshold != "" {
			CPUthresholdInt, err := strconv.ParseInt(CPUthreshold, 10, 64)
			if err != nil {
				log.Fatalf("%v", err)
				panic(nil)
			}
			cpuUsage := nodeMetrics.Usage["cpu"]
			cpuValue := cpuUsage.MilliValue()
			log.Printf("Node %s CPU usage: %dm", nodeName, cpuValue)
			if cpuValue <= CPUthresholdInt {
				checksBelowThreshold++
				remainingMinutes := shutdownDelayInt - checksBelowThreshold
				log.Printf("CPU usage below configured threshold. Shutting down in %d minutes.", remainingMinutes)
			} else {
				log.Printf("CPU usage above configured threshold. Resetting shutdown delay back to %d minutes", shutdownDelayInt)
				checksBelowThreshold = 0
			}
		}

		if checksBelowThreshold >= shutdownDelayInt {
			if dryRun == "true" {
				log.Printf("Shutdown delay has been reached. If you want to send an actual shutdown signal, disable dry run mode.")
				checksBelowThreshold = 0
			} else {
				sshConfig := &ssh.ClientConfig{
					User: sshUser,
					Auth: []ssh.AuthMethod{
						// Use the PublicKeys method for remote authentication.
						ssh.PublicKeys(signer),
					},
					HostKeyCallback: ssh.InsecureIgnoreHostKey(),
					Timeout:         0,
				}
				sshClient, err := ssh.Dial("tcp", nodeName+":"+sshPort, sshConfig)
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

				log.Printf("Shutdown delay reached. Sending shutdown signal to node.")
				output, err := session.CombinedOutput("sudo /usr/sbin/shutdown now")
				if err != nil {
					log.Printf("Error executing SSH command: %v", err)
					panic(nil)
				}
				fmt.Printf("%s", output)
				checksBelowThreshold = 0
			}
		}
	}

	checkAndSSH()
	for range timer.C {
		checkAndSSH()
	}

	stopCh := make(chan struct{})
	<-stopCh
}
