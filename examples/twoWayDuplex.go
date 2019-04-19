package main

import (
	"fmt"
	"os"
	"runtime"
	"../src/fpnn"
)

type DemoQuestPeocessor struct {}

func (processor *DemoQuestPeocessor) Process(method string) func(*fpnn.Quest) (*fpnn.Answer, error) {
	if method == "duplexQuest" {
		return processor.duplexQuest
	} else {
		fmt.Println("Receive unknown method:", method)
		return nil
	}
}

func (processor *DemoQuestPeocessor) duplexQuest(quest *fpnn.Quest) (*fpnn.Answer, error) {

	value, _ := quest.GetInt("int")
	fmt.Println("Receive server push. value of key 'int' is", value)
	return fpnn.NewAnswer(quest), nil
}

func main() {

	if len(os.Args) != 2 {
		fmt.Println("Usage:", os.Args[0], "<endpoint>")
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	client := fpnn.NewTCPClient(os.Args[1])
	client.SetQuestProcessor(&DemoQuestPeocessor{})

	quest := fpnn.NewQuest("duplex demo")
	quest.Param("duplex method", "duplexQuest")

	answer, err := client.SendQuest(quest)

	if err != nil {
		fmt.Println("Send duplex quest failed, err:", err)
		return
	}

	if answer.IsException() {
		fmt.Println("Received error answer of quest. Code:", answer.WantInt("code"))
	} else {
		fmt.Println("Received answer of quest.")
	}
}