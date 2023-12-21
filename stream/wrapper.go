package stream

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

type WebsocketWrapper struct {
	Conn *websocket.Conn
	mu   *sync.Mutex
}

func NewWrapper(c *websocket.Conn) *WebsocketWrapper {
	return &WebsocketWrapper{
		Conn: c,
		mu:   &sync.Mutex{},
	}
}

func (w *WebsocketWrapper) WriteSafe(mt int, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Conn.WriteMessage(mt, data)
}

func (w *WebsocketWrapper) WriteSafeJSON(data interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Conn.WriteJSON(data)
}
