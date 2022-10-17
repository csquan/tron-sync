package subscribe

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/starslabhq/chainmonitor/mtypes"
)

type Subscriber interface {
	SubBlock(chan *mtypes.Block) event.Subscription
	//SubRevert(chan *mtypes.BlockRevert) event.Subscription
}
