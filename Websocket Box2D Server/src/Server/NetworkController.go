package Server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"code.google.com/p/go.net/websocket"
)

type NetworkController struct {

	}
func NewNetworkController() *NetworkController {
    return &NetworkController{/*X: 5*/}
}


/************** SERVER *****************/
type NetworkServer struct {
	messages  []*Message
	clients   map[int]*Client
	addClientChannel     chan *Client
	removeClientChannel     chan *Client
	sendMsgToAllChannel chan *Message
	shutdownChannel    chan bool
	errChannel     chan error
}

// Create new chat server.
func NewNetworkServer(pattern string) *NetworkServer {
	messages := []*Message{}
	clients := make(map[int]*Client)
	addClientChannel := make(chan *Client)
	removeClientChannel := make(chan *Client)
	sendMsgToAllChannel := make(chan *Message)
	shutdownChannel := make(chan bool)
	errChannel := make(chan error)

	return &NetworkServer{
		messages,
		clients,
		addClientChannel,
		removeClientChannel,
		sendMsgToAllChannel,
		shutdownChannel,
		errChannel,
	}
}

func (s *NetworkServer) addClient(c *Client) {
	s.addClientChannel <- c
}

func (s *NetworkServer) removeClient(c *Client) {
	s.removeClientChannel <- c
}

func (s *NetworkServer) SendMsgToAll(msg *Message) {
	s.sendMsgToAllChannel <- msg
}

func (s *NetworkServer) ShutdownServer() {
	s.shutdownChannel <- true
}

func (s *NetworkServer) Err(err error) {
	s.errChannel <- err
}

func (s *NetworkServer) sendPastMessages(c *Client) {
	for _, msg := range s.messages {
		c.SendMessage(msg)
	}
}

func (s *NetworkServer) sendAll(msg *Message) {
	for _, c := range s.clients {
		c.SendMessage(msg)
	}
}

// Listen and serve.
// It serves client connection and broadcast request.
func (s *NetworkServer) Listen() {

	log.Println("Listening server...")

	//This is a function that runs on a separate thread
	onConnected := func(ws *websocket.Conn) {
		defer func() {
			err := ws.Close()
			if err != nil {
				s.errChannel <- err
			}
		}()

		client := NewClient(ws, s)
		s.addClient(client)
		client.Listen()
	}
	http.Handle("/", websocket.Handler(onConnected))

	for {
		select {

		// Add new a client
		case c := <-s.addClientChannel:
			log.Println("Added new client")
			s.clients[c.id] = c
			log.Println("Now", len(s.clients), "clients connected.")
			s.sendPastMessages(c)

		// del a client
		case c := <-s.removeClientChannel:
			log.Println("Delete client")
			delete(s.clients, c.id)

		// broadcast message for all clients
		case msg := <-s.sendMsgToAllChannel:
			log.Println("Send all:", msg)
			s.messages = append(s.messages, msg)
			s.sendAll(msg)

		case err := <-s.errChannel:
			log.Println("Error:", err.Error())

		case <-s.shutdownChannel:
			return
		}
	}
}


/************** MESSAGE *****************/
type Message struct {
	Author string `json:"author"`
	Body   string `json:"body"`
}

func (self *Message) String() string {
	return self.Author + " says " + self.Body
}


/*************** CLIENT *****************/
const channelBufSize = 100

var maxId int = 0

// Chat client.
type Client struct {
	id     int
	webConnection     *websocket.Conn
	networkServer *NetworkServer
	messageChannel     chan *Message
	closeChannel chan bool
}

// Create new chat client.
func NewClient(webConnection *websocket.Conn, networkServer *NetworkServer) *Client {
	if webConnection == nil {
		panic("ws cannot be nil")
	}

	if networkServer == nil {
		panic("server cannot be nil")
	}

	maxId++
	messageChannel := make(chan *Message, channelBufSize)
	closeChannel := make(chan bool)

	return &Client{maxId, webConnection, networkServer, messageChannel, closeChannel}
}

func (c *Client) GetWebConnection() *websocket.Conn {
	return c.webConnection
}

func (c *Client) SendMessage(msg *Message) {
	select {
	case c.messageChannel <- msg:
	default:
		c.networkServer.removeClient(c)
		err := fmt.Errorf("client %d is disconnected.", c.id)
		c.networkServer.Err(err)
	}
}

func (c *Client) CloseClient() {
	c.closeChannel <- true
}

// Listen Write and Read request via chanel
func (c *Client) Listen() {
	go c.listenWrite()
	c.listenRead()
}

// Listen write request via chanel
func (c *Client) listenWrite() {
	log.Println("Listening write to client")
	for {
		select {

		//Got message from message channel
		case msg := <-c.messageChannel:
			log.Println("Send:", msg)
			websocket.JSON.Send(c.webConnection, msg)

		//Got close client from close channel
		case <-c.closeChannel:
			c.networkServer.removeClient(c)
			//Set again for the listenRead to catch
			c.closeChannel <- true
			return
		}
	}
}

// Listen read request via chanel
func (c *Client) listenRead() {
	log.Println("Listening read from client")
	for {
		select {

		//Got a close client request from close channel
		case <-c.closeChannel:
			c.networkServer.removeClient(c)
			//Set again for the listenWrite
			c.closeChannel <- true
			return

		//Get message data from client
		default:
			var msg Message
			err := websocket.JSON.Receive(c.webConnection, &msg)
			if err == io.EOF {
				c.closeChannel <- true
			} else if err != nil {
				c.networkServer.Err(err)
			} else {
				c.networkServer.SendMsgToAll(&msg)
			}
		}
	}
}