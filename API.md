# FPNN Go SDK API Docs

# Package fpnn

	import "github.com/highras/fpnn-sdk-go/src/fpnn"

fpnn 包提供go连接和访问 FPNN 技术生态的能力，可以实现加密通讯、server push、和连接事件。

# Index

[TOC]

## Constants

	const SDKVersion = "1.1.1"

### FPNN Framework Standard Error Code

Please refer: [errorCodes.go](src/fpnn/errorCodes.go)

## Variables

	var Config

### func (conf *config) SetLogger(logger *fpnn.Logger)

	func (conf *config) SetLogger(logger *fpnn.Logger)

配置日志路由。
Logger是一个interface，需实现Println(...any)和Printf(string, ...any)两个方法。log.Logger直接作为fpnn.Logger使用。
如果没有为 [TCPClient][tcpClient] 实例单独配置日志路由，则所有 [TCPClient][tcpClient] 均采用该配置。


### func (conf *config) SetQuestTimeout(timeout time.Duration)

	func (conf *config) SetQuestTimeout(timeout time.Duration)

配置请求超时。  
未配置时，默认为 5 秒。  
如果没有为 [TCPClient][tcpClient] 实例单独配置请求超时，则所有 [TCPClient][tcpClient] 均采用该配置。


### func (conf *config) SetConnectTimeout(timeout time.Duration)

	func (conf *config) SetConnectTimeout(timeout time.Duration)

配置连接超时。  
未配置时，默认为 5 秒。  
如果没有为 [TCPClient][tcpClient] 实例单独配置连接超时，则所有 [TCPClient][tcpClient] 均采用该配置。

### func (conf *config) SetNetChannelBufferSize(size int)

	func (conf *config) SetNetChannelBufferSize(size int)

配置 [TCPClient][tcpClient] 发送队列长度。
所有 [TCPClient][tcpClient] 均采用该配置。

### func (conf *config) SetMaxPayloadSize(size int)

	func (conf *config) SetMaxPayloadSize(size int)

配置 FPNN 包最大长度。如果超过该长度，则拒绝接收，并关闭连接。
所有 [TCPClient][tcpClient] 均采用该配置。

## type AnswerCallback

	type AnswerCallback interface {
		OnAnswer(answer *Answer)
		OnException(answer *Answer, errorCode int)
	}

请求响应的回调接口。

* 当请求被正常响应时，将调用接口的 `func OnAnswer(answer *Answer)` 函数；
* 当请求失败，或者请求异常时，将调用接口的 `func OnException(answer *Answer, errorCode int)` 函数。

	+ 当请求失败时，**answer** 为 nil
	+ 当请求异常时，**answer** 为 FPNN 标准异常响应，`errorCode` 为标准异常响应中包含的错误代码。


## type QuestProcessor

	type QuestProcessor interface {
		Process(method string) func(*Quest) (*Answer, error)
	}

服务器事件处理函数的路由函数。

参数：

+ **method**：服务器事件名称/服务器请求的方法名/服务器请求客户端的接口名称

返回值：

	func(*Quest) (*Answer, error)

对于 twoWay 请求，需要返回 *Answer 对象；  
对于 oneWay 请求，返回的 *Answer 必须为 nil。

参考：

