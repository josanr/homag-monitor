package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/josanr/HomagMonitor/board"
	"github.com/josanr/HomagMonitor/part"
	"github.com/josanr/HomagMonitor/runsync"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Dto struct {
	UserId     int    `json:"userId"`
	OrderId    int    `json:"orderId"`
	ToolId     int    `json:"toolId"`
	Gid        int    `json:"gid"`
	PartId     int    `json:"partId"`
	OffcutId   int    `json:"offcutId"`
	ActionType string `json:"actionType"`
	Amount     int    `json:"amount"`
}

/*
	types:
	 	Eingestapelt = взять лист
		Produziert = произведено
		className - даёт тип произведённой детали
				Rest - обрезок
				Teil - деталь => intID это partid - 1, val => это количество произведённых деталей
				Platte - выплнение листа окончено

		)

*/
var connectionStrings = map[string]string{
	"Homag": "server=sysadmin.numina.local\\Holzma;user id=sa;password=holzma;encrypt=disable",
	"H2008": "server=sysadmin.numina.local\\HO2008;user id=sa;password=Homag;encrypt=disable"}

type Config struct {
	UserID int
	ToolID int
}

var db *sql.DB
var monitorConfig Config
var theAPIClient *http.Client

func createConn() (*sql.DB, error) {

	conn, err := sql.Open("mssql", connectionStrings["Homag"])
	if err != nil {
		return nil, err
	}
	err = conn.Ping()
	if err == nil {
		log.Println("connected to homag")
		return conn, nil
	}
	log.Println("Could not connect to Database Homag....")

	conn, err = sql.Open("mssql", connectionStrings["H2008"])
	if err != nil {
		return nil, err
	}
	err = conn.Ping()
	if err == nil {
		log.Println("connected to holzma")
		return conn, nil
	}

	log.Println("Could not connect to Database H2008....")
	return nil, errors.New("could not connect to any Homag Database")
}

func closeFile(f *os.File) {
	log.Println("closing")
	err := f.Close()
	if err != nil {
		log.Printf("error: %v \n", err)
		os.Exit(1)
	}
}
func closeWithErr(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Printf("closing error: %v \n", err)
	}
}
func main() {
	//init logging
	if _, err := os.Stat("counter.log"); err == nil {
		newFilename := "counter-" + time.Now().Format("20060102150405") + ".log"
		err := os.Rename("counter.log", newFilename)
		if err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.OpenFile("counter.log", os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)
	log.Println("Counter Monitor Started")
	defer closeWithErr(f)

	//load config
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatal("Fatal error reading config file")
	}
	servicePort := viper.GetInt("servicePort")

	connectionStrings["Homag"] = viper.GetString("dsns.Homag")
	connectionStrings["H2008"] = viper.GetString("dsns.H2008")

	baseRunPath := viper.GetString("monitorbasepath")
	monitorConfig.ToolID = viper.GetInt("toolId")
	monitorConfig.UserID = viper.GetInt("defaultUserID")
	lastUserID := viper.GetInt("lastUserID")
	if lastUserID != 0 {
		monitorConfig.UserID = lastUserID
	}
	boardsUrl := viper.GetString("apiendpoints.board")
	partsUrl := viper.GetString("apiendpoints.part")

	exitChan := make(chan bool)
	syncer := runsync.New(baseRunPath)
	//cui
	go startControlUI(servicePort)

	//init db connection
	db, err = createConn()
	if err != nil {
		log.Fatal(err)
	}
	//monitors
	boardRespChan, boardErrorChan, err := board.New(db, syncer, exitChan)
	if err != nil {
		log.Fatal("error on board monitor: " + err.Error())
	}

	partsRespChan, partsErrorChan, err := part.New(db, syncer, exitChan)
	if err != nil {
		log.Fatal("error on part monitor: " + err.Error())
	}
	theAPIClient = &http.Client{
		Timeout: time.Second * 10,
	}
	for {

		select {
		case boardItem := <-boardRespChan:

			param := Dto{
				UserId:     monitorConfig.UserID,
				OrderId:    boardItem.Info.OrderID,
				ToolId:     monitorConfig.ToolID,
				Gid:        boardItem.Info.Gid,
				PartId:     0,
				OffcutId:   boardItem.Info.Id,
				ActionType: boardItem.ActionType,
				Amount:     1,
			}
			fmt.Println("board data")
			_ = sendQuery(boardsUrl, param)

		case boardError := <-boardErrorChan:
			log.Println(boardError.ErrorMessage)
		case partItem := <-partsRespChan:
			param := Dto{
				UserId:  monitorConfig.UserID,
				OrderId: partItem.Info.OrderID,
				ToolId:  monitorConfig.ToolID,
				Gid:     partItem.Info.Gid,
				//partId:     partItem.Info.Id,
				//offcutId:   partItem.Info.Id,
				ActionType: "end",
				Amount:     partItem.PartAmount,
			}
			if partItem.IsOffcut {
				param.OffcutId = partItem.Info.Id
				fmt.Println("offcut data")
			} else {
				param.PartId = partItem.Info.Id
				fmt.Println("part data")
			}
			_ = sendQuery(partsUrl, param)

		case partError := <-partsErrorChan:
			log.Println(partError.ErrorMessage)
			//case part := <-monitorer.Parts:
			//	_, _ = json.Marshal(part)
			//case ex := <-monitorer.Errors:
			//	_, _ = json.Marshal(ex)
		}
		// log.Println(string(message))
	}

}

func sendQuery(urlString string, dto Dto) error {

	fmt.Println("URL:>", urlString)
	payload, err := json.Marshal(dto)
	if err != nil {
		log.Println("Marshaling Error" + err.Error())
		return err
	}
	fmt.Println(string(payload))

	req, err := http.NewRequest("POST", urlString, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("request create error: " + err.Error())
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := theAPIClient.Do(req)
	if err != nil {
		log.Println("request send error: " + err.Error())
		return err
	}
	defer closeWithErr(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Println("response status not 200 error: " + resp.Status)
		return err
	}
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1048576))
	if err != nil {
		log.Println("response body read error: " + err.Error())
		return err
	}

	var reqResult apiResult
	err = json.Unmarshal(body, &reqResult)
	if err != nil {
		log.Println("request unmarshal error: " + err.Error())
		return err
	}

	if reqResult.Result != "OK" {
		return errors.New("error set in response: " + reqResult.Result)
	}

	return nil
}

type apiResult struct {
	Result string `json:"result"`
}
