package main

import (
	// "fmt"
	"strconv"
	"time"
)

var machine_id = 1
var server_id = 1


func generate_uniqueid() string {

	timens := time.Now().UnixNano()

	var signedBit int64 = 0
	datacenterID := int64(server_id)
	timestamp := timens / 1e6
	machineID := int64(machine_id)
	sequenceID := (timens % 1e6) / 4096

	uid := signedBit<<63 | timestamp<<22 | datacenterID<<17 | machineID<<12 | sequenceID

	return strconv.FormatInt(uid, 10)
}