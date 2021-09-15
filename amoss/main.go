package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cliffordlab/amoss_services/amoss_login"
	"github.com/cliffordlab/amoss_services/amoss_streams"
	"github.com/cliffordlab/amoss_services/amoss_streams/moyo_mom/emory"
	"github.com/cliffordlab/amoss_services/bp_readings"
	"github.com/cliffordlab/amoss_services/capacity"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/amoss_services/download"
	"github.com/cliffordlab/amoss_services/fhir"
	"github.com/cliffordlab/amoss_services/garminauth"
	"github.com/cliffordlab/amoss_services/handlers"
	"github.com/cliffordlab/amoss_services/health"
	"github.com/cliffordlab/amoss_services/participant"
	"github.com/cliffordlab/amoss_services/vault"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var svc *s3.S3
var gMux *mux.Router

var (
	devPnt        *bool
	prodPnt       *bool
	localPnt      *bool
	falseEnvCount int
	environment   string
)

func init() {
	// OpenSource: This will be remove for OpenSource. Only environment test
	devPnt = flag.Bool("dev", false, "flag for development environment")
	prodPnt = flag.Bool("prod", false, "flag for prod environment")
	localPnt = flag.Bool("local", false, "flag for local environment")
	flag.Parse()

	envs := []bool{*devPnt, *prodPnt, *localPnt}
	//configuring environment for application using the flags
	falseEnvCount = 0
	for index, env := range envs {
		log.Printf("### index: " + strconv.Itoa(int(index)))
		log.Printf("### env: ", env)
		if env == false {
			falseEnvCount = falseEnvCount + 1
		} else {
			switch index {
			case 0:
				environment = "dev"
			case 1:
				environment = "prod"
			case 2:
				environment = "local"
			default:
				environment = "dev"
			}
		}
	}

	if environment == "local" {
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		svc = s3.New(sess)
	} else {
		svc = s3.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))
	}

	gMux = mux.NewRouter()

	setHandlers(svc)
}

