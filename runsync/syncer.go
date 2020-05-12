package runsync

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type BoardInfo struct {
	OrderID      int
	Id           int
	Gid          int
	Name         string
	Length       int
	Width        int
	Thick        int
	IsFromOffcut bool
}

type PartInfo struct {
	OrderID int
	Id      int
	Gid     int
	Length  int
	Width   int
	Num     int
}

type infoNode struct {
	boards      []BoardInfo
	parts       []PartInfo
	offcuts     []PartInfo
	isLoadError bool
	orderID     int
}

type InfoSyncImpl struct {
	basePath string
	cache    map[string]infoNode
}

func New(basePath string) InfoSyncImpl {
	return InfoSyncImpl{
		basePath: basePath,
		cache:    map[string]infoNode{},
	}
}

func (r InfoSyncImpl) GetBoardByID(runName string, id int) (BoardInfo, error) {
	if _, ok := r.cache[runName]; ok == false {
		r.cache[runName] = r.getInfoFromStore(runName)
	}

	store, _ := r.cache[runName]

	if len(store.boards)-1 < id {
		return BoardInfo{}, errors.New("index out of bonds")
	}

	if store.isLoadError == true {
		return BoardInfo{}, errors.New("Data is corrupt")
	}
	store.boards[id].OrderID = store.orderID
	return store.boards[id], nil
}

func (r InfoSyncImpl) GetPartByID(runName string, id int) (PartInfo, error) {
	if _, ok := r.cache[runName]; ok == false {
		r.cache[runName] = r.getInfoFromStore(runName)
	}

	store, _ := r.cache[runName]

	if len(store.parts)-1 < id {
		return PartInfo{}, errors.New("index out of bonds")
	}

	if store.isLoadError == true {
		return PartInfo{}, errors.New("Data is corrupt")
	}
	store.parts[id].OrderID = store.orderID
	return store.parts[id], nil
}

func (r InfoSyncImpl) GetOffcutByID(runName string, id int) (PartInfo, error) {
	if _, ok := r.cache[runName]; ok == false {
		r.cache[runName] = r.getInfoFromStore(runName)
	}

	store, _ := r.cache[runName]

	if len(store.offcuts)-1 < id {
		return PartInfo{}, errors.New("index out of bonds")
	}

	if store.isLoadError == true {
		return PartInfo{}, errors.New("Data is corrupt")
	}
	store.offcuts[id].OrderID = store.orderID
	return store.offcuts[id], nil
}

func (r *InfoSyncImpl) getInfoFromStore(runName string) infoNode {

	node := infoNode{
		boards:      make([]BoardInfo, 0),
		parts:       make([]PartInfo, 0),
		offcuts:     make([]PartInfo, 0),
		isLoadError: true,
		orderID:     0,
	}
	log.Println(r.basePath + "/" + runName + ".saw")
	file, err := os.Open(r.basePath + "/" + runName + ".saw")
	if err != nil {
		log.Println(err)
		return node
	}
	defer file.Close()

	fileStr, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		return node
	}
	fileArr := strings.Split(string(fileStr), "\n")

	for _, line := range fileArr {
		textLine := strings.Split(strings.Trim(line, "\r"), ",")
		if len(textLine) == 0 {
			continue
		}

		if textLine[0] == "BRD1" {
			idArr := strings.Split(textLine[1], "-")
			if len(idArr) == 1 {
				continue
			}
			v, err := strconv.Atoi(idArr[0])
			if err != nil {
				log.Println(err)
				return node
			}
			node.orderID = v

		}
		if textLine[0] == "BRD2" {
			gid, err := determineGID(textLine[7])
			if err != nil {
				log.Println(err)
				return node
			}
			length, err := strconv.Atoi(textLine[2])
			if err != nil {
				log.Println(err)
				return node
			}
			width, err := strconv.Atoi(textLine[3])
			if err != nil {
				log.Println(err)
				return node
			}
			thick, err := strconv.Atoi(textLine[6])
			if err != nil {
				log.Println(err)
				return node
			}
			id := 0
			isFromOffcut := false
			if len(textLine) >= 15 {
				id, err = strconv.Atoi(textLine[14])
				if err == nil {
					isFromOffcut = true
				}
			}
			node.boards = append(node.boards, BoardInfo{
				Gid:          gid,
				Id:           id,
				Name:         textLine[1],
				Length:       length,
				Width:        width,
				Thick:        thick,
				IsFromOffcut: isFromOffcut,
			})
		}
		if textLine[0] == "PNL2" {
			gid, err := determineGID(textLine[2])
			if err != nil {
				log.Println(err)
				return node
			}
			length, err := strconv.Atoi(textLine[3])
			if err != nil {
				log.Println(err)
				return node
			}
			width, err := strconv.Atoi(textLine[4])
			if err != nil {
				log.Println(err)
				return node
			}
			num, err := strconv.Atoi(textLine[5])
			if err != nil {
				log.Println(err)
				return node
			}
			id, err := strconv.Atoi(textLine[1])
			if err != nil {
				log.Println(err)
				return node
			}
			node.parts = append(node.parts, PartInfo{Gid: gid, Id: id, Length: length, Width: width, Num: num})
		}
		if textLine[0] == "XBRD2" {
			gid, err := determineGID(textLine[7])
			if err != nil {
				log.Println(err)
				return node
			}
			length, err := strconv.ParseInt(strings.Split(textLine[2], ".")[0], 10, 64)
			if err != nil {
				log.Println(err)
				return node
			}
			width, err := strconv.ParseInt(strings.Split(textLine[3], ".")[0], 10, 64)
			if err != nil {
				log.Println(err)
				return node
			}
			num := 1
			id, err := strconv.Atoi(textLine[1])
			if err != nil {
				id = 0
			}
			node.offcuts = append(node.offcuts, PartInfo{Gid: gid, Id: id, Length: int(length), Width: int(width), Num: num})
		}
	}
	node.isLoadError = false
	return node
}

func determineGID(textLine string) (int, error) {
	codeArr := strings.Split(textLine, "_")
	var gid int
	var err error
	if len(codeArr) > 1 {
		//bazis-soft type id
		codeHolder := codeArr[0]
		if codeHolder == "OBR" {
			codeHolder = codeArr[1]
		}
		gid, err = strconv.Atoi(codeHolder)
		if err != nil {
			return gid, err
		}
	} else {
		gid, err = strconv.Atoi(codeArr[0])
		if err != nil {
			return gid, err
		}
	}

	if gid == 0 {
		return gid, errors.New("Could not determine GID")
	}

	return gid, nil

}
