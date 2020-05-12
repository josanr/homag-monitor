package stats

import (
	"database/sql"
	"fmt"
	"log"
)

type FatalResponse struct {
	Error        bool
	ErrorMessage string
}

type Stat struct {
}

func New(exit chan bool) (chan Stat, chan FatalResponse, error) {
	db, err := createConnection()
	if err != nil {
		log.Println("Connection error: " + err.Error())
		return nil, nil, err
	}

	if err = db.Ping(); err != nil {
		log.Println("Ping error: " + err.Error())
		return nil, nil, err
	}
	resultCnan := make(chan Stat)
	errorChan := make(chan FatalResponse)
	lastID := getLastID()
	monitor(db, lastID)
	return resultCnan, errorChan, nil
}

func getLastID() int64 {
	return 16303840
}

func monitor(db *sql.DB, lastID int64) {

	rows, err := db.Query(`
SELECT T910_MACHINE_DATA.T910_MACHINE_DATA_ID,
       T910_MACHINE_DATA.T910_MACHINE_STATE_ID,
       T910_MACHINE_DATA.T910_STATE_DATE,
       T906_MACHINE_STATE.T906_DATA_TYPE_ID,
	   T910_MACHINE_DATA.T910_VALUE,
	   T910_MACHINE_DATA.T910_DURATION,
		B906_MACHINE_STATE_DSC.B906_DESCRIPTION
FROM ((T910_MACHINE_DATA
         LEFT JOIN T906_MACHINE_STATE ON T910_MACHINE_DATA.T910_MACHINE_STATE_ID =  T906_MACHINE_STATE.T906_MACHINE_STATE_ID)
         LEFT JOIN B906_MACHINE_STATE_DSC ON T910_MACHINE_DATA.T910_MACHINE_STATE_ID = B906_MACHINE_STATE_DSC.B906_MACHINE_STATE_ID )
WHERE B906_LANGUAGE_ID = ? 
AND	T910_MACHINE_DATA_ID > ? 
ORDER BY T910_MACHINE_DATA.T910_STATE_DATE, T910_MACHINE_DATA_ID;
	`, 1, lastID)
	if err != nil {
		log.Println("select", err)
		return
	}

	for rows.Next() {
		var id int64
		var stateID int
		var valType sql.NullInt64
		var stateDate string
		var value sql.NullInt64
		var description string
		var duration sql.NullString
		err = rows.Scan(&id, &stateID, &stateDate, &valType, &value, &duration, &description)
		if err != nil {
			fmt.Println("scan", err)
			return
		}
		fmt.Println(id, stateDate, stateID, valType, value, duration, description)
	}

	rows.Close()
	db.Close()
}
