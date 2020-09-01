# FPNN Go SDK

[TOC]

## Install & Update


### Install

	go get github.com/highras/fpnn-sdk-go/src/fpnn

### Update

	go get -u github.com/highras/fpnn-sdk-go/src/fpnn

### Use

	import "github.com/highras/fpnn-sdk-go/src/fpnn"


## Usage

### Create

	client := fpnn.NewTCPClient(endpoint string)

**endpoint** format: `"hostname/ip" + ":" + "port"`.  
e.g. `"localhost:8000"`


### Configure (Optional)

* Basic configs

		client.SetAutoReconnect(autoReconnect bool)
		client.SetConnectTimeOut(timeout time.Duration)
		client.SetQuestTimeOut(timeout time.Duration)
		client.SetLogger(logger *log.Logger)

* Set Duplex Mode (Server Push)

		client.SetQuestProcessor(questProcessor QuestProcessor)

* Set connection events' callbacks

		client.SetOnConnectedCallback(onConnected func(connId uint64, endpoint string, connected bool))
		client.SetOnClosedCallback(onClosed func(connId uint64, endpoint string))

* Config encrypted connection
	
		client.EnableEncryptor(pemKeyPath string)
		client.EnableEncryptor(pemKeyData []byte)

	FPNN Go SDK using **ECC**/**ECDH** to exchange the secret key, and using **AES-128** or **AES-256** in **CFB** mode to encrypt the whole session in **stream** way.


### Send Quest

	answer, err := client.SendQuest(quest *Quest)
	answer, err := client.SendQuest(quest *Quest, timeout time.Duration)

	err := client.SendQuestWithCallback(quest *Quest, callback AnswerCallback)
	err := client.SendQuestWithCallback(quest *Quest, callback AnswerCallback, timeout time.Duration)

	err := client.SendQuestWithLambda(quest *Quest, callback func(answer *Answer, errorCode int))
	err := client.SendQuestWithLambda(quest *Quest, callback func(answer *Answer, errorCode int), timeout time.Duration)


### Close (Optional)

	client.Close()


### SDK Version

	fmt.Println("FPNN Go SDK Version:", fpnn.SDKVersion)

## API docs

Please refer: [API docs](API.md)


## Directory structure

* **<fpnn-sdk-go>/src**

	Codes of SDK.

* **<fpnn-sdk-go>/example**

	Examples codes for using this SDK.  
	Testing server is <fpnn>/core/test/serverTest. Refer: [Cpp codes of serverTest](https://github.com/highras/fpnn/blob/master/core/test/serverTest.cpp)

* **<fpnn-sdk-go>/test**

	+ **<fpnn-sdk-go>/test/asyncStressClient.go**

		Stress & Concurrent testing codes for SDK.  
		Testing server is <fpnn>/core/test/serverTest. Refer: [Cpp codes of serverTest](https://github.com/highras/fpnn/blob/master/core/test/serverTest.cpp)

	+ **<fpnn-sdk-go>/test/singleClientConcurrentTest.go**

		Stability testing codes for SDK.  
		Testing server is <fpnn>/core/test/serverTest. Refer: [Cpp codes of serverTest](https://github.com/highras/fpnn/blob/master/core/test/serverTest.cpp)