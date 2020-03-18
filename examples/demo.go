package main

import (
	"fmt"
	"os"
	"time"
	"runtime"
	"github.com/highras/fpnn-sdk-go/src/fpnn"
)

type CallbackDemo struct{}

func (cb *CallbackDemo) OnAnswer(answer *fpnn.Answer) {
	fmt.Println("Receive answer in callback. Answer:", answer)
}

func (cb *CallbackDemo) OnException(answer *fpnn.Answer, errorCode int) {
	fmt.Println("Receive exception in callback. Answer:", answer, "error code:", errorCode)
}

func main() {

	if len(os.Args) != 2 {
		fmt.Println("Usage:", os.Args[0], "<endpoint>")
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	client := fpnn.NewTCPClient(os.Args[1])

	//---------------[ Two way demo: sync mode ]----------------------//
	/*
		Summary:
			Demo for create empty two way quest
			Demo for send quest and receive answer in sync mode
			Demo for fetch field value in answer/quest with panic
			Demo for fetch field value in answer/quest without panic
			Demo for process error/exception answer in FPNN standard best practice
	*/
	{
		quest := fpnn.NewQuest("two way demo")			//-- empty quest
		answer, err := client.SendQuest(quest)

		if answer != nil {
			if answer.IsException() {
				code := answer.WantInt("code")		//-- Fetch field with panic when type convert faild
				ex := answer.WantString("ex")		//-- Fetch field with panic when type convert faild

				fmt.Println("Receive error answer for quest", quest.Method(), answer, "error code:", code, "message:", ex)

			} else {
				value, _ := answer.GetString("Simple")		//-- Fetch field without panic

				fmt.Println("Receive answer for quest", quest.Method(), answer, "key 'Simple' is", value)
			}

		} else {
			fmt.Println("Send quest", quest.Method(), "failed, err:", err)
		}
	}

	//---------------[ Two way demo: async mode with callback object ]----------------------//
	/*
		Summary:
			Demo for create two way quest
			Demo for add data into quest
			Demo for using callback object & interface
			Demo for send quest and receive answer in async mode
	*/
	{
		quest := fpnn.NewQuest("two way demo")
		quest.Param("key-1", 123)
		quest.Param("key-2", "string value")

		if err := client.SendQuestWithCallback(quest, &CallbackDemo{}); err != nil {
			fmt.Println("Send quest", quest.Method(), "with callback failed, err:", err)
		}
	}

	//---------------[ Two way demo: async mode with lambda callback func ]----------------------//
	/*
		Summary:
			Demo for create two way quest
			Demo for add data into quest
			Demo for using lambda callback func
			Demo for send quest and receive answer in async mode
			Demo for fetch field value in answer/quest with panic
	*/
	{
		quest := fpnn.NewQuest("httpDemo")
		quest.Param("key-1", 123)
		quest.Param("key-2", "string value")

		err := client.SendQuestWithLambda(quest, func(answer *fpnn.Answer, errorCode int) {
			if errorCode == fpnn.FPNN_EC_OK {
				value := answer.WantInt("TEST")		//-- Fetch field with panic
				fmt.Println("Receive answer in lambda callback func. answer:", answer, "key 'TEST' is", value)
			} else {
				fmt.Println("Receive exception in lambda callback func. Answer:", answer, "error code:", errorCode)
			}
		})
		if err != nil {
			fmt.Println("Send quest", quest.Method(), "with lambda callback func failed, err:", err)
		}
	}

	//---------------[ One way demo ]----------------------//
	/*
		Summary:
			Demo for create one way quest
			Demo for add data into quest
	*/
	{
		quest := fpnn.NewOneWayQuest("one way demo")
		quest.Param("key-1", 123)

		if _, err := client.SendQuest(quest); err != nil {
			fmt.Println("Send one way quest", quest.Method(), "failed, err:", err)
		}
	}

	time.Sleep(time.Second)		//-- Waiting for the callbacks received & printed.
}