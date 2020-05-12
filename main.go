package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/josanr/HomagMonitor/board"
	"github.com/josanr/HomagMonitor/part"
	"github.com/josanr/HomagMonitor/runsync"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Dto struct {
	userId     int
	orderId    int
	toolId     int
	gid        int
	partId     int
	offcutId   int
	actionType string
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
	defer closeFile(f)

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

	baseRunPath := viper.GetString("runBasePath")
	monitorConfig.ToolID = viper.GetInt("toolId")
	monitorConfig.UserID = viper.GetInt("defaultUserID")
	lastUserID := viper.GetInt("lastUserID")
	if lastUserID != 0 {
		monitorConfig.UserID = lastUserID
	}
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

	for {

		select {
		case boardItem := <-boardRespChan:
			log.Println(boardItem)
		case boardError := <-boardErrorChan:
			log.Println(boardError.ErrorMessage)
		case partItem := <-partsRespChan:
			log.Println(partItem)
		case partError := <-partsErrorChan:
			log.Println(partError.ErrorMessage)
			//case part := <-monitorer.Parts:
			//	_, _ = json.Marshal(part)
			//case ex := <-monitorer.Errors:
			//	_, _ = json.Marshal(ex)
		}
		// log.Println(string(message))
	}

	//var connActive = make(chan bool)
	//var monitorer = monitor.New(db, &monitorConfig, connActive)
	//// var message []byte
	//for {
	//
	//	select {
	//	case board := <-monitorer.Boards:
	//
	//		err = markBoard(board)
	//		log.Println("board sent")
	//	case part := <-monitorer.Parts:
	//		_, _ = json.Marshal(part)
	//	case ex := <-monitorer.Errors:
	//		_, _ = json.Marshal(ex)
	//	}
	//	// log.Println(string(message))
	//}

}

func startControlUI(servicePort int) {
	http.HandleFunc("/setUid/", setUIDHandler)
	log.Println("CUI starting")
	err := http.ListenAndServe(":"+strconv.Itoa(servicePort), nil)
	if err != nil {
		log.Println("could not start control interface")
		log.Println(err)
		return
	}

}

//func markBoard(board monitor.Board) error {
//	urlString := viper.GetString("apiendpoints.board")
//
//	form := url.Values{}
//	form.Add("gid", strconv.Itoa(board.Info.Gid))
//	form.Add("length", strconv.Itoa(board.Info.Length))
//	form.Add("width", strconv.Itoa(board.Info.Width))
//	form.Add("thick", strconv.Itoa(board.Info.Thick))
//
//	form.Add("orderId", strconv.Itoa(board.Info.OrderID))
//	form.Add("toolId", strconv.Itoa(board.Info.ToolID))
//	form.Add("userId", strconv.Itoa(board.Info.UserID))
//
//	form.Add("boardId", strconv.Itoa(board.Info.Id))
//	form.Add("isFromOffcut", strconv.FormatBool(board.Info.IsFromOffcut))
//
//	form.Add("actionType", board.ActionType)
//
//	req, err := http.NewRequest(http.MethodPost, urlString, strings.NewReader(form.Encode()))
//	if err != nil {
//		return err
//	}
//	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
//
//	resp, err := theAPIClient.Do(req)
//	if err != nil {
//		return err
//	}
//	defer resp.Body.Close()
//	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1048576))
//	if err != nil {
//		return err
//	}
//
//	reqResult := make(map[string]string)
//	json.Unmarshal(body, &reqResult)
//
//	if _, ok := reqResult["result"]; ok == false {
//		return errors.New("wrong response format")
//	}
//
//	if v, _ := reqResult["result"]; v != "OK" {
//		return errors.New("error set in response: " + v)
//	}
//
//	return nil
//}

type okResponse struct {
	Message string `json:"message"`
}

func setUIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path != "/setUid/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "wrong request method", http.StatusBadRequest)
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "request form data corrupt", http.StatusBadRequest)
		return
	}
	uid := r.FormValue("uid")
	monitorConfig.UserID, _ = strconv.Atoi(uid)
	viper.Set("lastUserID", monitorConfig.UserID)
	_ = viper.WriteConfig()
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(okResponse{"action done, id set: " + strconv.Itoa(monitorConfig.UserID)})
	return
}
