package vault

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
)

var (
	mutex sync.Mutex
)

type amossCreds struct {
	DBAddr   string
	DBUser   string
	DBUserPW string
}

type vaultClient struct {
	client          *api.Client
	dbLeaseID       string
	dbLeaseDuration int
	dbRenewable     bool
}

//for pasing json to get CreationTime and TTL
type token struct {
	RequestID     string `json:"request_id"`
	LeaseID       string `json:"lease_id"`
	Renewable     bool   `json:"renewable"`
	LeaseDuration int    `json:"lease_duration"`
	Data          struct {
		Accessor       string      `json:"accessor"`
		CreationTime   int         `json:"creation_time"`
		CreationTTL    int         `json:"creation_ttl"`
		DisplayName    string      `json:"display_name"`
		ExplicitMaxTTL int         `json:"explicit_max_ttl"`
		ID             string      `json:"id"`
		Meta           interface{} `json:"meta"`
		NumUses        int         `json:"num_uses"`
		Orphan         bool        `json:"orphan"`
		Path           string      `json:"path"`
		Policies       []string    `json:"policies"`
		Renewable      bool        `json:"renewable"`
		TTL            int         `json:"ttl"`
	} `json:"data"`
	WrapInfo interface{} `json:"wrap_info"`
	Warnings interface{} `json:"warnings"`
	Auth     interface{} `json:"auth"`
}

//NewVaultClient akes vault token and address to acquire
//vault client and return it
func NewVaultClient(addr string, token string, httpClient *http.Client) (*vaultClient, error) {
	config := api.Config{Address: addr, HttpClient: httpClient}
	client, err := api.NewClient(&config)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)
	return &vaultClient{client: client}, nil
}

//GetJWTSecret function to get secret from vault
func (v *vaultClient) GetJWTSecret(path string) (string, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return "Error reading secret", err
	}

	if err != nil {
		return "", err
	}
	return secret.Data["JWT_SECRET"].(string), nil
}

func (v *vaultClient) GetCryptoKey(path string) (string, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", err
	}
	return secret.Data["ENCRYPTION_KEY"].(string), nil
}

func (v *vaultClient) GetGarminSecret(path string) (string, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", err
	}
	return secret.Data["GARMIN_SECRET"].(string), nil
}

func (v *vaultClient) GetGarminToken(path string) (string, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", err
	}
	return secret.Data["GARMIN_TOKEN"].(string), nil
}

//GetJWTSecret function to get secret from vault
func (v *vaultClient) GetEmailLambdaAPIKey(path string) (string, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", err
	}
	return secret.Data["EMAIL_LAMBDA_API_KEY"].(string), nil
}

func (v *vaultClient) GetJawboneSecret(path string) (string, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return "", err
	}
	return secret.Data["jb_secret"].(string), nil
}

func (v *vaultClient) GetDbCreds(path string) (amossCreds, error) {
	secret, err := v.client.Logical().Read(path)
	var creds amossCreds
	if err != nil {
		return creds, err
	}

	creds.DBAddr = secret.Data["dbaddr"].(string)
	creds.DBUser = secret.Data["dbuser"].(string)
	creds.DBUserPW = secret.Data["dbuserpw"].(string)
	return creds, nil
}

func (v *vaultClient) AutomateVaultTokenRenewal() error {
	timeToLive, err := v.getTokenTimeToLive()
	log.Println("This is the token TTL(Time To Live) left:", timeToLive)
	if err != nil {
		log.Fatalln(err)
	}
	// Renew token after every start or restart.
	v.vaultTokenRenewal()
	for {
		mutex.Lock()
		v.vaultTokenRenewal()
		mutex.Unlock()
		time.Sleep(time.Second * 604800)
	}
}

func (v *vaultClient) getTokenTimeToLive() (int, error) {
	r := v.client.NewRequest("GET", "/v1/auth/token/lookup-self")
	log.Println("Looking up token TTL.")
	resp, err := v.client.RawRequest(r)
	if err != nil {
		log.Println("Unable to retrieve token TTL.")
	}
	respBuf := new(bytes.Buffer)
	respBuf.ReadFrom(resp.Body)
	tokenRes := token{}
	json.NewDecoder(respBuf).Decode(&tokenRes)
	return tokenRes.Data.TTL, nil
}

func (v *vaultClient) vaultTokenRenewal() error {
	r := v.client.NewRequest("POST", "/v1/auth/token/renew-self")
	_, err := v.client.RawRequest(r)
	log.Println("Renewing token..")
	if err != nil {
		log.Println("Token renewal unsucessful.")
		log.Println(err)
	}
	v.tokenTimeLeftAfterRenewal()
	return nil
}

func (v *vaultClient) tokenTimeLeftAfterRenewal() error {
	timeToLiveAfterRenewal, err := v.getTokenTimeToLive()
	log.Println("This is the token TTL(After Renewal): ", timeToLiveAfterRenewal)
	if err != nil {
		log.Fatalln(err)
	}
	return nil
}
