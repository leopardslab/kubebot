package main

import (
	"fmt"
	"github.com/leopardslab/kubebot/kube_Executor/api/handlers"
	kubebot_executor "github.com/leopardslab/kubebot/kube_Executor/api/proto"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 5001))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	kubebot_executor.RegisterExecutorRoutesServer(s, handlers.NewExecutorServer())
	log.Printf("server listening at %v", lis.Addr())

	err = s.Serve(lis)
	if err != nil {

		log.Fatalf("Error : %s", err.Error())
	}

}
