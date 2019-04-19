package main

import (
	"fmt"
	"os"
	"time"
	"runtime"
	"../src/fpnn"
)

func main() {

	if len(os.Args) != 2 {
		fmt.Println("Usage:", os.Args[0], "<endpoint>")
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	endpoint := os.Args[1]
	client := fpnn.NewTCPClient(endpoint)

	client.SetOnConnectedCallback(func(connId uint64) {
			fmt.Printf("Connected to %s, connId is %d\n", endpoint, connId)
		})

	client.SetOnClosedCallback(func(connId uint64) {
			fmt.Println("Connection no", endpoint, "closed, connId is", connId)
		})

	if ok := client.Connect(); !ok {
		fmt.Println("Connect to", endpoint, "failed")
	}

	client.Close()

	time.Sleep(time.Second)		//-- Waiting for the closed event printed.
}