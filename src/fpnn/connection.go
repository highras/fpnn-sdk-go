package fpnn

import (
	"log"
	"io"
	"fmt"
	"net"
	"sync"
	"time"
	"runtime"
	"bytes"
	"errors"
	"unsafe"
	"encoding/binary"
)

type rawData struct {
	header		[]byte
	body		[]byte
}

func newRawData() *rawData {
	data := &rawData{}
	data.header = make([]byte, 12)
	return data
}

type connCallback struct {
	timeout			int64
	callback		AnswerCallback
	callbackFunc 	func(answer *Answer, errorCode int)
}

type tcpConnection struct {
	mutex			sync.Mutex
	answerMap		map[uint32]*connCallback
	conn			net.Conn
	seqNum			uint32
	closeSignChan	chan bool
	writeChan		chan []byte
	ticker			*time.Ticker
	connected		bool
	logger			*log.Logger
	onConnected		func(connId uint64)
	onClosed		func(connId uint64)
	questProcessor	QuestProcessor
	activeClosed	bool
}

func newTCPConnection(logger *log.Logger, onConnected func(connId uint64), onClosed func(connId uint64),
	questProcessor QuestProcessor) *tcpConnection {

	conn := new(tcpConnection)
	conn.answerMap = make(map[uint32]*connCallback)
	conn.closeSignChan = make(chan bool)
	conn.writeChan = make(chan []byte, Config.netChanBufferSize)

	now := time.Now()
	conn.seqNum = uint32(now.UnixNano() & 0xFFF)

	conn.connected = false
	if (logger != nil) {
		conn.logger = logger
	} else {
		conn.logger = Config.logger
	}

	conn.onConnected = onConnected
	conn.onClosed = onClosed

	conn.questProcessor = questProcessor
	conn.activeClosed = false
	
	return conn
}

func (conn *tcpConnection) isConnected() bool {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	return conn.connected
}

func cleanTCPConnection(conn *tcpConnection) {
	go conn.close()
}

func (conn *tcpConnection) realConnect(endpoint string, timeout time.Duration) (ok bool) {
	var err error

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if conn.connected {
		return true
	}

	conn.conn, err = net.DialTimeout("tcp", endpoint, timeout);
	if err != nil {
		conn.connected = false
		conn.logger.Printf("[ERROR] Connect to %s failed, err: %v", endpoint, err)
		return false
	}

	go conn.readLoop()
	go conn.workLoop()
	conn.ticker = time.NewTicker(1 * time.Second)
	conn.connected = true

	runtime.SetFinalizer(conn, cleanTCPConnection)
	return true
}

func (conn *tcpConnection) connect(endpoint string, timeout time.Duration) (ok bool) {
	if conn.realConnect(endpoint, timeout) {
		if conn.onConnected != nil {
			var addr uintptr = uintptr(unsafe.Pointer(conn))
			conn.onConnected(uint64(addr))
		}
		return true
	}
	return false
}

func (conn *tcpConnection) readRawData() *rawData {
	buffer := newRawData()

	if _, err := io.ReadFull(conn.conn, buffer.header); err != nil {
		if err == io.EOF {
		} else {
			conn.mutex.Lock()
			actived := conn.activeClosed
			conn.mutex.Unlock()

			if !actived {
				conn.logger.Printf("[ERROR] Read header from connection failed, err: %v", err)
			}
		}
		return nil
	}

	var payloadSize uint32
	headReader := bytes.NewReader(buffer.header[8:])
	binary.Read(headReader, binary.LittleEndian, &payloadSize)
	
	if payloadSize > uint32(Config.maxPayloadSize) {
		conn.logger.Printf("[ERROR] Read huge payload, size: %d", payloadSize)
		return nil
	}

	switch buffer.header[6] {
	case MessageTypeOneWay:
		buffer.body = make([]byte, payloadSize + uint32(buffer.header[7]))
	case MessageTypeTwoWay:
		buffer.body = make([]byte, payloadSize + 4 + uint32(buffer.header[7]))
	case MessageTypeAnswer:
		buffer.body = make([]byte, payloadSize + 4)
	default:
		conn.logger.Printf("[ERROR] Receive invalid FPNN MType: %d", buffer.header[6])
		return nil
	}

	if _, err := io.ReadFull(conn.conn, buffer.body); err != nil {
		if err == io.EOF {
		} else {
			conn.logger.Printf("[ERROR] Read body from connection failed, err: %v", err)
		}
		return nil
	}

	return buffer
}

