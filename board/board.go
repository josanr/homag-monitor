package board

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/josanr/HomagMonitor/runsync"
)

type FatalResponse struct {
	Error        bool
	ErrorMessage string
}

type result struct {
	id           int    //ID
	partid       int    //IntID
	amount       int    //Val
	actionType   string //Type
	actionResult string //ClassName
	mapID        string //Plan
	runName      string //Lauf
}

func (r result) getIndex() string {

	return strconv.FormatInt(int64(r.id), 10) + ":" + r.runName + ":" + r.actionResult + ":" + strconv.FormatInt(int64(r.partid), 10)
}

//Board Board response from tool db
type Board struct {
	RecordID     string
	Info         runsync.BoardInfo
	MapID        int
	ActionType   string
	Error        bool
	ErrorMessage string
}

func setActionType(actionResult string) (string, error) {
	switch actionResult {
	case "Eingestapelt":
		return "start", nil
	case "Produziert":
		return "end", nil
	default:
		return "", errors.New("action type is not palete or produced")
	}
}

func New(conn *sql.DB, syncer runsync.InfoSync, exit chan bool) (chan Board, chan FatalResponse, error) {
	var lastID int
	lastIDArr := conn.QueryRow(`
		SELECT 
			max(id) 
		from 
			Cadmatic4.dbo.PieceCounter  
		WHERE 
			ClassName NOT IN ('Rest', 'Teil');
			`)
	err := lastIDArr.Scan(&lastID)
	if err != nil {
		log.Println("error getting last board id: " + err.Error())
		lastID = 0
	}

	stmt, err := conn.Prepare(`select
							ID,
							"Lauf",
							"Plan",
							"ClassName",
							"Type",
							"IntID",
							Val
						from Cadmatic4.dbo.PieceCounter
						WHERE id > ?
						AND ClassName NOT IN ('Rest', 'Teil')`)
	if err != nil {
		return nil, nil, errors.New("Prepare Boards query failed:" + err.Error())
	}

	resultCnan := make(chan Board)
	errorChan := make(chan FatalResponse)

	go monitorBoards(stmt, lastID, syncer, resultCnan, errorChan, exit)
	return resultCnan, errorChan, nil
}

func monitorBoards(stmt *sql.Stmt, lastID int, syncer runsync.InfoSync, boardCan chan Board, errorChanel chan FatalResponse, exit chan bool) {

	for {
		rows, err := stmt.Query(lastID)
		switch {
		case err == sql.ErrNoRows:
			_ = rows.Close()
			log.Println("no rows")
			time.Sleep(time.Second * 1)
			continue
		case err != nil:
			log.Println(err)
			errorChanel <- FatalResponse{
				Error:        true,
				ErrorMessage: "Query Boards error:" + err.Error(),
			}
			continue
		}

		for rows.Next() {

			row := result{}

			err = rows.Scan(&row.id, &row.runName, &row.mapID, &row.actionResult, &row.actionType, &row.partid, &row.amount)
			if err != nil {
				log.Println(err)
				continue
			}

			lastID = row.id
			boardID, _ := strconv.Atoi(row.mapID)
			actType, err := setActionType(row.actionType)
			if err != nil {
				log.Println("Row scan: acttype: ", err)
				continue
			}

			boardInfo, err := syncer.GetBoardByID(row.runName, row.partid)
			if err != nil {
				log.Println("get board info: acttype: ", err)
				continue
			}
			boardCan <- Board{
				RecordID:     row.runName,
				Info:         boardInfo,
				MapID:        boardID,
				ActionType:   actType,
				Error:        false,
				ErrorMessage: "",
			}
		}
		if err = rows.Err(); err != nil {
			errorChanel <- FatalResponse{
				Error:        true,
				ErrorMessage: "Error on board rows:" + err.Error(),
			}
		}
		_ = rows.Close()
		select {
		case _, ok := <-exit:
			if !ok {
				_ = stmt.Close()
				log.Println("closing watch of boards")
				return
			}
		default:
			time.Sleep(time.Second * 1)
		}

	}

}
