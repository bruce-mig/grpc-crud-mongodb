package main

import (
	"context"
	"fmt"
	"log"

	pb "github.com/bruce-mig/grpc-crud-mongodb/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	port = ":8080"
)

func main() {
	fmt.Println("Blog Client")
	opts := grpc.WithTransportCredentials(insecure.NewCredentials())

	cc, err := grpc.Dial("0.0.0.0"+port, opts)
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer cc.Close()

	c := pb.NewBlogServiceClient(cc)

	fmt.Println("Creating the blog")
	blog := &pb.Blog{
		AuthorId: "Bruce",
		Title:    "My First Blog",
		Content:  "Content of the first blog",
	}

	blogReq := &pb.CreateBlogRequest{
		Blog: blog,
	}

	createBlogRes, err := c.CreateBlog(context.Background(), blogReq)
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	fmt.Printf("Blog has been created: %v", createBlogRes)

}