func (conn *tcpConnection) processRawData(data *rawData) bool {
	switch data.header[6] {

	case MessageTypeOneWay, MessageTypeTwoWay:

		quest, err := NewQuestWithRawData(data)
		if err != nil {
			conn.logger.Printf("[ERROR] Decode quest failed, err: %v", err)
			return false
		}
		
		conn.dealQuest(quest)

	case MessageTypeAnswer:
		answer, err := NewAnswerWithRawData(data)
		if err != nil {
			conn.logger.Printf("[ERROR] Decode answer failed, err: %v", err)
			return false
		}

		conn.mutex.Lock()
		callback, ok := conn.answerMap[answer.seqNum]
		if ok {
			delete(conn.answerMap, answer.seqNum)
			conn.mutex.Unlock()

			go callAnswerCallback(answer, callback)
		} else {
			conn.mutex.Unlock()
			conn.logger.Printf("[ERROR] Received invalid answer, seqNum: %d", answer.seqNum)
		}
	}

	return true
}

func callAnswerCallback(answer *Answer, cb *connCallback) {

	if cb.callback != nil {

		if !answer.IsException() {
			cb.callback.OnAnswer(answer)
		} else {
			code, _ := answer.GetInt("code")
			cb.callback.OnException(answer, code)
		}
		return
	}

	if cb.callbackFunc != nil {

		code, _ := answer.GetInt("code")
		cb.callbackFunc(answer, code)
	}
}

func (conn *tcpConnection) dealQuest(quest *Quest) {

	defer func() {
		if r := recover(); r != nil {
			conn.logger.Printf("[ERROR] Process quest panic. Method: %s, panic: %v.", quest.method, r)
		}
	}()

	if conn.questProcessor != nil {
		conn.realDealQuest(quest)
	} else {
		if quest.isTwoWay {

			answer := NewErrorAnswer(quest, FPNN_EC_CORE_UNKNOWN_METHOD, "Client quest processor is unconfiged.")
			if err := conn.sendAnswer(answer); err == nil {
				conn.logger.Printf("[ERROR] Received twoway quest, but quest processor is nil. Method: %s.", quest.method)
			} else {
				conn.logger.Printf("[ERROR] Received twoway quest, but quest processor is nil. Method: %s. Send default answer error, err: %v",
					quest.method, err)
			}
			
		} else {
			conn.logger.Printf("[ERROR] Received oneway quest, but quest processor is nil. Method: %s.", quest.method)
		}
	}
}

func (conn *tcpConnection) realDealQuest(quest *Quest) {

	processFunc := conn.questProcessor.Process(quest.method)
	if processFunc == nil {
		if quest.isTwoWay {

			answer := NewErrorAnswer(quest, FPNN_EC_CORE_UNKNOWN_METHOD, "Method function is unconfiged.")
			if err := conn.sendAnswer(answer); err == nil {
				conn.logger.Printf("[ERROR] Received twoway quest, but method function is unconfiged. Method: %s.", quest.method)
			} else {
				conn.logger.Printf("[ERROR] Received twoway quest, but method function is unconfiged. Method: %s. Send default answer error, err: %v",
					quest.method, err)
			}
			
		} else {
			conn.logger.Printf("[ERROR] Received oneway quest, but method function is unconfiged. Method: %s.", quest.method)
		}

		return
	}

	answer, err := processFunc(quest)
	if err != nil {
		conn.logger.Printf("[ERROR] Process quest error. Method: %s, err: %v", quest.method, err)
	}

	if answer != nil {

		if quest.isTwoWay {
			if err := conn.sendAnswer(answer); err != nil {
				conn.logger.Printf("[ERROR] Send quest answer error. Method: %s, err: %v", quest.method, err)
			}
		} else {
			conn.logger.Printf("[ERROR] Return answer for oneway quest. Method: %s, answer: %v", quest.method, answer)
		}

	} else {

		if quest.isTwoWay {

			ex := "Quest processer don't return invalid answer."
			if err != nil {
				ex = fmt.Sprintf("Client error: %v", err)
			}
			
			answer = NewErrorAnswer(quest, FPNN_EC_CORE_UNKNOWN_ERROR, ex)

			if sendErr := conn.sendAnswer(answer); sendErr != nil {
				conn.logger.Printf("[ERROR] Send quest error answer error. Method: %s, send error: %v, quest error: %v",
					quest.method, sendErr, err)
			}
		}
	}
}

