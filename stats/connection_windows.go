package stats

import (
	"database/sql"
	"github.com/go-ole/go-ole"
	_ "github.com/mattn/go-adodb"
	"log"
)

func createConnection() (*sql.DB, error) {
	ole.CoInitialize(0)
	defer ole.CoUninitialize()
	provider := "Microsoft.Jet.OLEDB.4.0"
	filePath := "MDE_DB.mdb"

	db, err := sql.Open("adodb", "Provider="+provider+";Data Source="+filePath+";")
	if err != nil {
		log.Println("could not use provider: " + provider)
		provider = "Microsoft.ACE.OLEDB.12.0"
		db, err = sql.Open("adodb", "Provider="+provider+";Data Source="+filePath+";")
		if err != nil {
			log.Println("open error", err)
			return nil, err
		}

	}
	err = db.Ping()
	if err != nil {
		log.Println("ping error", err)
		return nil, err
	}
	return db, nil
}
