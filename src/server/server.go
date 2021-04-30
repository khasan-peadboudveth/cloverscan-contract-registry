package server

import (
	"context"
	"github.com/clover-network/cloverscan-contract-registry/src/blockchain/eth"
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	"github.com/clover-network/cloverscan-contract-registry/src/entity"
	proto "github.com/clover-network/cloverscan-proto-contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-pg/pg/v10"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"strings"
)

type GrpcServer struct {
	config   *config.Config
	listener net.Listener
	server   *grpc.Server
	database *pg.DB
	proto.ExtractorServer
}

func NewGrpcServer(
	config *config.Config, database *pg.DB,
) *GrpcServer {
	return &GrpcServer{
		config:   config,
		database: database,
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
	proto.RegisterExtractorServer(server.server, server)
	go func() {
		log.Printf("extractor gRPC server is listening on address %s", server.config.GrpcAddress)
		if err = server.server.Serve(listener); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()
}

func (server *GrpcServer) GetLatestBalance(ctx context.Context, request *proto.GetLatestBalanceRequest) (*proto.GetLatestBalanceReply, error) {
	result := &entity.ExtractorConfig{}
	if err := server.database.Model(result).Order("id asc").Limit(1).Where("upper(name) = ?", strings.ToUpper(request.BlockchainName)).Select(); err != nil {
		if err == pg.ErrNoRows {
			return &proto.GetLatestBalanceReply{}, status.Errorf(codes.InvalidArgument, "no such blockchain")
		}
		log.Errorf("error occured while calling GetLatestBalance: %s", err)
		return &proto.GetLatestBalanceReply{}, status.Errorf(codes.Internal, "unable to get blockchain service")
	}
	if result.ChainType != "ETH" {
		return &proto.GetLatestBalanceReply{}, status.Errorf(codes.InvalidArgument, "method is not supported for this blockchain")
	}
	service := eth.NewService(result.Name, result.Nodes[0])
	err := service.Start()
	if err != nil {
		log.Errorf("error occured while calling GetLatestBalance: %s", err)
		return &proto.GetLatestBalanceReply{}, status.Errorf(codes.Internal, "failed to connect to blockchain")
	}
	if !common.IsHexAddress(request.Address) {
		return &proto.GetLatestBalanceReply{}, status.Errorf(codes.InvalidArgument, "address is invalid")
	}
	balance, err := service.LatestBalance(request.Address)
	if err != nil {
		log.Errorf("error occured while calling GetLatestBalance: %s", err)
		return &proto.GetLatestBalanceReply{}, status.Errorf(codes.Internal, "failed to get balance from blockchain")
	}
	return &proto.GetLatestBalanceReply{Value: balance}, nil
}

func (server *GrpcServer) GetTransactionByHash(ctx context.Context, request *proto.GetTransactionByHashRequest) (*proto.GetTransactionByHashReply, error) {
	result := &entity.ExtractorConfig{}
	if err := server.database.Model(result).Order("id asc").Limit(1).Where("upper(name) = ?", strings.ToUpper(request.BlockchainName)).Select(); err != nil {
		if err == pg.ErrNoRows {
			return &proto.GetTransactionByHashReply{}, status.Errorf(codes.InvalidArgument, "no such blockchain")
		}
		log.Errorf("error occured while calling GetTransactionByHash during config lookup: %s", err)
		return &proto.GetTransactionByHashReply{}, status.Errorf(codes.Internal, "unable to get blockchain service")
	}
	if result.ChainType != "ETH" {
		return &proto.GetTransactionByHashReply{}, status.Errorf(codes.InvalidArgument, "method is not supported for this blockchain")
	}
	service := eth.NewService(result.Name, result.Nodes[0])
	err := service.Start()
	if err != nil {
		log.Errorf("error occured while calling GetTransactionByHash during service start: %s", err)
		return &proto.GetTransactionByHashReply{}, status.Errorf(codes.Internal, "failed to get initialize connector")
	}
	details, err := service.TransactionByHash(request.TransactionHash)
	if err != nil {
		log.Errorf("error occured while calling GetTransactionByHash during request to blockchain: %s", err)
		return &proto.GetTransactionByHashReply{}, status.Errorf(codes.Internal, "failed to get transaction from blockchain")
	}
	return &proto.GetTransactionByHashReply{Details: details}, nil
}

func (server *GrpcServer) GetBlockByHeight(ctx context.Context, request *proto.GetBlockByHeightRequest) (*proto.GetBlockByHeightReply, error) {
	result := &entity.ExtractorConfig{}
	if err := server.database.Model(result).Order("id asc").Limit(1).Where("upper(name) = ?", strings.ToUpper(request.BlockchainName)).Select(); err != nil {
		if err == pg.ErrNoRows {
			return &proto.GetBlockByHeightReply{}, status.Errorf(codes.InvalidArgument, "no such blockchain")
		}
		log.Errorf("error occured while calling GetBlockByHeight during config lookup: %s", err)
		return &proto.GetBlockByHeightReply{}, status.Errorf(codes.Internal, "unable to get blockchain service")
	}
	if result.ChainType != "ETH" {
		return &proto.GetBlockByHeightReply{}, status.Errorf(codes.InvalidArgument, "method is not supported for this blockchain")
	}
	service := eth.NewService(result.Name, result.Nodes[0])
	err := service.Start()
	if err != nil {
		log.Errorf("error occured while calling GetBlockByHeight during service start: %s", err)
		return &proto.GetBlockByHeightReply{}, status.Errorf(codes.Internal, "failed to get initialize connector")
	}
	details, err := service.ProcessBlock(int64(request.BlockHeight))
	if err != nil {
		log.Errorf("error occured while calling GetBlockByHeight during request to blockchain: %s", err)
		return &proto.GetBlockByHeightReply{}, status.Errorf(codes.Internal, "failed to get transaction from blockchain")
	}
	return &proto.GetBlockByHeightReply{Details: details}, nil
}

func (server *GrpcServer) Stop() {
	server.server.Stop()
	_ = server.listener.Close()
}
