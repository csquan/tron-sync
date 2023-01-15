package task

import (
	"github.com/chainmonitor/subscribe"
)

type ITask interface {
	Start(s subscribe.Subscriber)
	Stop()
}
