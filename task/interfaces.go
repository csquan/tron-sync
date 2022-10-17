package task

import (
	"github.com/starslabhq/chainmonitor/subscribe"
)

type ITask interface {
	Start(s subscribe.Subscriber)
	Stop()
}
