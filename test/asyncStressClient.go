package main

import (
	"os"
	"fmt"
	"time"
	"runtime"
	"strconv"
	"sync/atomic"
	"../src/fpnn"
)

type Tester struct {
	endpoint	string
//---------------------//
	send		int64
	recv		int64
	sendError	int64
	recvError	int64
	timecost	int64
}

func QWriter() *fpnn.Quest {

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

func launch(endpoint string, connCount int, qps int) *Tester {

	pqps := qps / connCount
	if pqps == 0 {
		pqps = 1
	}

	remain := qps - pqps * connCount

	tester := &Tester{}
	tester.endpoint = endpoint

	for i := 0; i < connCount; i++ {
		go tester.test_worker(pqps)
	}

	if remain > 0 {
		go tester.test_worker(remain)
	}

	return tester
}

func (tester *Tester) test_worker(qps int) {

	usec := 1000 * 1000 / qps

	fmt.Println("-- qps:", qps, ", usec:", usec)

	client := fpnn.NewTCPClient(tester.endpoint)
	client.Connect()

	for {
		quest := QWriter()
		send_time := time.Now().UnixNano()
		err := client.SendQuestWithLambda(quest, func(answer *fpnn.Answer, errorCode int) {
			if errorCode == fpnn.FPNN_EC_OK {
				
				atomic.AddInt64(&tester.recv, 1)
				recv_time := time.Now().UnixNano()
				diff := recv_time - send_time
				atomic.AddInt64(&tester.timecost, diff)

			} else {
				atomic.AddInt64(&tester.recvError, 1)
				if errorCode == fpnn.FPNN_EC_CORE_TIMEOUT {
					fmt.Println("Timeouted occurred when recving.")
				} else {
					fmt.Println("error occurred when recving.")
				}
			}
		})
		if err == nil {
			atomic.AddInt64(&tester.send, 1)
		} else {
			atomic.AddInt64(&tester.sendError, 1)
		}

		sent_time := time.Now().UnixNano()
		diff := int64(usec) * 1000 - (sent_time - send_time)
		if diff > 0 {
			time.Sleep(time.Duration(diff) * time.Nanosecond)
		}
	}
}

func (tester *Tester) showStatistics() {

	sleepSeconds := 3 * time.Second

	var send int64
	var recv int64
	var sendError int64
	var recvError int64
	var timecost int64

	for {
		start := time.Now().UnixNano()

		time.Sleep(sleepSeconds)

		s := atomic.LoadInt64(&tester.send)
		r := atomic.LoadInt64(&tester.recv)
		se := atomic.LoadInt64(&tester.sendError)
		re := atomic.LoadInt64(&tester.recvError)
		tc := atomic.LoadInt64(&tester.timecost)

		ent:= time.Now().UnixNano()

		ds := s - send
		dr := r - recv
		dse := se - sendError
		dre := re - recvError
		dtc := tc - timecost

		send = s
		recv = r
		sendError = se
		recvError = re
		timecost = tc

		real_time := ent - start

		ds = ds * 1000 * 1000 * 1000 / real_time
		dr = dr * 1000 * 1000 * 1000 / real_time
		//dse = dse * 1000 * 1000 * 1000 / real_time
		//dre = dre * 1000 * 1000 * 1000 / real_time
		if dr > 0 {
			dtc = dtc / dr
		}

		fmt.Println("time interval: ", (real_time / (1000 * 1000)), " ms, send error: ", dse, ", recv error: ", dre)
		fmt.Println("[QPS] send: ", ds, ", recv: ", dr, ", per quest time cost: ", dtc/1000, " usec")
	}
}

func main() {

	if len(os.Args) != 5 {
		fmt.Println("Usage:", os.Args[0], "ip", "port", "connections", "total-qps")
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	connCount, _ := strconv.Atoi(os.Args[3])
	qps, _ := strconv.Atoi(os.Args[4])
	tester := launch(os.Args[1] + ":" + os.Args[2], connCount, qps)
	tester.showStatistics()
}