package fpnn

import (
	"log"
	"sync"
	"time"
	"errors"
)

type AnswerCallback interface {
	OnAnswer(answer *Answer)
	OnException(answer *Answer, errorCode int)
}

type QuestProcessor interface {
	Process(method string) func(*Quest) (*Answer, error)
}

type TCPClient struct {
	mutex			sync.Mutex
	autoReconnect	bool
	endpoint		string
	timeout			time.Duration
	connectTimeout	time.Duration
	conn			*tcpConnection
	questProcessor	QuestProcessor
	onConnected		func(connId uint64)
	onClosed		func(connId uint64)
	logger			*log.Logger
}

func NewTCPClient(endpoint string) *TCPClient {

	client := &TCPClient{}
	
	client.autoReconnect = true
	client.endpoint = endpoint
	client.timeout = Config.questTimeout
	client.connectTimeout = Config.connectTimeout

	//runtime.SetFinalizer(client, client.Close)
	return client
}

func (client *TCPClient) SetAutoReconnect(autoReconnect bool) {
	client.autoReconnect = autoReconnect
}

func (client *TCPClient) SetConnectTimeOut(timeout time.Duration) {
	client.connectTimeout = timeout
}

func (client *TCPClient) SetQuestTimeOut(timeout time.Duration) {
	client.timeout = timeout
}

func (client *TCPClient) SetQuestProcessor(questProcessor QuestProcessor) {
	client.questProcessor = questProcessor
}

func (client *TCPClient) SetOnConnectedCallback(onConnected func(connId uint64)) {
	client.onConnected = onConnected
}

func (client *TCPClient) SetOnClosedCallback(onClosed func(connId uint64)) {
	client.onClosed = onClosed
}

func (client *TCPClient) SetLogger(logger *log.Logger) {
	client.logger = logger
}

func (client *TCPClient) IsConnected() bool {
	client.mutex.Lock()

	if client.conn == nil {
		client.mutex.Unlock()
		return false
	}

	conn := client.conn
	client.mutex.Unlock()

	return conn.isConnected()
}

func (client *TCPClient) Endpoint() string {
	return client.endpoint
}

func (client *TCPClient) Connect() bool {

	conn := newTCPConnection(client.logger, client.onConnected, client.onClosed, client.questProcessor)

	client.mutex.Lock()
	defer client.mutex.Unlock()

	if client.conn != nil && client.conn.isConnected() {
		return true
	}

	client.conn = conn
	ok := conn.connect(client.endpoint, client.connectTimeout)

	return ok
}

func (client *TCPClient) Dail() bool {
	return client.Connect()
}

func (client *TCPClient) checkConnection() *tcpConnection {

	ok := client.IsConnected()
	if !ok {
		if client.autoReconnect {
			ok = client.Connect()
		} else {
			return nil
		}
	}

	client.mutex.Lock()
	defer client.mutex.Unlock()

	if client.conn != nil && client.conn.isConnected() {
		return client.conn
	}
	return nil
}

func (client *TCPClient) realSendQuest(quest *Quest, cb *connCallback) error {
	conn := client.checkConnection()
	if conn == nil {
		return errors.New("Connection is invalid.")
	}

	return conn.sendQuest(quest, cb)
}

func (client *TCPClient) SendQuest(quest *Quest) (*Answer, error) {

	if !quest.isTwoWay {
		err := client.realSendQuest(quest, nil)
		return nil, err
	}

	//------------ send two way quest ---------------//
	answerChan := make(chan *Answer)

	cb := &connCallback{}
	cb.timeout = time.Now().Unix() + int64(Config.questTimeout / time.Second)
	cb.callbackFunc = func(answer *Answer, errorCode int) {
		if answer == nil {
			answer = newErrorAnswerWitSeqNum(quest.seqNum, errorCode, "")	
		}

		answerChan <- answer
	}

	err := client.realSendQuest(quest, cb)
	if err != nil {
		return nil, err
	}

	answer := <- answerChan

	return answer, nil
}

func (client *TCPClient) SendQuestWithTimeout(quest *Quest, timeout time.Duration) (*Answer, error) {

	if !quest.isTwoWay {
		err := client.realSendQuest(quest, nil)
		return nil, err
	}

	//------------ send two way quest ---------------//
	answerChan := make(chan *Answer)

	cb := &connCallback{}
	cb.timeout = time.Now().Unix() + int64(timeout / time.Second)
	cb.callbackFunc = func(answer *Answer, errorCode int) {
		if answer == nil {
			answer = newErrorAnswerWitSeqNum(quest.seqNum, errorCode, "")	
		}

		answerChan <- answer
	}

	err := client.realSendQuest(quest, cb)
	if err != nil {
		return nil, err
	}

	answer := <- answerChan
	
	return answer, nil
}

func (client *TCPClient) SendQuestWithCallback(quest *Quest, callback AnswerCallback) error {

	var cb *connCallback

	if quest.isTwoWay {
		cb = &connCallback{}
		
		cb.timeout = time.Now().Unix() + int64(Config.questTimeout / time.Second)
		cb.callback = callback
	}

	return client.realSendQuest(quest, cb)
}

func (client *TCPClient) SendQuestWithCallbackTimeout(quest *Quest, callback AnswerCallback, timeout time.Duration) error {
	
	var cb *connCallback

	if quest.isTwoWay {
		cb = &connCallback{}
		
		cb.timeout = time.Now().Unix() + int64(timeout / time.Second)
		cb.callback = callback
	}

	return client.realSendQuest(quest, cb)
}

func (client *TCPClient) SendQuestWithLambda(quest *Quest, callback func(answer *Answer, errorCode int)) error {
	
	var cb *connCallback

	if quest.isTwoWay {
		cb = &connCallback{}
		
		cb.timeout = time.Now().Unix() + int64(Config.questTimeout / time.Second)
		cb.callbackFunc = callback
	}

	return client.realSendQuest(quest, cb)
}

func (client *TCPClient) SendQuestWithLambdaTimeout(quest *Quest, callback func(answer *Answer, errorCode int), timeout time.Duration) error {
	
	var cb *connCallback

	if quest.isTwoWay {
		cb = &connCallback{}
		
		cb.timeout = time.Now().Unix() + int64(timeout / time.Second)
		cb.callbackFunc = callback
	}

	return client.realSendQuest(quest, cb)
}

func (client *TCPClient) Close() {

	client.mutex.Lock()

	if client.conn == nil {
		return
	}

	conn := client.conn
	client.conn = nil

	client.mutex.Unlock()

	conn.close()
}