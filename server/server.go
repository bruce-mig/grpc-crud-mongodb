package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	pb "github.com/bruce-mig/grpc-crud-mongodb/proto"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/internal/status"
)

var (
	collection *mongo.Collection
	mongoURI   = "mongodb://localhost:27017"
)

const dbName = "grpc-crud"

type (
	GRPCBloggerServer struct {
		pb.BlogServiceServer
	}

	blogItem struct {
		ID       primitive.ObjectID `bson:"_id,omitempty"`
		AuthorID string             `bson:"author_id"`
		Content  string             `bson:"content"`
		Title    string             `bson:"title"`
	}
)

const Endpoint = "localhost:50051"

func main() {
	//if go code crashes, we get file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// connect to mongodb
	fmt.Println("Connecting to MongoDB")
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	collection = client.Database(dbName).Collection("blog")

	fmt.Println("Blog server started")
	grpcBloggerServer := &GRPCBloggerServer{}
	lis, err := net.Listen("tcp", Endpoint)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{}
	s := grpc.NewServer(opts...)
	pb.RegisterBlogServiceServer(s, grpcBloggerServer)

	go func() {
		fmt.Println("Starting Server...")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	//Wait for Ctl+C to exit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	//Block until a signal is received
	<-ch
	fmt.Println("Stopping the server")
	s.Stop()
	fmt.Println("Closing the listener")
	lis.Close()
	fmt.Println("Closing MongoDB Connection")
	client.Disconnect(context.TODO())
	fmt.Println("End of program")

}

func (*GRPCBloggerServer) CreateBlog(ctx context.Context, req *pb.CreateBlogRequest) (*pb.CreateBlogResponse, error) {
	fmt.Println("Create blog request")
	blog := req.GetBlog()

	data := &blogItem{
		AuthorID: blog.GetAuthorId(),
		Title:    blog.GetTitle(),
		Content:  blog.GetContent(),
	}

	res, err := collection.InsertOne(ctx, data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error:%v", err),
		)
	}

	objectID, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot convert to OID"),
		)
	}

	return &pb.CreateBlogResponse{
		Blog: &pb.Blog{
			Id:       objectID.Hex(),
			AuthorId: blog.GetAuthorId(),
			Title:    blog.GetTitle(),
			Content:  blog.Content,
		},
	}, nil
}
