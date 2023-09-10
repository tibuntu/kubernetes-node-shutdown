# Use an ARM-based image as the build environment
FROM arm64v8/golang:1.21 as builder

# Set the working directory in the container
WORKDIR /app

# Copy the Go application source code into the container
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOARCH=arm GOARM=7 go build -o kubernetes-node-shutdown

# Final stage: Create a minimal container to run the application
FROM scratch

# Copy the built binary from the builder container to the final container
COPY --from=builder /app/kubernetes-node-shutdown /kubernetes-node-shutdown

# Set the entrypoint
ENTRYPOINT ["/kubernetes-node-shutdown"]
