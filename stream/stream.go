package stream

import (
	"encoding/json"
	"log"

	"github.com/gofiber/websocket/v2"
)

type Stream struct {
	stopCh    chan struct{}
	publishCh chan []byte
	subCh     chan chan []byte
	wsSubCh   chan *WebsocketWrapper
	unsubCh   chan chan []byte
	wsUnsubCh chan *WebsocketWrapper
}

func New() *Stream {
	return &Stream{
		stopCh:    make(chan struct{}),
		publishCh: make(chan []byte, 50),
		subCh:     make(chan chan []byte, 50),
		wsSubCh:   make(chan *WebsocketWrapper, 50),
		unsubCh:   make(chan chan []byte, 50),
		wsUnsubCh: make(chan *WebsocketWrapper, 50),
	}
}

func (s *Stream) Start() {
	subs := map[chan []byte]struct{}{}
	wsSubs := map[*WebsocketWrapper]struct{}{}
	for {
		select {
		case <-s.stopCh:
			return
		case msgCh := <-s.subCh:
			subs[msgCh] = struct{}{}
		case w := <-s.wsSubCh:
			wsSubs[w] = struct{}{}
		case msgCh := <-s.unsubCh:
			delete(subs, msgCh)
		case w := <-s.wsUnsubCh:
			delete(wsSubs, w)
		case msg := <-s.publishCh:
			for w := range wsSubs {
				if err := w.WriteSafe(websocket.TextMessage, msg); err != nil {
					log.Println("write error:", err)

					s.wsUnsubCh <- w
					w.WriteSafe(websocket.CloseMessage, []byte{})
					w.Conn.Close()
				}
			}
			for msgCh := range subs {
				select {
				case msgCh <- msg:
				default:
				}
			}
		}
	}
}

func (s *Stream) Subscribe() chan []byte {
	msgCh := make(chan []byte, 50)
	s.subCh <- msgCh
	return msgCh
}

func (s *Stream) SubscribeWebsocket(w *WebsocketWrapper) {
	s.wsSubCh <- w
}

func (s *Stream) Unsubscribe(msgCh chan []byte) {
	s.unsubCh <- msgCh
}

func (s *Stream) UnsubscribeWebsocket(w *WebsocketWrapper) {
	s.wsUnsubCh <- w
}

func (s *Stream) Publish(msg []byte) {
	s.publishCh <- msg
}

func (s *Stream) PublishJSON(msg interface{}) {
	j, err := json.Marshal(msg)
	if err != nil {
		return
	}

	s.Publish(j)
}
