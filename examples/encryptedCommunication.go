package main

import (
	"fmt"
	"os"
	"time"
	"runtime"
	"github.com/highras/fpnn-sdk-go/src/fpnn"
)

func main() {

	if len(os.Args) != 3 {
		fmt.Println("Usage:", os.Args[0], "<endpoint>", "<server-pem-key-file>")
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	client := fpnn.NewTCPClient(os.Args[1])
	err := client.EnableEncryptor(os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	for i := 0; i < 2; i++ {

		{
			questId := i * 2
			quest := fpnn.NewQuest("httpDemo")
			quest.Param("quest", "two")

			answer, err := client.SendQuest(quest)

			if answer != nil {
				if answer.IsException() {
					fmt.Println("Receive error answer for the", questId, "quest, method:", quest.Method(),
						"error code:", answer.WantInt("code"), "message:", answer.WantString("ex"))

				} else {
					fmt.Println("Receive answer for the", questId, "quest")
				}

			} else {
				fmt.Println("Send the", questId, "quest with method:", quest.Method(), "failed, err:", err)
			}
		}

		{
			questId := i * 2 + 1
			quest := fpnn.NewQuest("two way demo")
			quest.Param("int", 2)
			quest.Param("double", 3.3)
			quest.Param("boolean", true)

			err := client.SendQuestWithLambda(quest, func(answer *fpnn.Answer, errorCode int) {
				if errorCode == fpnn.FPNN_EC_OK {
					fmt.Println("Received answer of the", questId, " quest.")
				} else {
					fmt.Println("Received error answer of the", questId, " quest. code is ", errorCode)
				}
			})
			if err != nil {
				fmt.Println("Send the", questId, "quest", quest.Method(), "with lambda callback func failed, err:", err)
			}
		}
	}

	time.Sleep(time.Second)		//-- Waiting for the callbacks received & printed.
}