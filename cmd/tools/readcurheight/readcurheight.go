package main

import (
	"encoding/binary"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	fname := `height.record`
	f, err := os.Open(fname)
	if err != nil {
		log.Fatalf("open hfile err:%v", err)
	}
	hs, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("read height record err:%v", err)
	}
	if len(hs) == 0 {
		log.Printf("read empty")
	}
	height := binary.BigEndian.Uint64(hs)
	log.Printf("cur block height:%d", height)
}
