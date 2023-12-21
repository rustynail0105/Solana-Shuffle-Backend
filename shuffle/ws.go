package shuffle

import "github.com/solanashuffle/backend/stream"

func (r *Room) Subsribe() chan []byte {
	return r.stream.Subscribe()
}

func (r *Room) SubscribeWebsocket(w *stream.WebsocketWrapper) {
	r.stream.SubscribeWebsocket(w)
}

func (r *Room) UnsubscribeWebsocket(w *stream.WebsocketWrapper) {
	r.stream.UnsubscribeWebsocket(w)
}
