package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/cliffordlab/amoss_services/capacity"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/amoss_services/handlers"
	"github.com/cliffordlab/amoss_services/vault"
	"github.com/rs/cors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func main() {
	mux := http.NewServeMux()
	svc := dynamodb.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))
	dynamoHandler := handlers.DynamoHandler{Name: "dynamo handler", Svc: svc}
	mux.Handle("api/query", handlers.HandleReq(dynamoHandler))

	prodPtr := flag.Bool("prod", false, "set environment to prod")
	flag.Parse()
	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		log.Fatalln("VAULT_TOKEN must be set and non-empty")
	}
	//get vault address from env variables
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		log.Fatalln("VAULT_ADDR must be set and non-empty")
	}

	pwd, _ := os.Getwd()
	caPath := pwd + "/incommonca.crt"

	// Load CA cert
	caCert, err := ioutil.ReadFile(caPath)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	vc, err := vault.NewVaultClient(vaultAddr, vaultToken, client)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Getting JWT secret...")
	secretJWTVaultEndpoint := "secret/amoss"

	secret, err := vc.GetJWTSecret(secretJWTVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	capacity.JwtSecret = secret

	log.Println("Getting Jawbone secret...")
	jawboneSecretEndpoint := "secret/amossJB"
	jbSecret, err := vc.GetJawboneSecret(jawboneSecretEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	handlers.JawboneClientSecret = jbSecret

	log.Println("Getting DBCreds secret...")
	secretDBVaultEndpoint := "secret/amossDB"

	creds, err := vc.GetDbCreds(secretDBVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("AMoSS login is starting up")
	if *prodPtr {
		log.Println("Running production db")
		database.InitDb(creds.DBUser, creds.DBUserPW, creds.DBAddr)
	} else {
		log.Println("Running development db")
		database.InitDb(creds.DBUser, creds.DBUserPW, "localhost")
	}

	handler := cors.New(cors.Options{
		AllowedOrigins:     []string{"*"},
		AllowedMethods:     []string{"GET", "POST", "PATCH", "OPTIONS"},
		AllowedHeaders:     []string{"Accept", "Accept-Language", "Content-Type", "authorization"},
		AllowCredentials:   true,
		OptionsPassthrough: true,
	}).Handler(mux)

	log.Println("Starting amoss login application server on port :8082")
	go func() {
		log.Fatalln(http.ListenAndServe(":8082", handler))
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("Shutdown signal received, exiting...\n")
}
