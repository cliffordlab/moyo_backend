package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/amoss_services/jawbone"
	"github.com/cliffordlab/amoss_services/vault"
	"github.com/jasonlvhit/gocron"
)

func main() {
	log.Println("Start Jawbone Job")

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

	log.Println("Getting DBCreds secret...")
	secretDBVaultEndpoint := "secret/amossDB"

	creds, err := vc.GetDbCreds(secretDBVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}

	if *prodPtr {
		log.Println("Running production db")
		database.InitDb(creds.DBUser, creds.DBUserPW, creds.DBAddr)
	} else {
		log.Println("Running development db")
		database.InitDb(creds.DBUser, creds.DBUserPW, "localhost")
	}

	gocron.Every(1).Day().At("16:16").Do(jawboneTask)
	<-gocron.Start()
}

func jawboneTask() {
	jawboneClient := &http.Client{}

	jawbone.PutJawboneFileInS3(jawboneClient)
}
