package participant

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/amoss_services/mathb"
	"log"
	"net/http"
)

const (
	doesIDExist = `SELECT EXISTS(SELECT 1 FROM participants WHERE(participant_ID) = ($1))`
)

type IDGenerationHandler struct {
	Name string
	Svc  *s3.S3
}

func (idGH IDGenerationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Generating unique participant ID... ")
	pid, doesIDExist, _ := getUniqueID()
	for {
		if string(doesIDExist) == "false" {
			break
		} else {
			pid, doesIDExist, _ = getUniqueID()
		}
	}
	response := fmt.Sprintf(`{"participantID":%d}`, pid)
	log.Println("Unique ID found!")
	log.Println(pid)
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.Write([]byte(response))
	return
}

func getUniqueID() (int64, []byte, bool) {
	pid := mathb.RandInt(100000, 999999)

	stmt, err := database.ADB.Db.Prepare(doesIDExist)
	if err != nil {
		log.Println("failed to prepare create participant statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(pid)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Println(err)
		return 0, nil, true
	}

	var doesIDExist []byte
	for rows.Next() {
		err = rows.Scan(&doesIDExist)
		if err != nil {
			log.Println("failed to scan row for ID")
			log.Println(err)
		}
	}

	rows.Close()
	return pid, doesIDExist, false
}
