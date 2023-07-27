# Use an official Golang runtime as a base image
FROM golang:1.20

# Set the working directory inside the container
WORKDIR /go/src/app

# Copy the current directory contents into the container at /go/src/app
COPY . .

# Build the Go application
RUN go build -o ProfileManagementService

# Expose the port that the service listens on (replace 8081 with the actual port if different)
EXPOSE 8081

# Command to run the ProfileManagementService binary when the container starts
CMD ["./ProfileManagementService"]