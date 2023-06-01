package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
  "sync"
)

type job struct {
	Id      int
	Command string
	Status  string
}

type jobQueue struct {
	queue []job
	mutex sync.Mutex
}

var queue jobQueue
var poped_queue jobQueue


func main() {
	var addr string
	var clientPort int
	var nodePort int
	var certPath string
	var keyPath string
	var caCertPath string
	var caCertPool *x509.CertPool

	flag.StringVar(&addr, "addr", "localhost", "server address (hostname or IP)")
	flag.IntVar(&clientPort, "client-port", 8443, "port number for client communication")
	flag.IntVar(&nodePort, "node-port", 8444, "port number for node communication")
	flag.StringVar(&certPath, "cert", ".server_keys/server.crt", "path to server certificate file")
	flag.StringVar(&keyPath, "key", ".server_keys/server.key", "path to server private key file")
	flag.StringVar(&caCertPath, "cacert", ".client_keys/client.crt", "path to CA certificate file")
	flag.Parse()

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Fatalf("failed to load server key pair: %s", err)
	}

	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		log.Fatalf("failed to read CA certificate: %s", err)
	}

	caCertPool = x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientTLSConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientAuth:         tls.RequireAndVerifyClientCert,
		ClientCAs:          caCertPool,
		InsecureSkipVerify: true,
	}
	nodeTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	clientServer := &http.Server{
		Addr:      fmt.Sprintf("%s:%d", addr, clientPort),
		TLSConfig: clientTLSConfig,
	}
	nodeServer := &http.Server{
		Addr:      fmt.Sprintf("%s:%d", addr, nodePort),
		TLSConfig: nodeTLSConfig,
	}

	http.HandleFunc("/addJob", handleAddJob)
	http.HandleFunc("/showQueue", handleShowQueue)
	http.HandleFunc("/getJob", handleGetJob)
	http.HandleFunc("/applyJobState", handleApplyJobState)

	go func() {
		log.Fatal(clientServer.ListenAndServeTLS(certPath, keyPath))
	}()

	log.Fatal(nodeServer.ListenAndServeTLS(certPath, keyPath))
}

func handleAddJob(w http.ResponseWriter, r *http.Request) {
  queue.mutex.Lock()
  defer queue.mutex.Unlock()
  poped_queue.mutex.Lock()
  defer poped_queue.mutex.Unlock()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 {
		http.Error(w, "Mutual TLS required", http.StatusUnauthorized)
		return
	}

  var j job
  job_str, err := ioutil.ReadAll(r.Body)
  if err != nil {
      http.Error(w, "Invalid request", http.StatusBadRequest)
      return
  }
  j.Command = string(job_str)

	decoded_job, _ := url.QueryUnescape(string(job_str))
	log.Printf("Add new job %s from %s.", decoded_job, r.RemoteAddr)
	j.Id = len(queue.queue) + len(poped_queue.queue)
	j.Status = "waiting"
	queue.queue = append(queue.queue, j)

	w.WriteHeader(http.StatusCreated)
}

func handleShowQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("Request Show from %s.", r.RemoteAddr)

	fmt.Fprintf(w, "ID, COMMAND, STATUS\n")
	for _, j := range queue.queue {
		decoded_command, _ := url.QueryUnescape(j.Command)
		fmt.Fprintf(w, "%d, %s, %s\n", j.Id, decoded_command, j.Status)
	}
	for _, j := range poped_queue.queue {
		decoded_command, _ := url.QueryUnescape(j.Command)
		fmt.Fprintf(w, "%d, %s, %s\n", j.Id, decoded_command, j.Status)
	}
}

func handleGetJob(w http.ResponseWriter, r *http.Request) {
  queue.mutex.Lock()
  defer queue.mutex.Unlock()
  poped_queue.mutex.Lock()
  defer poped_queue.mutex.Unlock()

	if r.TLS == nil {
		http.Error(w, "TLS required", http.StatusUnauthorized)
		return
	}

	if len(queue.queue) == 0 {
		http.Error(w, "Queue is empty", http.StatusNoContent)
		return
	}
	j := queue.queue[0]
	queue.queue = queue.queue[1:]
	poped_queue.queue = append(poped_queue.queue, j)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
  err := json.NewEncoder(w).Encode(j)
	if err != nil {
		log.Print(err)
		return
	}
	log.Printf("Get Job from %s. Send Job %d: %s", r.RemoteAddr, j.Id, j.Command)
}

func handleApplyJobState(w http.ResponseWriter, r *http.Request) {


	if r.TLS == nil {
		http.Error(w, "TLS required", http.StatusUnauthorized)
		return
	}

	var j job
	if _, err := fmt.Fscanf(r.Body, "%d %s", &j.Id, &j.Status); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
  poped_queue.mutex.Lock()
  defer poped_queue.mutex.Unlock()

	for i, job := range poped_queue.queue {
		if job.Id == j.Id {
			poped_queue.queue[i].Status = j.Status
			break
		}
	}
  log.Printf("Job ID %d updated to %s", j.Id, j.Status)
	w.WriteHeader(http.StatusOK)
}