+ [oneWayDuplex.go](examples/oneWayDuplex.go)
+ [twoWayDuplex.go](examples/twoWayDuplex.go)
+ [rtmServerQuestProcessor.go](https://github.com/highras/rtm-server-sdk-go/blob/master/src/rtm/rtmServerQuestProcessor.go)


## type TCPClient

	type TCPClient struct {
		//-- same hidden fields
	}

FPNN 客户端。

### func NewTCPClient(endpoint string) *TCPClient

	func NewTCPClient(endpoint string) *TCPClient

创建 FPNN TCP 客户端。  
endpoint 格式为：`"hostname/ip" + ":" + "port"`  
endpoint 例子：`endpoint := "localhost:8000"`

### func (client *TCPClient) SetAutoReconnect(autoReconnect bool)

	func (client *TCPClient) SetAutoReconnect(autoReconnect bool)

配置 FPNN TCP Client 在断线，但有消息发送的时候，是否自动重连。

未配置时，默认行为是**自动重连**。

### func (client *TCPClient) SetKeepAlive(keepAlive bool)

	func (client *TCPClient) SetKeepAlive(keepAlive bool)

设置是否开启连接保活，开启保活后默认10s没有收到数据会发送保活请求，若连续2次保活请求都没有收到响应，将会关闭连接

默认为**连接不保活**

### func (client *TCPClient) SetKeepAliveTimeoutSecond(second time.Duration)

	func (client *TCPClient) SetKeepAliveTimeoutSecond(second time.Duration)

设置保活请求的超时时间，**单位为秒**，设置完后将**开启连接保活**
未配置时，默认采用 Config 的请求超时参数

### func (client *TCPClient) SetKeepAliveIntervalSecond(second time.Duration)

	func (client *TCPClient) SetKeepAliveIntervalSecond(second time.Duration)

设置多久没有收到数据将发送保活请求的时间间隔，**单位为秒**，设置完后将**开启连接保活**
未配置时，默认连接保活的间隔为10s

### func (client *TCPClient) SetKeepAliveMaxPingRetryCount(count int)

	func (client *TCPClient) SetKeepAliveMaxPingRetryCount(count int)

设置最大连续保活请求的个数，设置完后将**开启连接保活**
未配置时，默认最大连续保活请求个数为2，即如果连续2个保活请求没有收到响应，将关闭连接

### func (client *TCPClient) SetConnectTimeOut(timeout time.Duration)

	func (client *TCPClient) SetConnectTimeOut(timeout time.Duration)

配置 FPNN TCP Client 的连接超时。  
未配置时，默认采用 Config 的连接超时参数。

### func (client *TCPClient) SetQuestTimeOut(timeout time.Duration)

	func (client *TCPClient) SetQuestTimeOut(timeout time.Duration)

配置 FPNN TCP Client 的请求超时。  
未配置时，默认采用 Config 的请求超时参数。

### func (client *TCPClient) SetQuestProcessor(questProcessor QuestProcessor)

	func (client *TCPClient) SetQuestProcessor(questProcessor QuestProcessor)

配置 Duplex 模式（Server Push）下，服务器推送消息的请求接口的处理函数的路由函数。  
具体参考：[QuestProcessor](#type-QuestProcessor)

### func (client *TCPClient) SetOnConnectedCallback(onConnected func(connId uint64, endpoint string, connected bool))

	func (client *TCPClient) SetOnConnectedCallback(onConnected func(connId uint64, endpoint string, connected bool))

配置连接建立事件的回调函数。

### func (client *TCPClient) SetOnClosedCallback(onClosed func(connId uint64, endpoint string))

	func (client *TCPClient) SetOnClosedCallback(onClosed func(connId uint64, endpoint string))

配置连接断开事件的回调函数。

### func (client *TCPClient) SetLogger(logger *log.Logger)

	func (client *TCPClient) SetLogger(logger *log.Logger)

配置 FPNN TCP Client 的日志路由。  
未配置时，默认采用 Config 的日志路由。

### func (client *TCPClient) EnableEncryptor(rest ... interface{}) (err error)

	func (client *TCPClient) EnableEncryptor(rest ... interface{}) (err error)

配置使用加密链接。

可接受的参数为：

+ `pemKeyPath string`

	服务器公钥文件路径。PEM 格式。与 pemKeyData 参数互斥。

+ `pemKeyData []byte`

	服务器公钥文件内容。PEM 格式。与 pemKeyPath 参数互斥。

+ `reinforce bool`

	true 采用 256 位密钥加密，false 采用 128 位密钥加密。  
	默认为 true

### func (client *TCPClient) IsConnected() bool

	func (client *TCPClient) IsConnected() bool

判断 FPNN TCP Client 是否已建立连接。

### func (client *TCPClient) Endpoint() string

	func (client *TCPClient) Endpoint() string

获取 FPNN TCP Client 连接/目标地址。

### func (client *TCPClient) Connect() bool

	func (client *TCPClient) Connect() bool

连接目标服务器。(FPNN 风格接口)

### func (client *TCPClient) Dial() bool

	func (client *TCPClient) Dial() bool

连接目标服务器。(Go 风格接口)

### func (client *TCPClient) SendQuest(quest *Quest, timeout ... time.Duration) (*Answer, error)

	func (client *TCPClient) SendQuest(quest *Quest, timeout ... time.Duration) (*Answer, error) 

发送 oneWay 请求，或着**同步**发送 twoWay 请求。

发送 oneWay 请求时，若无异常，SendQuest() 将立刻返回，并且 *Answer 为 nil。

发送 twoWay 请求时，若无异常，SendQuest() 将进入**等待**，直到接收到服务器应答，或请求超时后才返回。

Quest 请参考：[Quest][quest]

Answer 请参考：[Answer][answer]

使用方式：

	client.SenfQuest(quest)
	client.SenfQuest(quest, 5 * time.Second)

缺少 **timeout** 参数时，将采用 FPNN TCP Client 实例的配置。  
若 FPNN TCP Client 实例未配置，将采用 Config 的相应配置。


### func (client *TCPClient) SendQuestWithCallback(quest *Quest, callback AnswerCallback, timeout ... time.Duration) error

	func (client *TCPClient) SendQuestWithCallback(quest *Quest, callback AnswerCallback, timeout ... time.Duration) error

**异步**发送 twoWay 请求。

Quest 请参考：[Quest][quest]

Answer 请参考：[Answer][answer]

AnswerCallback 请参考：[AnswerCallback](#type-AnswerCallback)

使用方式：

	client.SendQuestWithCallback(quest, callback)
	client.SendQuestWithCallback(quest, callback, 5 * time.Second)

缺少 **timeout** 参数时，将采用 FPNN TCP Client 实例的配置。  
若 FPNN TCP Client 实例未配置，将采用 Config 的相应配置。


### func (client *TCPClient) SendQuestWithLambda(quest *Quest, callback func(answer *Answer, errorCode int), timeout ... time.Duration) error

	func (client *TCPClient) SendQuestWithLambda(quest *Quest, callback func(answer *Answer, errorCode int), timeout ... time.Duration) error

**异步**发送 twoWay 请求。

Quest 请参考：[Quest][quest]

Answer 请参考：[Answer][answer]

使用方式：

	client.SendQuestWithLambda(quest, func(answer *Answer, errorCode int){ ... })
	client.SendQuestWithLambda(quest, func(answer *Answer, errorCode int){ ... }, 5 * time.Second)

缺少 **timeout** 参数时，将采用 FPNN TCP Client 实例的配置。  
若 FPNN TCP Client 实例未配置，将采用 Config 的相应配置。

### func (client *TCPClient) Close()

	func (client *TCPClient) Close()

关闭当前连接。


## type Quest

	type Quest struct {
		//-- same hidden fields
	}

FPNN 请求对象。

Quest 数据接口请参见：[Payload][payload]

### func NewQuest(method string) *Quest

	func NewQuest(method string) *Quest

创建 twoWay 请求对象。

### func NewQuestWithPayload(method string, payload *Payload) *Quest

	func NewQuestWithPayload(method string, payload *Payload) *Quest

使用 payload 创建 twoWay 请求对象。

### func NewOneWayQuest(method string) *Quest

	func NewOneWayQuest(method string) *Quest

创建 oneWay 请求对象。

###	func NewOneWayQuestWithPayload(method string, payload *Payload) *Quest

	func NewOneWayQuestWithPayload(method string, payload *Payload) *Quest

使用 payload 创建 oneWay 请求对象。

### func (quest *Quest) IsOneWay() bool

	func (quest *Quest) IsOneWay() bool

是否是 oneWay 请求。

### func (quest *Quest) IsTwoWay() bool

	func (quest *Quest) IsTwoWay() bool

是否是 twoWay 请求。

### func (quest *Quest) IsMsgPack() bool

	func (quest *Quest) IsMsgPack() bool

原始数据是否是 msgPack 编码。

### func (quest *Quest) IsJson() bool

	func (quest *Quest) IsJson() bool

原始数据是否是 JSON 编码。

### func (quest *Quest) SeqNum() uint32

	func (quest *Quest) SeqNum() uint32

twoWay 请求的序号。**当且仅当** SendQuest() 之后才被设置。

### func (quest *Quest) Method() string

	func (quest *Quest) Method() string

Quest 请求的接口名称。

### func (quest *Quest) Raw() ([]byte, error)

	func (quest *Quest) Raw() ([]byte, error)

串行化请求对象。



## type Answer

	type Answer struct {
		//-- same hidden fields
	}

FPNN 服务器应答对象。

Answer 数据接口请参见：[Payload][payload]

### func NewAnswer(quest *Quest) *Answer

	func NewAnswer(quest *Quest) *Answer

通过 Quest 请求对象，创建对应的应答对象。

### func NewErrorAnswer(quest *Quest, code int, ex string) *Answer

	func NewErrorAnswer(quest *Quest, code int, ex string) *Answer

通过 Quest 请求对象，创建对应的 FPNN 标准异常应答对象。

### func newErrorAnswerWitSeqNum(seqNum uint32, code int, ex string) *Answer

	func newErrorAnswerWitSeqNum(seqNum uint32, code int, ex string) *Answer

在没有 Quest 对象的情况下，使用 quest 的序号，创建对应的 FPNN 标准异常应答对象。

### func (answer *Answer) SeqNum() uint32

	func (answer *Answer) SeqNum() uint32

Answer 对象的序号，同时也是对应 Quest 对象的序号。

### func (answer *Answer) Status() uint8

	func (answer *Answer) Status() uint8

Answer 对象类别。  
如果为 0，则为普通应答对象，否则为 FPNN 标准异常应答对象。

### func (answer *Answer) IsException() bool

	func (answer *Answer) IsException() bool

判断是否为 FPNN 标准异常应答对象。

### func (answer *Answer) IsMsgPack() bool

	func (answer *Answer) IsMsgPack() bool

原始数据是否是 msgPack 编码。

### func (answer *Answer) IsJson() bool

	func (answer *Answer) IsJson() bool

原始数据是否是 JSON 编码。

### func (answer *Answer) Raw() ([]byte, error)

	func (answer *Answer) Raw() ([]byte, error)

串行化应答对象。


## type Payload

	type TCPClient struct {
		//-- same hidden fields
	}

FPNN 消息数据。Quest & Answer 的核心对象。

### func NewPayload() *Payload

	func NewPayload() *Payload

创建 Payload 实例。

### func (payload *Payload) GetInt64(key string) (int64, bool)

	func (payload *Payload) GetInt64(key string) (int64, bool)

获取 int64 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetInt32(key string) (int32, bool)

	func (payload *Payload) GetInt32(key string) (int32, bool)

获取 int32 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetInt16(key string) (int16, bool)

	func (payload *Payload) GetInt16(key string) (int16, bool)

获取 int16 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetInt8(key string) (int8, bool)

	func (payload *Payload) GetInt8(key string) (int8, bool)

获取 int8 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetInt(key string) (int, bool)

	func (payload *Payload) GetInt(key string) (int, bool)

获取 int 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetUint64(key string) (uint64, bool)

	func (payload *Payload) GetUint64(key string) (uint64, bool)

获取 uint64 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetUint32(key string) (uint32, bool)

	func (payload *Payload) GetUint32(key string) (uint32, bool)

获取 uint32 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetUint16(key string) (uint16, bool)

	func (payload *Payload) GetUint16(key string) (uint16, bool)

获取 uint16 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetUint8(key string) (uint8, bool)

	func (payload *Payload) GetUint8(key string) (uint8, bool)

获取 uint8 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetUint(key string) (uint, bool)

	func (payload *Payload) GetUint(key string) (uint, bool)

获取 uint 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetString(key string) (value string, ok bool)

	func (payload *Payload) GetString(key string) (value string, ok bool)

获取 string 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 ""。

### func (payload *Payload) GetBool(key string) (value bool, ok bool)

	func (payload *Payload) GetBool(key string) (value bool, ok bool)

获取 bool 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 false。

### func (payload *Payload) GetFloat32(key string) (float32, bool)

	func (payload *Payload) GetFloat32(key string) (float32, bool)

获取 float32 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetFloat64(key string) (float64, bool)

	func (payload *Payload) GetFloat64(key string) (float64, bool)

获取 float64 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 零值。

### func (payload *Payload) GetSlice(key string) (value []interface{}, ok bool)

	func (payload *Payload) GetSlice(key string) (value []interface{}, ok bool)

获取 切片类型/数组类型 数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 nil。

### func (payload *Payload) GetMap(key string) (value map[interface{}]interface{}, ok bool)

	func (payload *Payload) GetMap(key string) (value map[interface{}]interface{}, ok bool)

获取 map 类型数据，以及存在状态。  
如果数据不存在，或者类型不匹配，返回 nil。

### func (payload *Payload) GetDict(key string) (value *Payload, ok bool)

	func (payload *Payload) GetDict(key string) (value *Payload, ok bool)

获取 map 类型数据，并转化为 Payload 类型，并获取存在状态。  
如果数据不存在，或者类型不匹配，返回 nil。

### func (payload *Payload) WantInt64(key string) int64

	func (payload *Payload) WantInt64(key string) int64

获取 int64 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantInt32(key string) int32

	func (payload *Payload) WantInt32(key string) int32

获取 int32 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantInt16(key string) int16

	func (payload *Payload) WantInt16(key string) int16

获取 int16 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantInt8(key string) int8

	func (payload *Payload) WantInt8(key string) int8

获取 int8 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantInt(key string) int

	func (payload *Payload) WantInt(key string) int

获取 int 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantUint64(key string) uint64

	func (payload *Payload) WantUint64(key string) uint64

获取 uint64 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantUint32(key string) uint32

	func (payload *Payload) WantUint32(key string) uint32

获取 uint32 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantUint16(key string) uint16

	func (payload *Payload) WantUint16(key string) uint16

获取 uint16 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantUint8(key string) uint8

	func (payload *Payload) WantUint8(key string) uint8

获取 uint8 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantUint(key string) uint

	func (payload *Payload) WantUint(key string) uint

获取 uint 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantString(key string) string

	func (payload *Payload) WantString(key string) string

获取 string 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantBool(key string) bool

	func (payload *Payload) WantBool(key string) bool

获取 bool 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantFloat32(key string) float32

	func (payload *Payload) WantFloat32(key string) float32

获取 float32 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantFloat64(key string) float64

	func (payload *Payload) WantFloat64(key string) float64

获取 float64 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantSlice(key string) []interface{}

	func (payload *Payload) WantSlice(key string) []interface{}

获取 切片类型/数组类型 数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantMap(key string) map[interface{}]interface{}

	func (payload *Payload) WantMap(key string) map[interface{}]interface{}

获取 map 类型数据。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) WantDict(key string) *Payload

	func (payload *Payload) WantDict(key string) *Payload

获取 map 类型数据，并转化为 Payload 类型。  
如果数据不存在，或者类型不匹配，触发 panic。

### func (payload *Payload) Param(key string, value interface{})

	func (payload *Payload) Param(key string, value interface{})

插入数据。

### func (payload *Payload) Get(key string) (value interface{}, ok bool)

	func (payload *Payload) Get(key string) (value interface{}, ok bool)

获取数据。

### func (payload *Payload) Exist(key string) bool

	func (payload *Payload) Exist(key string) bool

检查数据是否存在。


[tcpClient]: #type-TCPClient
[quest]: #type-Quest
[answer]: #type-Answer
[payload]: #type-Payload

