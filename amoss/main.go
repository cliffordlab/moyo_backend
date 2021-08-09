package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
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
	"github.com/cliffordlab/amoss_services/support"
	"github.com/cliffordlab/amoss_services/vault"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//var serveMux *http.ServeMux
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
	devPnt = flag.Bool("dev", false, "flag for development environment")
	prodPnt = flag.Bool("prod", false, "flag for prod environment")
	localPnt = flag.Bool("local", false, "flag for local environment")
	flag.Parse()

	envs := []bool{*devPnt, *prodPnt, *localPnt}
	//configuring environment for application using the flags
	falseEnvCount = 0
	for index, env := range envs {
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
	if len(environment) == 0 {
		environment = "dev"
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

	if environment == "prod" {
		setProdHandlers(svc)
	} else {
		setDevHandlers("/dev", svc)
	}

}

type EmailHandler struct {
	Name string
	Svc  *s3.S3
	Type string
}

func main() {
	//if falseEnvCount == 2 {
	//	fmt.Println("Please specify environment arguments")
	//	return
	//}

	log.Printf("Starting %s server...", environment)
	//prodPtr := flag.Bool("prod", false, "set environment to prod")
	//flag.Parse()

	database.ADB.Environment = environment

	//if *prodPnt {
	//	emailHandler := handlers.EmailHandler{Name: "email handler", Svc: svc, Type: "emails"}
	//	serveMux.Handle("/api/emails", handlers.HandleReq(emailHandler))
	//}
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

	var secretVaultEndpoint = "secret/amoss/dev"
	if *prodPnt {
		secretVaultEndpoint = "secret/amoss"
	}

	log.Println("Getting JWT secrets...")
	secret, err := vc.GetJWTSecret(secretVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	capacity.JwtSecret = secret

	log.Println("Getting Encryption key...")
	cryptoKey, err := vc.GetCryptoKey(secretVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	capacity.CryptoKey = cryptoKey

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

	log.Println("Getting AWS SES Lambda API Key...")

	apiKey, err := vc.GetEmailLambdaAPIKey(secretVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	capacity.AWSSESLambdaAPIKey = apiKey

	log.Println("Getting DBCreds secrets...")
	var secretDBVaultEndpoint = "secret/amossDB"

	if *devPnt {
		secretDBVaultEndpoint = "secret/amossDB/Dev"
	}

	creds, err := vc.GetDbCreds(secretDBVaultEndpoint)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("AMoSS application is starting up")
	if *prodPnt {
		log.Println("Running production db")
		database.InitDb(creds.DBUser, creds.DBUserPW, creds.DBAddr, "amoss")
	} else if *devPnt {
		log.Println("Running development db")
		database.InitDb(creds.DBUser, creds.DBUserPW, creds.DBAddr, "amoss_dev")
	} else {
		log.Println("Running local db")
		database.InitDb("postgres", "", "localhost", "amoss")
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

func setProdHandlers(svc *s3.S3) {
	log.Println("Setting Production Handlers..")
	s := gMux.PathPrefix("/api/moyo/mom/emory").Subrouter()
	s.Handle("/participants", handlers.HandleReqWithBearerToken(participant.ListParticipantsHandler{Name: "list participants handler"}))
	s.Handle("/participants/{participant_id:[0-9]+}/charts", handlers.HandleReqWithBearerToken(participant.VitalChartHandler{Name: "query db to visualize vital chart"}))
	//s.Handle("/participants/{participant_id}/vitals/unverified_uploads", handlers.HandleReqWithBearerToken(participant.VitalChartHandler{Name: "query db to visualize vital chart"}))
	s.Handle("/participants/{participant_id:[0-9]+}/vitals/unverified_uploads", handlers.HandleReqWithBearerToken(participant.ListUnverifiedFilesHandler{Name: "list unverified files handler"}))
	s.Handle("/participants/{participant_id:[0-9]+}/vitals/unverified_uploads/{created_at:[0-9]+}", handlers.HandleReqWithBearerToken(participant.UnverifiedBPFileHandler{Name: "unverified bp file handler", Svc: svc}))
	s.Handle("/vitals/upload", handlers.HandleReq(emory.UploadMMEVitalsHandler{Name: "moyo mom emory vitals upload handler", Svc: svc}))
	s.Handle("/symptoms/upload", handlers.HandleReq(emory.UploadMMESymptomsHandler{Name: "moyo mom emory symptoms upload handler", Svc: svc}))
	//gMux.Handle("/api/createAdmin", handlers.HandleReq(amoss_login.RegistrationHandler{Name: "registration handler"}))

	gMux.Handle("/api/createCoordinator", handlers.HandleReqWithBearerToken(amoss_login.RegistrationHandler{Name: "registration handler"}))
	gMux.Handle("/api/createPatient", handlers.HandleReqWithBearerToken(amoss_login.RegistrationHandler{Name: "registration handler"}))
	gMux.Handle("/api/getUniqueID", handlers.HandleReqWithBearerToken(participant.IDGenerationHandler{Name: "ID generation handler"}))
	gMux.Handle("/loginParticipant", handlers.HandleReq(amoss_login.LoginHandler{Name: "login handler"}))
	gMux.Handle("/api/addGarmin", handlers.HandleReqWithBearerToken(garminauth.GarminAccessTokenHandler{Name: "add garmin handler"}))
	gMux.Handle("/api/garmin_uauth_token", handlers.HandleReqWithBearerToken(garminauth.GarminUnauthorizedRequestHandler{Name: "garmin request token handler"}))
	gMux.Handle("/api/utsw/fhir/filter", handlers.HandleReq(fhir.FhirFilterHandler{Name: "upload utsw fhir handler", Svc: svc}))
	gMux.Handle("/api/upload_s3", handlers.HandleReq(amoss_streams.UploadHandler{Name: "upload s3 handler", Svc: svc}))
	gMux.Handle("/api/hf/upload_s3", handlers.HandleReq(amoss_streams.UploadHFHandler{Name: "upload hf handler", Svc: svc}))
	gMux.Handle("/api/utsw/upload_s3", handlers.HandleReq(amoss_streams.UploadUTSWHandler{Name: "upload utsw handler", Svc: svc}))
	gMux.Handle("/api/moyo/upload_s3", handlers.HandleReq(amoss_streams.UploadMoyoHandler{Name: "upload moyo handler", Svc: svc}))
	gMux.Handle("/api/moyo/register", handlers.HandleReq(amoss_login.MoyoRegistrationHandler{Name: "moyo registration handler"}))
	gMux.Handle("/api/moyo/test/decrypt/email", handlers.HandleReq(support.MoyoDecryptEmailHandler{Name: "moyo decrypt email handler"}))
	gMux.Handle("/api/moyo/test/decrypt/phone", handlers.HandleReq(support.MoyoDecryptPIDHandler{Name: "moyo decrypt phone handler"}))
	gMux.Handle("/api/moyo/beta/authenticate_participant", handlers.HandleReq(amoss_login.MoyoBetaRegistrationHandler{Name: "moyo beta registration handler"}))
	gMux.Handle("/api/moyo/beta/send_forgot_email", handlers.HandleReq(support.ContactSupportHandler{Name: "moyo beta contact support for forgotten email"}))
	gMux.Handle("/api/moyo/beta/send_password_error", handlers.HandleReq(support.ContactSupportHandler{Name: "moyo beta contact support for password error"}))
	gMux.Handle("/api/moyo/moyo-mom/bp/{participant_id:[0-9]+}", handlers.HandleReq(bp_readings.QueryHandler{Name: "query bp handler"}))
	gMux.Handle("/api/moyo/download", handlers.HandleReq(download.APKDownloadHandler{Name: "Download MSM handler", Svc: svc}))
	gMux.Handle("/api/utsw/download", handlers.HandleReq(download.APKDownloadHandler{Name: "Download UTSW handler", Svc: svc}))
	gMux.Handle("/api/hf/download", handlers.HandleReq(download.APKDownloadHandler{Name: "Download HF handler", Svc: svc}))
	gMux.HandleFunc("/api/health", health.Handler)
	// If unable to create new Garmin Health API consumer and secret for prod environment, than:
	// In prod environment this handler will never be called.
	// Garmin Health API's endpoint configuration console can only be set up with one end point.
	// This means production server will need to handle the routing of prod environments Garmin summary uploads.
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
	//gMux.Handle("/prod/", http.StripPrefix("/prod/", http.FileServer(http.Dir("prod"))))
	gMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		http.ServeFile(w, r, "prod/index.html")
	})
}

func setDevHandlers(namespace string, svc *s3.S3) {
	log.Println("Setting Dev Handlers...")
	s := gMux.PathPrefix("/dev/api/moyo/mom/emory").Subrouter()
	s.Handle("/participants", handlers.HandleReqWithBearerToken(participant.ListParticipantsHandler{Name: "list participants handler"}))
	s.Handle("/participants/{participant_id:[0-9]+}/charts", handlers.HandleReqWithBearerToken(participant.VitalChartHandler{Name: "query db to visualize vital chart"}))
	//s.Handle("/participants/{participant_id}/vitals/unverified_uploads", handlers.HandleReqWithBearerToken(participant.VitalChartHandler{Name: "query db to visualize vital chart"}))
	s.Handle("/participants/{participant_id:[0-9]+}/vitals/unverified_uploads", handlers.HandleReqWithBearerToken(participant.ListUnverifiedFilesHandler{Name: "list unverified files handler"}))
	s.Handle("/participants/{participant_id:[0-9]+}/vitals/unverified_uploads/{created_at:[0-9]+}", handlers.HandleReqWithBearerToken(participant.UnverifiedBPFileHandler{Name: "unverified bp file handler", Svc: svc}))
	s.Handle("/vitals/upload", handlers.HandleReq(emory.UploadMMEVitalsHandler{Name: "moyo mom emory vitals upload handler", Svc: svc}))
	s.Handle("/symptoms/upload", handlers.HandleReq(emory.UploadMMESymptomsHandler{Name: "moyo mom emory symptoms upload handler", Svc: svc}))
	//gMux.Handle("/api/createAdmin", handlers.HandleReq(amoss_login.RegistrationHandler{Name: "registration handler"}))

	sr := gMux.PathPrefix(namespace).Subrouter()
	sr.Handle("/api/createCoordinator", handlers.HandleReqWithBearerToken(amoss_login.RegistrationHandler{Name: "registration handler"}))
	sr.Handle("/api/createPatient", handlers.HandleReqWithBearerToken(amoss_login.RegistrationHandler{Name: "registration handler"}))
	sr.Handle("/api/getUniqueID", handlers.HandleReqWithBearerToken(participant.IDGenerationHandler{Name: "ID generation handler"}))
	sr.Handle("/loginParticipant", handlers.HandleReq(amoss_login.LoginHandler{Name: "login handler"}))
	sr.Handle("/api/addGarmin", handlers.HandleReqWithBearerToken(garminauth.GarminAccessTokenHandler{Name: "add garmin handler"}))
	sr.Handle("/api/garmin_uauth_token", handlers.HandleReqWithBearerToken(garminauth.GarminUnauthorizedRequestHandler{Name: "garmin request token handler"}))
	sr.Handle("/api/utsw/fhir/filter", handlers.HandleReq(fhir.FhirFilterHandler{Name: "upload utsw fhir handler", Svc: svc}))
	sr.Handle("/api/upload_s3", handlers.HandleReq(amoss_streams.UploadHandler{Name: "upload s3 handler", Svc: svc}))
	sr.Handle("/api/hf/upload_s3", handlers.HandleReq(amoss_streams.UploadHFHandler{Name: "upload hf handler", Svc: svc}))
	sr.Handle("/api/utsw/upload_s3", handlers.HandleReq(amoss_streams.UploadUTSWHandler{Name: "upload utsw handler", Svc: svc}))
	sr.Handle("/api/moyo/upload_s3", handlers.HandleReq(amoss_streams.UploadMoyoHandler{Name: "upload moyo handler", Svc: svc}))
	sr.Handle("/api/moyo/register", handlers.HandleReq(amoss_login.MoyoRegistrationHandler{Name: "moyo registration handler"}))
	sr.Handle("/api/moyo/test/decrypt/email", handlers.HandleReq(support.MoyoDecryptEmailHandler{Name: "moyo decrypt email handler"}))
	sr.Handle("/api/moyo/test/decrypt/phone", handlers.HandleReq(support.MoyoDecryptPIDHandler{Name: "moyo decrypt phone handler"}))
	sr.Handle("/api/moyo/beta/authenticate_participant", handlers.HandleReq(amoss_login.MoyoBetaRegistrationHandler{Name: "moyo beta registration handler"}))
	sr.Handle("/api/moyo/beta/send_forgot_email", handlers.HandleReq(support.ContactSupportHandler{Name: "moyo beta contact support for forgotten email"}))
	sr.Handle("/api/moyo/beta/send_password_error", handlers.HandleReq(support.ContactSupportHandler{Name: "moyo beta contact support for password error"}))
	sr.Handle("/api/moyo/moyo-mom/bp/{participant_id:[0-9]+}", handlers.HandleReq(bp_readings.QueryHandler{Name: "query bp handler"}))
	sr.Handle("/api/moyo/download", handlers.HandleReq(download.APKDownloadHandler{Name: "Download MSM handler", Svc: svc}))
	sr.Handle("/api/utsw/download", handlers.HandleReq(download.APKDownloadHandler{Name: "Download UTSW handler", Svc: svc}))
	sr.Handle("/api/hf/download", handlers.HandleReq(download.APKDownloadHandler{Name: "Download HF handler", Svc: svc}))
	sr.HandleFunc("/api/health", health.Handler)
	// If unable to create new Garmin Health API consumer and secret for Dev environment, than:
	// In dev environment this handler will never be called.
	// Garmin Health API's endpoint configuration console can only be set up with one end point.
	// This means production server will need to handle the routing of dev environments Garmin summary uploads.
	sr.Handle("/garmin/ping", handlers.HandleReq(garminauth.GarminPingHandler{Name: "garmin ping handler", Svc: svc}))
	sr.Handle("/upload", handlers.HandleReq(amoss_streams.UploadHandler{Name: "upload handler", Svc: svc}))

	// Create room for static files serving
	gMux.PathPrefix("/dev/moyo-beta").Handler(http.StripPrefix("/dev/moyo-beta", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/consent-form").Handler(http.StripPrefix("/dev/consent-form", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/moyo/download").Handler(http.StripPrefix("/dev/moyo/download", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/hf/download").Handler(http.StripPrefix("/dev/hf/download", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/utsw/download").Handler(http.StripPrefix("/dev/utsw/download", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/login").Handler(http.StripPrefix("/dev/login", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/account").Handler(http.StripPrefix("/dev/account", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/admin").Handler(http.StripPrefix("/dev/admin", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/logo").Handler(http.StripPrefix("/dev/logo", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/utsw").Handler(http.StripPrefix("/dev/utsw", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/about").Handler(http.StripPrefix("/dev/about", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/query").Handler(http.StripPrefix("/dev/query", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/documentation").Handler(http.StripPrefix("/dev/documentation", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/coordinator").Handler(http.StripPrefix("/dev/coordinator", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/participant").Handler(http.StripPrefix("/dev/participant", http.FileServer(http.Dir("dev"))))
	gMux.PathPrefix("/dev/moyo/mom/emory/participants").Handler(http.StripPrefix("/dev/moyo/mom/emory/participants", http.FileServer(http.Dir("dev"))))

	gMux.PathPrefix("/dev").Handler(http.StripPrefix("/dev", http.FileServer(http.Dir("dev"))))
	//gMux.Handle("/dev/", http.StripPrefix("/dev/", http.FileServer(http.Dir("dev"))))
	gMux.HandleFunc("/dev", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		http.ServeFile(w, r, "dev/index.html")
	})
	//gMux.Handle("/dev/", http.StripPrefix("/dev/", http.FileServer(http.Dir("dev"))))
	//gMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	http.ServeFile(w, r, "dev/index.html")
	//})
	//gMux.PathPrefix("/").Handler(http.FileServer(http.Dir("dev/index.html")))
}
