package participant

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/cliffordlab/amoss_services/capacity"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/amoss_services/log_writer"
	"github.com/cliffordlab/amoss_services/mathb"
	"github.com/cliffordlab/amoss_services/support"
)

const (
	duplicateUserErr = `{"error":"cannot create a duplicate participant"}`
	noSuchUserErr    = `{"error":"invalid participant id or password"}`
	getEmail         = `SELECT encrypted_phone FROM participants WHERE participant_id=$1`
	getIV            = `SELECT phone_iv FROM participants WHERE participant_id=$1`

	insertAdmin = `INSERT INTO participants (participant_id, password_hash, password_salt, capacity_id)
    VALUES ($1, $2, $3, (SELECT capacity_id FROM participant_capacity WHERE capacity_id=$4))`

	insertParticipant = `INSERT INTO participants (participant_id, password_hash, password_salt, capacity_id, study_id)
	VALUES ($1, $2, $3, (SELECT capacity_id FROM participant_capacity WHERE capacity_id=$4),
	(SELECT study_id FROM studies WHERE study_id=$5))`

	insertMoyoParticipant = `INSERT INTO participants (participant_id, password_hash, password_salt, capacity_id, study_id, email_hash, encryption_iv, encrypted_email, encrypted_phone, phone_iv, is_consented)
	VALUES ($1, $2, $3, (SELECT capacity_id FROM participant_capacity WHERE capacity_id=$4),
	(SELECT study_id FROM studies WHERE study_id=$5),
	$6, $7, $8, $9, $10, $11)`

	selectMoyoParticipant = `SELECT email_hash FROM participants where (email_hash) = ($1)`

	alterParticipant  = `UPDATE participants SET (password_hash, password_salt) = ($1, $2) WHERE participant_id = $3;`
	insertAccessToken = `UPDATE participants SET access_token = $1 WHERE participant_id = $2;`
)

//Participant data type of users interacting with application
type Participant struct {
	EmailEncoded []byte
	Email        string
	Phone        []byte
	ID           int64
	Capacity     string
	Salt         string
	PasswordHash string
	Study        string
	IV           []byte
	PhoneIV      []byte
	EmailHash    string
	Password     string
	HasConsented bool
}

//Salt gets the salt for the participant
func Salt(currentParticipant *Participant, ptID int64, w http.ResponseWriter) {
	log.Println("Querying db for participant's salt... ")
	w = log_writer.LogWriter{ResponseWriter: w}
	query := "SELECT password_salt, capacity_id, study_id FROM participants WHERE participant_id = $1"
	rows, err := database.ADB.Db.Query(query, ptID)
	if err != nil {
		log.Println("failed to execute get salt query statement")
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(noSuchUserErr))
		return
	}
	var salt string
	var participantCapacity string
	var study string
	for rows.Next() {
		err = rows.Scan(&salt, &participantCapacity, &study)
		if err != nil {
			log.Println("failed to scan row for salt")
			log.Println(err)
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte(noSuchUserErr))
			return
		}
	}
	rows.Close()
	currentParticipant.Salt = salt
	currentParticipant.Capacity = participantCapacity
	currentParticipant.Study = study
	if salt == "" {
		log.Println("Salt is nil in database for participant")
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(noSuchUserErr))
		return
	}

}

//Password get current participants password
func Password(ptID int64, currentParticipant *Participant) {
	log.Println("Querying for participants password...")
	query := "SELECT password_hash FROM participants WHERE participant_id = $1"
	rows, err := database.ADB.Db.Query(query, ptID)
	if err != nil {
		log.Println("failed to execute get password query")
		log.Println(err)
		return
	}
	var passwordHash string
	for rows.Next() {
		err = rows.Scan(&passwordHash)
		if err != nil {
			log.Println("failed to scan row for password hash")
			log.Println(err)
		}
	}
	rows.Close()
	currentParticipant.PasswordHash = passwordHash
}

//LoginParticipant check if participant creds match what
//is in the database
func LoginParticipant(currentParticipant *Participant, w http.ResponseWriter) {
	w = log_writer.LogWriter{ResponseWriter: w}
	token := capacity.CreateAccessToken(currentParticipant.Capacity, currentParticipant.Study, currentParticipant.ID)
	saveTokenToDb(currentParticipant, token, w)
	tokenResponse := fmt.Sprintf(`{"token":"%s", "capacity":"%s", "participantID":"%v", "study":"%s", "isConsented":"%v"}`, token, currentParticipant.Capacity, currentParticipant.ID, currentParticipant.Study, currentParticipant.HasConsented)
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.Write([]byte(tokenResponse))
}

