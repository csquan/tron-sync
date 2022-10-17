package output

import (
	"github.com/starslabhq/chainmonitor/mtypes"
	"github.com/starslabhq/chainmonitor/output/mysqldb"
)

type Output interface {
	Output(<-chan *mtypes.Block) error
	GetCurrentBlock() *mysqldb.Block
	GetBlockByNum(num uint64, state int) *mysqldb.Block
}
