package Server

import (
	"Log"
	"fmt"
	"sync"
	"Server/Networking"
)

type Server struct {
	activeThreads sync.WaitGroup
	networkController *Networking.NetworkController
	}
func NewServer() *Server {
    return &Server{/*X: 5*/}
}
func (s *Server) Init() {
	log.Println("Starting Server");
	
	//Now initialize the NetworkingController
	s.networkController = Networking.NewNetworkController()
	
	mainLoop: //Label of loop, useful for only breaking out
	for {
		log.Println("Enter a course of action(1 = Save and Exit): ")
		var i int
    	_, err := fmt.Scanf("%d", &i)
    	if (err == nil) {
    		switch i {
    			case 1: {
    				log.Println("Server is shutting down")
    				break mainLoop
    			}
    		}
    	}
	}
	//Wait for the other threads to terminate before exiting
	s.activeThreads.Wait()
    log.Println("Server has stopped")
}