package main

import (
	"os"
	"fmt"
	"time"
	"runtime"
	"../src/fpnn"
)

func showSignDesc() {
	fmt.Println("Sign:")
	fmt.Println("    +: establish connection")
	fmt.Println("    ~: close connection")
	fmt.Println("    #: connection error")

	fmt.Println("    *: send sync quest")
	fmt.Println("    &: send async quest")

	fmt.Println("    ^: sync answer Ok")
	fmt.Println("    ?: sync answer exception")
	fmt.Println("    |: sync answer exception by connection closed")
	fmt.Println("    (: sync operation fpnn exception")
	fmt.Println("    ): sync operation unknown exception")

	fmt.Println("    $: async answer Ok")
	fmt.Println("    @: async answer exception")
	fmt.Println("    ;: async answer exception by connection closed")
	fmt.Println("    {: async operation fpnn exception")
	fmt.Println("    }: async operation unknown exception")

	fmt.Println("    !: close operation")
	fmt.Println("    [: close operation fpnn exception")
	fmt.Println("    ]: close operation unknown exception")
}

func genQuest() *fpnn.Quest {

	quest := fpnn.NewQuest("two way demo")
	quest.Param("quest", "one");
	quest.Param("int", 2); 
	quest.Param("double", 3.3);
	quest.Param("boolean", true);
	quest.Param("ARRAY", []interface{}{"first_vec", 4})

	mapData := make(map[string]interface{})
	mapData["map1"] = "first_map"
	mapData["map2"] = true
	mapData["map3"] = 5
	mapData["map4"] = 5.7
	mapData["map5"] = "中文"

	quest.Param("MAP", mapData)

	return quest
}

func testThread(client *fpnn.TCPClient, count int, finishChan chan bool) {
	
	defer func () {
		finishChan <- true
	} ()
	
	act := 0
	for i := 0; i < count; i++ {
		index := time.Now().UnixNano() % 64
		if i >= 10 {
			if index < 6 {
				act = 2	//-- close operation
			} else if index < 32 {
				act = 1	//-- async quest
			} else {
				act = 0	//-- sync quest
			}
		} else {
			act = int(index & 0x1)
		}

		switch act {
			case 0:
			
				fmt.Print("*")
				answer, _ := client.SendQuest(genQuest())
				if answer != nil {
					if answer.Status() == 0 {
						fmt.Print("^")
					} else {
						code := answer.WantInt("code")
						if code == fpnn.FPNN_EC_CORE_CONNECTION_CLOSED || code == fpnn.FPNN_EC_CORE_INVALID_CONNECTION {
							fmt.Print("|")
						} else {
							fmt.Print("?")
						}
					}
				} else {
					fmt.Print("?")
				}
			
			case 1:
				fmt.Print("&")
				err := client.SendQuestWithLambda(genQuest(), func(answer *fpnn.Answer, errorCode int) {
				if errorCode == fpnn.FPNN_EC_OK {
					fmt.Print("$")
				} else if errorCode == fpnn.FPNN_EC_CORE_CONNECTION_CLOSED || errorCode == fpnn.FPNN_EC_CORE_INVALID_CONNECTION {
					fmt.Print(";")
				} else {
					fmt.Print("@")
				}
			})
			if err != nil {
				fmt.Print("@")
			}

			case 2:
				fmt.Print("!")
				client.Close()
		}
	}
}

func test(client *fpnn.TCPClient, threadCount int, questCount int) {

	fmt.Println("========[ Test: go routine ", threadCount, ", per thread quest: ", questCount, " ]==========")

	finishChan := make(chan bool)

	for i := 0 ; i < threadCount; i++ {
		go testThread(client, questCount, finishChan)
	}

	count := 0
	for _ = range finishChan {
		count += 1
		if count == threadCount {
			fmt.Println()
			fmt.Println()

			return
		}
	}
}

func main() {

	if len(os.Args) != 3 {
		fmt.Println("Usage:", os.Args[0], "ip", "port")
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	endpoint := os.Args[1] + ":" + os.Args[2]
	client := fpnn.NewTCPClient(endpoint)

	client.SetOnConnectedCallback(func(connId uint64) {
			fmt.Print("+")
		})

	client.SetOnClosedCallback(func(connId uint64) {
			fmt.Print("~")
		})

	showSignDesc()

	test(client, 10, 30000)
	test(client, 20, 30000)
	test(client, 30, 30000)
	test(client, 40, 30000)
	test(client, 50, 30000)
	test(client, 60, 30000)

	//-- Special for Go.
	test(client, 100, 30000)
	test(client, 200, 30000)
}
