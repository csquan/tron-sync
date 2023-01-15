package output

import (
	"github.com/chainmonitor/mtypes"
	"github.com/chainmonitor/output/mysqldb"
)

type Output interface {
	Output(<-chan *mtypes.Block) error
	GetCurrentBlock() *mysqldb.Block
	GetBlockByNum(num uint64, state int) *mysqldb.Block
}
