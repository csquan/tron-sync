package utils

import (
	"reflect"
	"runtime"
	"time"

	"github.com/starslabhq/chainmonitor/log"

	"github.com/sirupsen/logrus"
)

func HandleErrorWithRetry(handle func() error, retryTimes int, interval time.Duration) {
	err := HandleErrorWithRetryMaxTime(handle, retryTimes, interval)
	if err != nil {
		log.Fatal(err)
	}
}

type RFunc func(func() error)

func HandleErrorWithRetryV2(retryTimes int, interval time.Duration) RFunc {
	return func(handle func() error) {
		err := HandleErrorWithRetryMaxTime(handle, retryTimes, interval)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func HandleErrorWithRetryMaxTime(handle func() error, retryTimes int, interval time.Duration) error {
	inc := 0
	for {
		err := handle()
		if err == nil {
			return nil
		}
		logrus.Warnf("%v handle error with retry: %v", runtime.FuncForPC(reflect.ValueOf(handle).Pointer()).Name(), err)

		inc++
		if inc > retryTimes {
			return err
		}

		time.Sleep(interval)
	}
}

func HandleErrorWithRetryAndReturn(handle func() (interface{}, error), retryTimes int, interval time.Duration) interface{} {
	inc := 0
	for {
		res, err := handle()
		if err == nil {
			return res
		}

		logrus.Warnf("%v handle error with retry: %v", runtime.FuncForPC(reflect.ValueOf(handle).Pointer()).Name(), err)

		inc++
		if inc > retryTimes {
			log.Fatal(err)
		}

		time.Sleep(interval)
	}
}
