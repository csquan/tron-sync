package mysqldb

import "testing"

func TestTransTraceAddressToString(t *testing.T) {
	opcode := "CALL"
	traceAddress1 := []uint64{0, 0, 5}
	res := transTraceAddressToString(opcode, traceAddress1)
	if res == "call_0_0_5" {
		t.Log("right")
	} else {
		t.Errorf("expect: call_0_0_5, res: %s", res)
		return
	}
	traceAddress1 = nil
	res = transTraceAddressToString(opcode, traceAddress1)
	if res == "call" {
		t.Log("right")
	} else {
		t.Errorf("expect: call, res %s", res)
		return
	}
}
