package utils

import "strings"

const (
	ZeroAddress = "0x0000000000000000000000000000000000000000"
)

var (
	NoMreRetryErrors = []string{
		"execution reverted",
		"invalid jump destination",
		"invalid opcode",
		// Ethereum says 1024 is the stack sizes limit, so this is deterministic.
		"stack limit reached 1024",
	}
)

func HitNoMoreRetryErrors(err error) bool {
	if err == nil {
		return false
	}

	for _, msg := range NoMreRetryErrors {
		if strings.Contains(err.Error(), msg) {
			return true
		}
	}

	return false
}