func saveTokenToDb(currentParticipant *Participant, token string, w http.ResponseWriter) {
	w = log_writer.LogWriter{ResponseWriter: w}
	log.Println("Ready to insert token")

	stmt, err := database.ADB.Db.Prepare(insertAccessToken)
	if err != nil {
		log.Println("failed to prepare insert access token statement")
		log.Fatalln(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(noSuchUserErr))
	}
	defer stmt.Close()

	ptidString := strconv.Itoa(int(currentParticipant.ID))

	rows, err := stmt.Query(token, ptidString)
	if err != nil {
		log.Println("failed to execute query statement. participant id error.")
		log.Println(err)
		http.Error(w, "Invalid email address.", http.StatusUnauthorized)
		return
	}
	rows.Close()
	log.Println("Access token inserted to db successfully")

}

//CreateAdmin insert participant into database
func CreateAdmin(admin Participant, w http.ResponseWriter) {
	w = log_writer.LogWriter{ResponseWriter: w}
	stmt, err := database.ADB.Db.Prepare(insertAdmin)
	if err != nil {
		log.Println("failed to prepare create admin statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(admin.ID, admin.PasswordHash, admin.Salt, admin.Capacity)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(duplicateUserErr))
		return
	}
	rows.Close()
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.Write([]byte("{\"success\":\"admin participant created\"}"))
}

//CreateNonAdmin insert participant with coordinator privileges into database
func CreateNonAdmin(pt Participant, w http.ResponseWriter) {
	w = log_writer.LogWriter{ResponseWriter: w}
	stmt, err := database.ADB.Db.Prepare(insertParticipant)
	if err != nil {
		log.Println("failed to prepare create participant statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(pt.ID, pt.PasswordHash, pt.Salt, pt.Capacity, pt.Study)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(duplicateUserErr))
		return
	}
	rows.Close()
	response := fmt.Sprintf("{\"success\":\"%s participant created\"}", pt.Capacity)
	log.Println("participant created successfully")
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(response))
}

var counter int

//CreateNonAdmin insert participant with coordinator privileges into database
func CreateMoyoNonAdmin(pt Participant, w http.ResponseWriter) (int int64, email string, err error) {
	log.Printf("Creating Moyo non admin....")
	//Find unused participant ID in database
	//Generate random 10 digit numerical number
	//rand.NewSource(time.Now().UnixNano())
	pid := mathb.RandInt(1000000000, 9999999999)
	log.Println(pid)
	//Attempt to insert into DB
	pt.ID = pid

	log.Println("-----------------")
	log.Println("this is the encrypted email before db insert: ")
	fmt.Println(pt.EmailEncoded)
	fmt.Println(string(pt.EmailEncoded))
	fmt.Printf("%x", pt.EmailEncoded)
	log.Println("-----------------")

	stmt, err := database.ADB.Db.Prepare(insertMoyoParticipant)
	if err != nil {
		log.Println("failed to prepare create participant statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(pt.ID, pt.PasswordHash, pt.Salt, pt.Capacity, pt.Study, pt.EmailHash, pt.IV, pt.EmailEncoded, pt.Phone, pt.PhoneIV, true)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Println(err)
		return 0, "failed", err
	}

	rows.Close()
	log.Println("participant created successfully. Participant ID: " + string(pt.ID))
	// send email with participant ID and password if everything is successful
	emailStatus := support.EmailMoyoParticipant(pt.Email, pt.ID, pt.Password, w)
	//decrypt(w, pt)
	return pid, emailStatus, nil
}

func IsParticipantInDB(pt Participant) (isRegistered bool) {
	log.Printf("Check if email is already in system...")

	stmt, err := database.ADB.Db.Prepare(selectMoyoParticipant)
	if err != nil {
		log.Println("failed to prepare create participant statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(pt.EmailHash)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Fatalln(err)
	}
	var emailHash string

	for rows.Next() {
		err = rows.Scan(&emailHash)
		if err != nil {
			log.Println("failed to scan row for email hash")
			log.Println(err)
			return
		}
	}
	rows.Close()

	return pt.EmailHash == emailHash
}

//AlterSaltAndPasswordHash to protect against hacking
func AlterSaltAndPasswordHash(currentParticipant *Participant) {
	log.Println("Ready to alter salt and pwHash")
	stmt, err := database.ADB.Db.Prepare(alterParticipant)
	if err != nil {
		log.Println("failed to prepare alter participant statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(currentParticipant.PasswordHash, currentParticipant.Salt, currentParticipant.ID)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Println(err)
		return
	}
	rows.Close()
}
