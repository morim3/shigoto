package main

import (
    "bytes"
    "crypto/tls"
    "crypto/x509"
    "flag"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
)

var (
    serverAddr  string
    job         string
    showQueue   bool
    caCertPath  string
    clientCert  string
    clientKey   string
)


func main() {
  flag.StringVar(&serverAddr, "server", "localhost:8443", "server address (hostname:port)")
    flag.StringVar(&job, "job", "sleep 5", "job to add to queue")
    flag.BoolVar(&showQueue, "show-queue", true, "show job queue")
    flag.StringVar(&caCertPath, "server-cert", ".server_keys/server.crt", "path to CA certificate file")
    flag.StringVar(&clientCert, "client-cert", ".client_keys/client.crt", "path to client certificate file")
    flag.StringVar(&clientKey, "client-key", ".client_keys/client.key", "path to client private key file")
    flag.Parse()

    if serverAddr == "" {
        fmt.Fprintln(os.Stderr, "server address must be specified")
        os.Exit(1)
    }
    // Load CA certificate
    caCert, err := ioutil.ReadFile(caCertPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to read CA certificate file: %v\n", err)
        os.Exit(1)
    }
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)

    // Load client certificate and key
    clientCert, err := tls.LoadX509KeyPair(clientCert, clientKey)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load client certificate and key: %v\n", err)
        os.Exit(1)
    }

    // Create HTTP client with mutual TLS
    client := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                RootCAs:      caCertPool,
                Certificates: []tls.Certificate{clientCert},
                InsecureSkipVerify: true,
            },
        },
    }

    // Send job to server
    if job != "" {
        reqBody := bytes.NewBufferString(job)
        _, err := client.Post("https://"+serverAddr+"/addJob", "text/plain", reqBody)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Failed to add job: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("Job added to queue")
    }

    // Show job queue
    if showQueue {
        resp, err := client.Get("https://" + serverAddr + "/showQueue")
        if err != nil {
            fmt.Fprintf(os.Stderr, "Failed to show job queue: %v\n", err)
            os.Exit(1)
        }
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Failed to read response body: %v\n", err)
            os.Exit(1)
        }
        fmt.Println(string(body))
    }
}
