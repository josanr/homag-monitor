package part

import (
	"database/sql"
	"errors"
	"github.com/josanr/HomagMonitor/runsync"
	"log"
	"strconv"
	"time"
)

//Parts response for part execution
type Parts struct {
	RecordID     string
	Info         runsync.PartInfo
	PartID       int
	PartAmount   int
	IsOffcut     bool
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

type FatalResponse struct {
	Error        bool
	ErrorMessage string
}

func isRest(actionResult string) (bool, error) {
	switch actionResult {
	case "Rest":
		return true, nil
	case "Teil":
		return false, nil
	default:
		return false, errors.New("action result is not tail or rest")
	}
}

var runList map[string]result

func New(conn *sql.DB, syncer runsync.InfoSync, exit chan bool) (chan Parts, chan FatalResponse, error) {
	runList = make(map[string]result)

	stmt, err := conn.Prepare(`select 
							ID,
							"Lauf",
							"Plan",
							"ClassName",
							"Type",
							"IntID",
							Val 
						from 
							Cadmatic4.dbo.PieceCounter 
						WHERE 
							ClassName IN ('Rest', 'Teil')`)
	if err != nil {
		return nil, nil, errors.New("Prepare Parts query failed:" + err.Error())
	}
	initialPartsSync(stmt)
	resultCnan := make(chan Parts)
	errorChan := make(chan FatalResponse)
	go monitorParts(stmt, syncer, resultCnan, errorChan, exit)

	return resultCnan, errorChan, nil

}

func monitorParts(stmt *sql.Stmt, syncer runsync.InfoSync, partsChan chan Parts, errorChanel chan FatalResponse, exit chan bool) {
	for {
		rows, err := stmt.Query()
		if err != nil {
			errorChanel <- FatalResponse{
				Error:        true,
				ErrorMessage: "Query Parts failed:" + err.Error(),
			}
			log.Println("retrying in 30 sec.")
			time.Sleep(time.Second * 30)
			continue
		}

		for rows.Next() {
			res := result{}

			err = rows.Scan(&res.id, &res.runName, &res.mapID, &res.actionResult, &res.actionType, &res.partid, &res.amount)
			if err != nil {
				errorChanel <- FatalResponse{
					Error:        true,
					ErrorMessage: "Part Scan failed:" + err.Error(),
				}
				continue
			}

			cachedItem, ok := runList[res.getIndex()]
			//not cached
			if ok == false {
				partRest, err := isRest(res.actionResult)
				if err != nil {
					log.Println(err.Error())
				}
				runList[res.getIndex()] = res
				newPart := Parts{
					RecordID:     res.runName,
					Info:         runsync.PartInfo{},
					PartID:       res.partid,
					PartAmount:   res.amount,
					IsOffcut:     partRest,
					Error:        false,
					ErrorMessage: "",
				}

				if newPart.IsOffcut == true {
					partInfo, _ := syncer.GetOffcutByID(res.runName, res.partid)
					newPart.Info = partInfo
				} else {
					partInfo, _ := syncer.GetPartByID(res.runName, res.partid)
					newPart.Info = partInfo
				}

				partsChan <- newPart

			} else if cachedItem.amount != res.amount {
				partRest, err := isRest(res.actionResult)
				if err != nil {
					log.Println(err.Error())
				}
				runList[res.getIndex()] = res

				newPart := Parts{
					RecordID:     res.runName,
					Info:         runsync.PartInfo{},
					PartID:       res.partid,
					PartAmount:   res.amount - cachedItem.amount,
					IsOffcut:     partRest,
					Error:        false,
					ErrorMessage: "",
				}

				if newPart.IsOffcut == true {
					partInfo, _ := syncer.GetOffcutByID(res.runName, res.partid)
					newPart.Info = partInfo
				} else {
					partInfo, _ := syncer.GetPartByID(res.runName, res.partid)
					newPart.Info = partInfo
				}
				partsChan <- newPart
			}

		}

		if err = rows.Err(); err != nil {
			errorChanel <- FatalResponse{
				Error:        true,
				ErrorMessage: "Error on rows:" + err.Error(),
			}
		}
		_ = rows.Close()

		select {
		case _, ok := <-exit:
			if !ok {
				log.Println("closing watch of parts")
				stmt.Close()
				return
			}
		default:
			time.Sleep(time.Second * 1)
		}

	}

}

func initialPartsSync(stmt *sql.Stmt) {
	rows, err := stmt.Query()
	if err != nil {
		log.Println("initial sync error")
		return
	}
	counter := 0
	for rows.Next() {
		res := result{}
		err = rows.Scan(&res.id, &res.runName, &res.mapID, &res.actionResult, &res.actionType, &res.partid, &res.amount)
		if err != nil {
			log.Println("initial sync error Part Scan failed:" + err.Error())
			continue
		}
		runList[res.getIndex()] = res
		counter++
	}
	if err = rows.Err(); err != nil {
		log.Print("initial sync error Error on rows:" + err.Error())
	}

	log.Println("initial parts sync count: ", counter)
}
