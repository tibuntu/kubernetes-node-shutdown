# kubernetes-node-shutdown

This is a simple Golang client that watches the Kubernetes metrics API.

It will determine the CPU usage of a specific node and if it's below a threshold
for n consecutive minutes, attempt to send a shutdown command via SSH.

I wrote this tool because I have a dedicated Kubernetes node that runs Plex Media Server
and if Plex isn't used and hence the node is not utilized, I want it to shut down.
