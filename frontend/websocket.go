package frontend

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/jxwr/cc/streams"
	"golang.org/x/net/websocket"
)

func stateServer(stream *streams.Stream, ws *websocket.Conn) {
	callback := func(ns interface{}) bool {
		data, err := json.Marshal(ns)
		if err != nil {
			return false
		}
		// 如果浏览器关闭，或发送数据失败，则取消该Callback
		_, err = io.Copy(ws, bytes.NewReader(data))
		if err != nil {
			return false
		}
		return true
	}

	quitCh := stream.Sub(callback)
	<-quitCh
	log.Println("Websocket closed")
}

func nodeStateServer(ws *websocket.Conn) {
	stateServer(streams.NodeStateStream, ws)
}

func migrateStateServer(ws *websocket.Conn) {
	stateServer(streams.MigrateStateStream, ws)
}

func rebalanceStateServer(ws *websocket.Conn) {
	stateServer(streams.RebalanceStateStream, ws)
}

func RunWebsockServer(bindAddr string) {
	http.Handle("/node/state", websocket.Handler(nodeStateServer))
	http.Handle("/migrate/state", websocket.Handler(migrateStateServer))
	http.Handle("/rebalance/state", websocket.Handler(rebalanceStateServer))

	err := http.ListenAndServe(bindAddr, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