func main() {

	log.Printf("Starting %s server...", environment)
	database.ADB.Environment = environment

	rand.Seed(time.Now().UTC().UnixNano())

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

	var secretVaultEndpoint = "secret/amoss"

	// Get JSON Web Token
	log.Println("Getting JWT secrets...")
	secret, err := vc.GetJWTSecret(secretVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	capacity.JwtSecret = secret

	// Get Cryptography secret key
	log.Println("Getting Encryption key...")
	cryptoKey, err := vc.GetCryptoKey(secretVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	capacity.CryptoKey = cryptoKey

	// Get Garmin Secret Key
	log.Println("Getting Garmin secret...")
	garminSecret, err := vc.GetGarminSecret(secretVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	capacity.GarminSecret = garminSecret

	log.Println("Getting Garmin token...")

	garminKey, err := vc.GetGarminToken(secretVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	capacity.GarminToken = garminKey

	// Lamda API Key
	log.Println("Getting AWS SES Lambda API Key...")
	apiKey, err := vc.GetEmailLambdaAPIKey(secretVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	capacity.AWSSESLambdaAPIKey = apiKey

	// Get Database credential
	log.Println("Getting DBCreds secrets...")
	var secretDBVaultEndpoint = "secret/amossDB"

	if *devPnt {
		secretDBVaultEndpoint = "secret/amossDB/Dev"
	}

	creds, err := vc.GetDbCreds(secretDBVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}

	// OpenSource: This will be changed for OpenSource. Only one db call
	// Connect to Database
	log.Println("AMoSS application is starting up")
	if *prodPnt {
		log.Println("Running production db")
		database.InitDb(creds.DBUser, creds.DBUserPW, creds.DBAddr, "amoss")
	} else if *devPnt {
		log.Println("Running development db")
		database.InitDb(creds.DBUser, creds.DBUserPW, creds.DBAddr, "amoss_dev")
	} else {
		log.Println("Running local db")
		database.InitDb("postgres", "password", "localhost", "amoss")
	}

	handler := cors.New(cors.Options{
		AllowedOrigins:     []string{"*"},
		AllowedMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:     []string{"Origin", "Accept", "Accept-Language", "Content-Type", "Authorization", "Access-Control-Allow-Headers", "X-Requested-With"},
		AllowCredentials:   true,
		OptionsPassthrough: true,
	}).Handler(gMux)

	log.Println("Starting amoss application server on port :8080")
	go func() {
		log.Fatalln(http.ListenAndServe(":8080", handler))
	}()

	quit := make(chan os.Signal, 1)
	//renew vault token
	go vc.AutomateVaultTokenRenewal()
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("Shutdown signal received, exiting...\n")
}

func setHandlers(svc *s3.S3) {
	log.Println("Setting Handlers..")
	s := gMux.PathPrefix("/api/moyo/mom/emory").Subrouter()
	s.Handle("/participants", handlers.HandleReqWithBearerToken(participant.ListParticipantsHandler{Name: "list participants handler"}))
	s.Handle("/participants/{participant_id:[0-9]+}/charts", handlers.HandleReqWithBearerToken(participant.VitalChartHandler{Name: "query db to visualize vital chart"}))
	s.Handle("/participants/{participant_id:[0-9]+}/vitals/unverified_uploads", handlers.HandleReqWithBearerToken(participant.ListUnverifiedFilesHandler{Name: "list unverified files handler"}))
	s.Handle("/participants/{participant_id:[0-9]+}/vitals/unverified_uploads/{created_at:[0-9]+}", handlers.HandleReqWithBearerToken(participant.UnverifiedBPFileHandler{Name: "unverified bp file handler", Svc: svc}))
	s.Handle("/vitals/upload", handlers.HandleReq(emory.UploadMMEVitalsHandler{Name: "moyo mom emory vitals upload handler", Svc: svc}))
	s.Handle("/symptoms/upload", handlers.HandleReq(emory.UploadMMESymptomsHandler{Name: "moyo mom emory symptoms upload handler", Svc: svc}))

	gMux.Handle("/api/createCoordinator", handlers.HandleReqWithBearerToken(amoss_login.RegistrationHandler{Name: "registration handler"}))
	gMux.Handle("/api/createPatient", handlers.HandleReqWithBearerToken(amoss_login.RegistrationHandler{Name: "registration handler"}))
	gMux.Handle("/api/passwordRevocery", handlers.HandleReqWithBearerToken(participant.PasswordRecoveryHandler{Name: "password recovery handler"}))
	gMux.Handle("/api/getUniqueID", handlers.HandleReqWithBearerToken(participant.IDGenerationHandler{Name: "ID generation handler"}))
	gMux.Handle("/loginParticipant", handlers.HandleReq(amoss_login.LoginHandler{Name: "login handler"}))
	gMux.Handle("/api/addGarmin", handlers.HandleReqWithBearerToken(garminauth.GarminAccessTokenHandler{Name: "add garmin handler"}))
	gMux.Handle("/api/garmin_uauth_token", handlers.HandleReqWithBearerToken(garminauth.GarminUnauthorizedRequestHandler{Name: "garmin request token handler"}))
	gMux.Handle("/api/utsw/fhir/filter", handlers.HandleReq(fhir.FhirFilterHandler{Name: "upload utsw fhir handler", Svc: svc}))
	gMux.Handle("/api/upload_s3", handlers.HandleReq(amoss_streams.UploadHandler{Name: "upload s3 handler", Svc: svc}))
	gMux.Handle("/api/moyo/upload_s3", handlers.HandleReq(amoss_streams.UploadMoyoHandler{Name: "upload moyo handler", Svc: svc}))
	gMux.Handle("/api/moyo/register", handlers.HandleReq(amoss_login.MoyoRegistrationHandler{Name: "moyo registration handler"}))
	gMux.Handle("/api/moyo/moyo-mom/bp/{participant_id:[0-9]+}", handlers.HandleReq(bp_readings.QueryHandler{Name: "query bp handler"}))
	gMux.Handle("/api/moyo/download", handlers.HandleReq(download.APKDownloadHandler{Name: "Download MSM handler", Svc: svc}))
	gMux.HandleFunc("/api/health", health.Handler)
	// If unable to create new Garmin Health API consumer and secret for Dev environment, than:
	// In dev environment this handler will never be called.
	// Garmin Health API's endpoint configuration console can only be set up with one end point.
	// This means production server will need to handle the routing of dev environments Garmin summary uploads.
	gMux.Handle("/garmin/ping", handlers.HandleReq(garminauth.GarminPingHandler{Name: "garmin ping handler", Svc: svc}))
	gMux.Handle("/upload", handlers.HandleReq(amoss_streams.UploadHandler{Name: "upload handler", Svc: svc}))

	// Create room for static files serving
	gMux.PathPrefix("/prod/moyo-beta").Handler(http.StripPrefix("/prod/moyo-beta", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/consent-form").Handler(http.StripPrefix("/prod/consent-form", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/moyo/download").Handler(http.StripPrefix("/prod/moyo/download", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/hf/download").Handler(http.StripPrefix("/prod/hf/download", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/utsw/download").Handler(http.StripPrefix("/prod/utsw/download", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/login").Handler(http.StripPrefix("/prod/login", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/account").Handler(http.StripPrefix("/prod/account", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/admin").Handler(http.StripPrefix("/prod/admin", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/logo").Handler(http.StripPrefix("/prod/logo", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/utsw").Handler(http.StripPrefix("/prod/utsw", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/about").Handler(http.StripPrefix("/prod/about", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/query").Handler(http.StripPrefix("/prod/query", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/documentation").Handler(http.StripPrefix("/prod/documentation", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/coordinator").Handler(http.StripPrefix("/prod/coordinator", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/participant").Handler(http.StripPrefix("/prod/participant", http.FileServer(http.Dir("prod"))))
	gMux.PathPrefix("/prod/moyo/mom/emory/participants").Handler(http.StripPrefix("/prod/moyo/mom/emory/participants", http.FileServer(http.Dir("prod"))))

	gMux.PathPrefix("/prod").Handler(http.StripPrefix("/prod", http.FileServer(http.Dir("prod"))))
	gMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		http.ServeFile(w, r, "prod/index.html")
	})
}
