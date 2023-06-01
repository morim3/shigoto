package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

type job struct {
	Id      int
	Command string
	Status  string
}

func main() {
	var serverAddr string
	var serverPort int
	var certPath string
	var numWorkers int

	flag.StringVar(&serverAddr, "addr", "localhost", "server address (hostname or IP)")
	flag.IntVar(&serverPort, "port", 8444, "port number for server communication")
	flag.StringVar(&certPath, "cert", ".server_keys/server.crt", "path to server certificate file")
	flag.IntVar(&numWorkers, "workers", 1, "number of concurrent workers")
	flag.Parse()

	tlsConfig := &tls.Config{
		RootCAs:            loadCACert(certPath),
		ServerName:         serverAddr,
		InsecureSkipVerify: true,
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	getJobURL := fmt.Sprintf("https://%s:%d/getJob", serverAddr, serverPort)
	applyJobStateURL := fmt.Sprintf("https://%s:%d/applyJobState", serverAddr, serverPort)

	wg := &sync.WaitGroup{}
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				j, err := getJob(client, getJobURL)
				if err != nil {
					log.Fatalf("failed to get job: %s", err)
				}
				if j == nil {
					continue
				}
				log.Printf("Got job %d: %s", j.Id, j.Command)
				runJob(j.Id, j.Command, client, applyJobStateURL)
				time.Sleep(1 * time.Second)
			}
		}()
	}

	wg.Wait()
}

func getJob(client *http.Client, url string) (*job, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var j job
	err = json.NewDecoder(resp.Body).Decode(&j)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func runJob(id int, command string, client *http.Client, url string) {
	err := updateJobStatus(id, "running", client, url)
	if err != nil {
		log.Fatalf("failed to update job status: %s", err)
	}

	log.Printf("Start Job %d: %s", id, command)
	cmd := exec.Command("/bin/sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
    err = updateJobStatus(id, "failed", client, url)
		log.Printf("Job %d failed: %s", id, err)
	} else {
    err = updateJobStatus(id, "done", client, url)
		log.Printf("Job %d done: %s", id, string(output))
	}

	if err != nil {
		log.Fatalf("failed to update job status: %s", err)
	}
}

func updateJobStatus(id int, status string, client *http.Client, url string) error {

	reqBody := bytes.NewBufferString(fmt.Sprintf("%d %s", id, status))
  resp, _ := client.Post(url, "text/plain", reqBody)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
func loadCACert(certPath string) *x509.CertPool {
	caCert, err := ioutil.ReadFile(certPath)
	if err != nil {
		log.Fatalf("failed to read CA certificate: %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	return caCertPool
}
