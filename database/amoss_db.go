package database

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "github.com/lib/pq"
)

//AmossDB global type to wrap database
type AmossDB struct {
	Db          *sql.DB
	Environment string
}

//ADB database instance for application
var ADB AmossDB

// InitDb used in main function to initialize the database
func InitDb(username, password, dbAddr string, dbName string) {
	sslmode := "require"
	if dbAddr == "localhost" {
		sslmode = "disable"
	}
	dsn := "user=" + username + " password=" + password + " host=" + dbAddr + " dbname=" + dbName + " sslmode=" + sslmode
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalln(err)
	}
	ADB.Db = db
	log.Println("Database instance is ready")
}

// JawboneUser contains data to make jawbone queries and send files to correct place on s3
type JawboneUser struct {
	ParticipantID int
	Study         string
	WearableID    string
	AccessToken   string
	JawboneDate   string
}

// FindJawboneUsers get user with there access token for jawbone requests
func FindJawboneUsers() []JawboneUser {
	// Query the DB
	rows, err := ADB.Db.Query(`SELECT participant_id, study_id, wearable_id FROM participants;`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var ptid int
	var study sql.NullString
	var wearableID sql.NullString

	var jbUsers []JawboneUser
	for rows.Next() {
		err := rows.Scan(&ptid, &study, &wearableID)
		if err != nil {
			log.Fatal(err)
		}

		jbUsers = append(jbUsers, JawboneUser{ParticipantID: ptid, Study: study.String, WearableID: wearableID.String})
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	for index, element := range jbUsers {
		if element.WearableID == "" {
			continue
		}
		log.Println("Below is the wearable Id:")
		log.Println(element.WearableID)

		stmt, err := ADB.Db.Prepare(`SELECT access_token, jawbone_date FROM wearables WHERE wearable_id = $1;`)
		if err != nil {
			log.Println("failed to prepare create access token query statement")
			log.Fatalln(err)
		}
		defer stmt.Close()

		var accessToken string
		var jawboneDate string

		stmtRows, err := stmt.Query(element.WearableID)
		if err != nil {
			log.Fatal(err)
		}
		for stmtRows.Next() {
			err := stmtRows.Scan(&accessToken, &jawboneDate)
			if err != nil {
				log.Fatal(err)
			}
			jbUsers[index].AccessToken = accessToken
			jbUsers[index].JawboneDate = jawboneDate
		}
		defer stmtRows.Close()
	}
	return jbUsers
}

func PgToJSON(rows *sql.Rows) []byte {
	log.Println("Transforming data to JSON...")
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Println("Printing Column types error: ")
		log.Println(err)
	}

	count := len(columnTypes)
	var finalRows []interface{}

	for rows.Next() {

		scanArgs := make([]interface{}, count)

		for i, v := range columnTypes {
			switch v.DatabaseTypeName() {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
				break
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
				break
			case "BIGINT":
				scanArgs[i] = new(sql.NullInt64)
				break
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		err := rows.Scan(scanArgs...)

		if err != nil {
			log.Println(err)
		}

		masterData := map[string]interface{}{}

		for i, v := range columnTypes {

			if z, ok := (scanArgs[i]).(*sql.NullBool); ok {
				masterData[v.Name()] = z.Bool
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullString); ok {
				masterData[v.Name()] = z.String
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullInt64); ok {
				masterData[v.Name()] = z.Int64
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullFloat64); ok {
				masterData[v.Name()] = z.Float64
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullInt32); ok {
				masterData[v.Name()] = z.Int32
				continue
			}

			masterData[v.Name()] = scanArgs[i]
		}

		finalRows = append(finalRows, masterData)
	}

	result, err := json.Marshal(finalRows)
	rows.Close()
	return result
}
