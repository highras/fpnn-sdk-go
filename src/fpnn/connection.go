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

type encryptionInfo struct {
	aesKeyBits		int
	secret			[]byte
	eccPublicKey	[]byte
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
	onConnected		tcpClientConnectedCallback
	onClosed		tcpClientCloseCallback
	questProcessor	QuestProcessor
	activeClosed	bool
	encryptInfo		*encryptionInfo
	lastActiveTime  int64
}

func newTCPConnection(logger *log.Logger, onConnected tcpClientConnectedCallback, onClosed tcpClientCloseCallback,
	questProcessor QuestProcessor) *tcpConnection {

	conn := new(tcpConnection)
	conn.answerMap = make(map[uint32]*connCallback)
	conn.closeSignChan = make(chan bool)
	conn.writeChan = make(chan []byte, Config.netChanBufferSize)

	now := time.Now()
	conn.seqNum = uint32(now.UnixNano() & 0xFFF)

	conn.connected = false
	if logger != nil {
		conn.logger = logger
	} else {
		conn.logger = Config.logger
	}

	conn.onConnected = onConnected
	conn.onClosed = onClosed

	conn.questProcessor = questProcessor
	conn.activeClosed = false
	conn.lastActiveTime = time.Now().Unix()

	return conn
}

func (conn *tcpConnection) isConnected() bool {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	return conn.connected
}

func (conn *tcpConnection) getActiveTime() int64 {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	return conn.lastActiveTime
}

func cleanTCPConnection(conn *tcpConnection) {
	go conn.close()
}

func (conn *tcpConnection) enableEncryptor(aesBits int, serverKey *eccPublicKeyInfo) bool {

	info, err := makeEcdhInfo(serverKey)
	if err != nil {
		conn.logger.Printf("[ERROR] Make ecdh info error, err: %v", err)
		return false
	}

	conn.encryptInfo = &encryptionInfo{}
	conn.encryptInfo.aesKeyBits = aesBits
	conn.encryptInfo.eccPublicKey = info.publicKey
	conn.encryptInfo.secret = info.secret

	return true
}

func (conn *tcpConnection) realConnect(endpoint string, timeout time.Duration) (ok bool) {
	var err error

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if conn.connected {
		return true
	}

	conn.conn, err = net.DialTimeout("tcp", endpoint, timeout)
	if err != nil {
		conn.connected = false
		conn.logger.Printf("[ERROR] Connect to %s failed, err: %v", endpoint, err)
		return false
	}
	conn.lastActiveTime = time.Now().Unix()
	conn.ticker = time.NewTicker(1 * time.Second)

	go conn.readLoop()
	go conn.workLoop()

	conn.connected = true

	runtime.SetFinalizer(conn, cleanTCPConnection)
	return true
}

func (conn *tcpConnection) connect(endpoint string, timeout time.Duration) (ok bool) {
	ok = conn.realConnect(endpoint, timeout)
	if conn.onConnected != nil {
		if ok {
			var addr uintptr = uintptr(unsafe.Pointer(conn))
			go conn.onConnected(uint64(addr), endpoint, ok)
		} else {
			go conn.onConnected(0, endpoint, ok)
		}
	}
	return
}

func (conn *tcpConnection) readRawData(decoder *encryptor) *rawData {
	buffer := newRawData()

	if _, err := io.ReadFull(conn.conn, buffer.header); err != nil {
		if err == io.EOF {
		}
		return nil
	}

	if decoder != nil {
		decHeader := decoder.decrypt(buffer.header)
		buffer.header = decHeader
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
		}
		return nil
	}

	if decoder != nil {
		decBody := decoder.decrypt(buffer.body)
		buffer.body = decBody
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
		conn.mutex.Lock()
		conn.lastActiveTime = time.Now().Unix()
		conn.mutex.Unlock()

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
			conn.lastActiveTime = time.Now().Unix()
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

	var decoder *encryptor
	if conn.encryptInfo != nil {
		decoder = newEncryptor(conn.encryptInfo.secret, conn.encryptInfo.aesKeyBits)
	}

	for {
		data := conn.readRawData(decoder)
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

	encoder, err := conn.prepareEncryptedConnection()
	if err != nil {
		conn.logger.Printf("[ERROR] Prepare ecnryption handshake failed, err: %v", err)
		close(conn.writeChan)
		conn.close()
		return
	}

	for {
		select {
		case binData := <-conn.writeChan:

			if encoder != nil {
				encBinary := encoder.encrypt(binData)
				binData = encBinary
			}

			if _, err := conn.conn.Write(binData); err != nil {
				conn.logger.Printf("[ERROR] Write data to connection failed, err: %v", err)
				go conn.close()
			}

		case <-conn.ticker.C:
			go conn.cleanTimeoutedCallback()

		case <-conn.closeSignChan:
			return
		}
	}
}

func (conn *tcpConnection) prepareEncryptedConnection() (*encryptor, error) {

	if conn.encryptInfo != nil {
		binData, err := conn.prepareECDHQuest()
		if err != nil {
			return nil, err
		}

		if _, err := conn.conn.Write(binData); err != nil {
			return nil, err
		}

		encoder := newEncryptor(conn.encryptInfo.secret, conn.encryptInfo.aesKeyBits)
		return encoder, nil
	} else {
		return nil, nil
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

func (conn *tcpConnection) prepareECDHQuest() ([]byte, error) {

	quest := NewQuest("*key")
	quest.Param("publicKey", conn.encryptInfo.eccPublicKey)
	quest.Param("bits", conn.encryptInfo.aesKeyBits)
	quest.Param("streamMode", true)

	callback := &connCallback{}
	callback.timeout = time.Now().Unix() + int64(Config.questTimeout / time.Second)
	callback.callbackFunc = func(answer *Answer, errorCode int) {
		if errorCode != FPNN_EC_OK {
			conn.logger.Printf("[ERROR] Encryption handshake failed, errorCode: %d", errorCode)
		}
	}

	//---------- prepare sending ---------//
	conn.mutex.Lock()
	if conn.seqNum == 0 {
		conn.seqNum = 1
	}

	quest.seqNum = conn.seqNum
	conn.seqNum += 1

	if conn.connected {
		conn.answerMap[quest.seqNum] = callback
	} else {
		conn.mutex.Unlock()
		return nil, errors.New("Connection is broken.")
	}
	conn.mutex.Unlock()

	return quest.Raw()
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

	conn.writeChan <- binData
	conn.lastActiveTime = time.Now().Unix()

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

	conn.writeChan <- binData
	conn.lastActiveTime = time.Now().Unix()
	conn.mutex.Unlock()

	return nil
}

func (conn *tcpConnection) close() {

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if conn.connected {
		endpoint := conn.conn.RemoteAddr().String()
		conn.activeClosed = true
		err := conn.conn.Close()
		if err != nil {
			conn.logger.Printf("[ERROR] Close connection failed, err: %v", err)
			return
		}

		conn.ticker.Stop()
		conn.connected = false

		conn.mutex.Unlock()
		conn.closeSignChan <- true
		conn.cleanCallbackMap()
		if conn.onClosed != nil {
			var addr uintptr = uintptr(unsafe.Pointer(conn))
			go conn.onClosed(uint64(addr), endpoint)
		}
		conn.mutex.Lock()
	}
}
