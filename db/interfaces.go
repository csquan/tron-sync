package db

import (
	"github.com/chainmonitor/mtypes"
	"github.com/chainmonitor/output/mysqldb"
	"xorm.io/xorm"
)

type IReader interface {
	GetCurrentBlock() (*mysqldb.Block, error)
	GetCurrentMainBlockNum() (uint64, error)
	GetBlockByNum(num uint64, filter *mysqldb.BlockFilter) (*mtypes.Block, error)
	GetBlocksByRange(start, end uint64, filter *mysqldb.BlockFilter) ([]*mtypes.Block, error)
	GetRevertBlocks(start, end uint64, filter *mysqldb.BlockFilter) ([]*mtypes.Block, error)
	GetTokenPairByAddr(addr string) (*mysqldb.TokenPair, error)
	GetErc1155Balance(address string, contractAddr string, tokenID string) (*mysqldb.BalanceErc1155, error)
	GetBlockByNumAndState(num uint64, state int) (*mysqldb.Block, error)
	GetErc20info(addr string) (*mysqldb.Erc20Info, error)

	GetMonitorUID(to string) (string, error)
	GetOpenMonitorTx(chain string) ([]*mysqldb.TxMonitor, error)
}

type IWriter interface {
	GetTaskByName(name string) (*mysqldb.AsyncTask, error)
	UpdateAsyncTaskNumByName(p xorm.Interface, name string, num uint64) error

	SaveBalances(balances []*mysqldb.Balance) error
	GetBalance(address string) (balance *mysqldb.Balance, err error)

	SaveRevertBalances(s xorm.Interface, balances []*mysqldb.Balance) error

	SaveBlock(xorm.Interface, *mysqldb.Block) (int64, error)
	SaveBlocks(xorm.Interface, []*mysqldb.Block) error
	SaveTxs(xorm.Interface, []*mysqldb.TxDB) error
	SaveInternalTxs(xorm.Interface, []*mysqldb.TxInternal) error
	SaveLogs(xorm.Interface, []*mysqldb.TxLog) error
	SaveContracts(xorm.Interface, []*mysqldb.Contract) error

	//revert block 设置blockNum = bNum 为state
	UpdateBlockSate(bNum uint64, state int) error

	//erc20
	SaveTxErc20s(xorm.Interface, []*mysqldb.TxErc20) error
	SaveErc20Infos(xorm.Interface, []*mysqldb.Erc20Info) error
	SaveErc20Balances(xorm.Interface, []*mysqldb.BalanceErc20) error
	GetBalanceErc20(address string, contractAddr string) (balanceErc20 *mysqldb.BalanceErc20, err error)
	//erc721
	SaveTxErc721s(xorm.Interface, []*mysqldb.TxErc721) error
	SaveErc721Infos(xorm.Interface, []*mysqldb.Erc721Info) error
	SaveErc721Tokens(xorm.Interface, []*mysqldb.Erc721Token) error
	SaveTxErc1155s(s xorm.Interface, txErc1155s []*mysqldb.TxErc1155) error
	SaveErc1155Balances(xorm.Interface, []*mysqldb.BalanceErc1155) error
	//TokenPair
	SaveTokenPairs(xorm.Interface, []*mysqldb.TokenPair) error
	UpdateTokenPairsReserve(s xorm.Interface, pairs []*mysqldb.TokenPair) error

	DeleteErc20InfoByAddr(session xorm.Interface, addr string) error
	UpdateMonitorHash(done int, gasLimit uint64, gasPrice string, gasUsed uint64, index int, status int, hash string, chain string, orderId string) error

	GetSession() *xorm.Session
	GetEngine() *xorm.Engine
}

type IDB interface {
	IReader
	IWriter
}