func (conn *tcpConnection) readLoop() {

	defer conn.close()

	for {
		data := conn.readRawData()
		if data == nil {
			return
		}

		ok := conn.processRawData(data)
		if !ok {
			return
		}
	}
}

func (conn *tcpConnection) workLoop() {

	for {
		select {
		case binData := <-conn.writeChan:
			
			if _, err := conn.conn.Write(binData); err != nil {
				conn.logger.Printf("[ERROR] Write data to connection failed, err: %v", err)
				close(conn.writeChan)
				conn.close()
				return
			}

		case <-conn.ticker.C:
			conn.cleanTimeoutedCallback()

		case <-conn.closeSignChan:
			return
		}
	}
}

func (conn *tcpConnection) cleanTimeoutedCallback() {

	now := time.Now()
	curr := now.Unix()

	timeoutedMap := make(map[uint32]*connCallback)
	{
		conn.mutex.Lock()

		for seqNum, callback := range conn.answerMap {
			if callback.timeout <= curr {
				timeoutedMap[seqNum] = callback
			}
		}

		for seqNum, _ := range timeoutedMap {
			delete(conn.answerMap, seqNum)
		}

		conn.mutex.Unlock()
	}

	for seqNum, callback := range timeoutedMap {

		answer := newErrorAnswerWitSeqNum(seqNum, FPNN_EC_CORE_TIMEOUT, "Quest is timeout.")
		go callAnswerCallback(answer, callback)
	}
}

func (conn *tcpConnection) cleanCallbackMap() {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	for seqNum, callback := range conn.answerMap {

		answer := newErrorAnswerWitSeqNum(seqNum, FPNN_EC_CORE_CONNECTION_CLOSED, "Connection is closed.")
		go callAnswerCallback(answer, callback)
	}
}

func (conn *tcpConnection) sendBinaryData(binData []byte) {
	defer func() {
		recover();
	}()
	
	conn.writeChan <- binData
}

func (conn *tcpConnection) sendQuest(quest *Quest, callback *connCallback) error {

	conn.mutex.Lock()
	if conn.seqNum == 0 {
		conn.seqNum = 1
	}

	quest.seqNum = conn.seqNum
	conn.seqNum += 1
	conn.mutex.Unlock()

	binData, err := quest.Raw()
	if err != nil {
		return err
	}

	conn.mutex.Lock()
	if !conn.connected {
		conn.mutex.Unlock()
		return errors.New("Connection is broken.")
	}

	if callback != nil {
		conn.answerMap[quest.seqNum] = callback
	}

	conn.sendBinaryData(binData)

	conn.mutex.Unlock()

	return nil
}

func (conn *tcpConnection) sendAnswer(answer *Answer) error {
	
	binData, err := answer.Raw()
	if err != nil {
		return err
	}

	conn.mutex.Lock()
	if !conn.connected {
		conn.mutex.Unlock()
		return errors.New("Connection is broken.")
	}

	conn.sendBinaryData(binData)

	conn.mutex.Unlock()

	return nil
}

func (conn *tcpConnection) close() {

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if conn.connected {

		conn.activeClosed = true
		err := conn.conn.Close()
		if err != nil {
			conn.logger.Printf("[ERROR] Close connection failed, err: %v", err)
			return
		}

		conn.ticker.Stop()
		conn.closeSignChan <- true
		conn.connected = false

		conn.mutex.Unlock()
		conn.cleanCallbackMap()

		if conn.onClosed != nil {
			var addr uintptr = uintptr(unsafe.Pointer(conn))
			go conn.onClosed(uint64(addr))
		}
		conn.mutex.Lock()
	}
}