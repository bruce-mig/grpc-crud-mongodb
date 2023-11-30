package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	pb "github.com/bruce-mig/grpc-crud-mongodb/proto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	collection *mongo.Collection
	mongoURI   = "mongodb://172.20.128.1:27017"
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
	fmt.Println("Gracefully Stopping the server")
	s.GracefulStop()
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
			fmt.Sprintln("Cannot convert to OID"),
		)
	}

	return &pb.CreateBlogResponse{
		Blog: &pb.Blog{
			Id:       objectID.Hex(),
			AuthorId: blog.GetAuthorId(),
			Title:    blog.GetTitle(),
			Content:  blog.GetContent(),
		},
	}, nil
}

func (*GRPCBloggerServer) ReadBlog(ctx context.Context, req *pb.ReadBlogRequest) (*pb.ReadBlogResponse, error) {
	fmt.Println("Read blog request")

	blogID := req.GetBlogId()
	oID, err := primitive.ObjectIDFromHex(blogID)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintln("Cannot parse ID."),
		)
	}

	//create an empty struct
	data := &blogItem{}

	filter := bson.D{{Key: "_id", Value: oID}}
	res := collection.FindOne(context.Background(), filter)

	if err = res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog with specified ID: %v.\n", err),
		)
	}

	return &pb.ReadBlogResponse{
		Blog: dataToBlogPb(data),
	}, nil

}

func dataToBlogPb(data *blogItem) *pb.Blog {
	return &pb.Blog{
		Id:       data.ID.Hex(),
		AuthorId: data.AuthorID,
		Content:  data.Content,
		Title:    data.Title,
	}
}

func (*GRPCBloggerServer) UpdateBlog(ctx context.Context, req *pb.UpdateBlogRequest) (*pb.UpdateBlogResponse, error) {
	fmt.Println("Update blog request")
	blog := req.GetBlog()
	oID, err := primitive.ObjectIDFromHex(blog.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintln("Cannot parse ID."),
		)
	}

	//create an empty struct
	data := &blogItem{}
	filter := bson.D{{Key: "_id", Value: oID}}

	res := collection.FindOne(context.Background(), filter)
	if err = res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog with specified ID: %v.\n", err),
		)
	}

	// update internal struct
	data.AuthorID = blog.GetAuthorId()
	data.Content = blog.GetContent()
	data.Title = blog.GetTitle()

	_, err = collection.ReplaceOne(context.Background(), filter, data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot update object in MongoDB: %v.\n", err),
		)
	}

	return &pb.UpdateBlogResponse{
		Blog: dataToBlogPb(data),
	}, nil
}

func (*GRPCBloggerServer) DeleteBlog(ctx context.Context, req *pb.DeleteBlogRequest) (*pb.DeleteBlogResponse, error) {
	fmt.Println("Delete blog request")
	oID, err := primitive.ObjectIDFromHex(req.GetBlogId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintln("Cannot parse ID."),
		)
	}

	filter := bson.D{{Key: "_id", Value: oID}}

	deleteRes, err := collection.DeleteOne(context.Background(), filter)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot delete object in MongoDB: %v.\n", err),
		)
	}

	if deleteRes.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog in MongoDB: %v.\n", err),
		)
	}

	return &pb.DeleteBlogResponse{BlogId: req.GetBlogId()}, nil

}

func (*GRPCBloggerServer) ListBlog(req *pb.ListBlogRequest, stream pb.BlogService_ListBlogServer) error {
	fmt.Println("List blog request")

	filter := bson.D{}
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}

	defer cursor.Close(context.Background())
	for cursor.Next(context.Background()) {
		data := &blogItem{}
		err := cursor.Decode(data)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data from MongoDB: %v", err),
			)
		}
		stream.Send(&pb.ListBlogResponse{Blog: dataToBlogPb(data)})
	}
	if err := cursor.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	return nil
}
