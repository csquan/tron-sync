package subscribe

import (
	"github.com/chainmonitor/mtypes"
	"github.com/ethereum/go-ethereum/event"
)

type Subscriber interface {
	SubBlock(chan *mtypes.Block) event.Subscription
	//SubRevert(chan *mtypes.BlockRevert) event.Subscription
}
