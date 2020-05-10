package runsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var localPath = "/home/ruslan/Projects/Golang/HomagMonitor/test_assets"

func Test_BazisSoft_board(t *testing.T) {
	syncer := New(localPath)
	board, err := syncer.GetBoardByID("11_1412R 54377 ДСП H3156 ST12 18мм Дуб Корбридж се", 0)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 54377, board.Gid)
}

func Test_CUtrite_board(t *testing.T) {
	syncer := New(localPath)

	board, err := syncer.GetBoardByID("28_11_2019OR Часть 3-Holzma 300", 1)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 65284, board.Gid)
	assert.Equal(t, 2760, board.Length)
	assert.Equal(t, 1355, board.Width)
	assert.Equal(t, 18, board.Thick)

	board, err = syncer.GetBoardByID("28_11_2019OR Часть 3-Holzma 300", 5)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 84163, board.Gid)
	assert.Equal(t, 2720, board.Length)
	assert.Equal(t, 650, board.Width)
	assert.Equal(t, 18, board.Thick)
}

func Test_CUtrite_part(t *testing.T) {
	syncer := New(localPath)

	part, err := syncer.GetPartByID("28_11_2019OR Часть 3-Holzma 300", 1)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 65284, part.Gid)
	assert.Equal(t, 1798, part.Length)
	assert.Equal(t, 560, part.Width)

	part, err = syncer.GetPartByID("28_11_2019OR Часть 3-Holzma 300", 50)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 84163, part.Gid)
	assert.Equal(t, 764, part.Length)
	assert.Equal(t, 60, part.Width)

}

func Test_CUtrite_offcut(t *testing.T) {
	syncer := New(localPath)

	offcut, err := syncer.GetOffcutByID("28_11_2019OR Часть 3-Holzma 300", 29)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 65284, offcut.Gid)
	assert.Equal(t, 942, offcut.Length)
	assert.Equal(t, 1355, offcut.Width)

}

func Test_index_out_of_bond(t *testing.T) {
	syncer := New(localPath)

	_, err := syncer.GetOffcutByID("28_11_2019OR Часть 3-Holzma 300", 500)
	if err == nil {
		t.Error("shoud be error out of bounds")
	}

}
func Test_plan_id_setting(t *testing.T) {
	syncer := New(localPath)

	item, err := syncer.GetBoardByID("28_11_2019OR Часть 3-Holzma 300", 1)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 1315672, item.Id)
	assert.Equal(t, true, item.IsFromOffcut)

	item, err = syncer.GetBoardByID("11_1412R 8327 ДСП W908 ST2_8мм Белый кожа EG YX", 0)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 0, item.Id)
	assert.Equal(t, false, item.IsFromOffcut)
}
