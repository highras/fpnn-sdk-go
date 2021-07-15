package fpnn

import (
	"log"
	"os"
	"time"
)

type config struct {
	logger            *log.Logger
	questTimeout      time.Duration
	connectTimeout    time.Duration
	netChanBufferSize int
	maxPayloadSize    int
	pingInterval      time.Duration
	maxPingRetryCount int
}

func (conf *config) SetLogger(logger *log.Logger) {
	conf.logger = logger
}

func (conf *config) SetQuestTimeout(timeout time.Duration) {
	conf.questTimeout = timeout
}

func (conf *config) SetConnectTimeout(timeout time.Duration) {
	conf.connectTimeout = timeout
}

func (conf *config) SetNetChannelBufferSize(size int) {
	conf.netChanBufferSize = size
}

func (conf *config) SetMaxPayloadSize(size int) {
	conf.maxPayloadSize = size
}

var Config = &config{
	log.New(os.Stdout, "[FPNN Go SDK] ", log.LstdFlags|log.Lshortfile),
	5 * time.Second,
	5 * time.Second,
	5,
	200 * 1024 * 1024,
	10 * time.Second,
	2,
}
