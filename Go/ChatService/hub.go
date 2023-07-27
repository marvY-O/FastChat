package main

type Hub struct {
	clients map[*Client]bool
	register chan *Client
	unregister chan *Client
	users map[string]bool
}

func newHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		users: make(map[string]bool),
	}
}

func (h *Hub) contains(user_id string) bool{
	return h.users[user_id];
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.users[client.UserID] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				h.users[client.UserID] = false
				// close(client.send)
			}
		// case message := <-h.broadcast:
		// 	for client := range h.clients {
		// 		select {
		// 		case client.send <- message:
		// 		default:
		// 			close(client.send)
		// 			delete(h.clients, client)
		// 		}
		// 	}
		}
	}
}