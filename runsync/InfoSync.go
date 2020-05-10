package runsync

type InfoSync interface {
	GetBoardByID(runName string, id int) (BoardInfo, error)
	GetPartByID(runName string, id int) (PartInfo, error)
	GetOffcutByID(runName string, id int) (PartInfo, error)
}
