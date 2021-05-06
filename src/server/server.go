package server

import (
	"context"
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	"github.com/clover-network/cloverscan-contract-registry/src/service"
	proto "github.com/clover-network/cloverscan-proto-contract"
	"github.com/go-pg/pg/v10"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
)

type GrpcServer struct {
	config   *config.Config
	listener net.Listener
	server   *grpc.Server
	database *pg.DB
	proto.ContractRegistryServer
	service *service.Service
}

func NewGrpcServer(
	config *config.Config, database *pg.DB, service *service.Service,
) *GrpcServer {
	return &GrpcServer{
		config:   config,
		database: database,
		service:  service,
	}
}

func (server *GrpcServer) Start() {
	listener, err := net.Listen("tcp", server.config.GrpcAddress)
	if err != nil {
		log.Fatalf("Could not listen to port in Start() %s: %v", server.config.GrpcAddress, err)
	}
	serverOps := []grpc.ServerOption{
		grpc.StreamInterceptor(grpcprometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpcprometheus.UnaryServerInterceptor),
	}
	server.server = grpc.NewServer(serverOps...)
	server.listener = listener
	grpcprometheus.EnableHandlingTimeHistogram()
	grpcprometheus.Register(server.server)
	proto.RegisterContractRegistryServer(server.server, server)
	go func() {
		log.Printf("extractor gRPC server is listening on address %s", server.config.GrpcAddress)
		if err = server.server.Serve(listener); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()
}

func (server *GrpcServer) GetContract(ctx context.Context, request *proto.GetContractRequest) (*proto.GetContractReply, error) {
	contract, err := server.service.Contract(request.BlockchainName, request.Address)
	if err != nil {
		log.Errorf("grpc error GetContract(%s %s): %s", request.BlockchainName, request.Address, err)
		return nil, status.Error(codes.Unknown, "failed to get contract")
	}
	return &proto.GetContractReply{Contract: contract.ToProto()}, nil
}
