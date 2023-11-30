package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/bruce-mig/grpc-crud-mongodb/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const addr = "localhost:50051"

func main() {
	fmt.Println("Blog Client")
	opts := grpc.WithTransportCredentials(insecure.NewCredentials())

	cc, err := grpc.Dial(addr, opts)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer cc.Close()

	c := pb.NewBlogServiceClient(cc)

	// Create Blog
	createBlogRes := callCreateBlog(ctx, c)
	fmt.Printf("Blog has been created: %v\n", createBlogRes)

	// Read Blog
	blogID := createBlogRes.Blog.GetId()
	readBlogRes := callReadBlog(ctx, c, blogID)
	fmt.Printf("Blog was read: %v\n", readBlogRes)

	// Update Blog
	newBlog := &pb.Blog{
		Id:       blogID,
		AuthorId: "Changed Author",
		Title:    "My First Blog (edited)",
		Content:  "Content of the first blog, with some awesome additions",
	}
	updateRes := callUpdateBlog(ctx, c, newBlog)
	fmt.Printf("Blog was updated: %v.\n", updateRes)
	// Delete blog
	deleteRes := callDeleteBlog(ctx, c, blogID)
	fmt.Printf("Blog was deleted: %v.\n", deleteRes)

	// List Blogs
	callListBlogs(ctx, c)
}

func callCreateBlog(ctx context.Context, c pb.BlogServiceClient) *pb.CreateBlogResponse {
	fmt.Println("Creating the blog")
	blog := &pb.Blog{
		AuthorId: "Bruce",
		Title:    "My First Blog",
		Content:  "Content of the first blog",
	}

	blogReq := &pb.CreateBlogRequest{
		Blog: blog,
	}

	createBlogRes, err := c.CreateBlog(ctx, blogReq)
	if err != nil {
		log.Fatalf("unexpected error: %v\n", err)
	}
	return createBlogRes

}

func callReadBlog(ctx context.Context, c pb.BlogServiceClient, blogID string) *pb.ReadBlogResponse {
	fmt.Println("Reading the blog")

	readBlogReq := &pb.ReadBlogRequest{BlogId: blogID}
	readBlogRes, err := c.ReadBlog(context.Background(), readBlogReq)
	if err != nil {
		fmt.Printf("Error happened while reading: %v\n", err)
	}
	return readBlogRes
}

func callUpdateBlog(ctx context.Context, c pb.BlogServiceClient, newBlog *pb.Blog) *pb.UpdateBlogResponse {

	updateRes, err := c.UpdateBlog(context.Background(), &pb.UpdateBlogRequest{Blog: newBlog})
	if err != nil {
		fmt.Printf("Error happened while updating: %v\n", err)

	}
	return updateRes
}

func callDeleteBlog(ctx context.Context, c pb.BlogServiceClient, blogID string) *pb.DeleteBlogResponse {
	deleteRes, err := c.DeleteBlog(context.Background(), &pb.DeleteBlogRequest{BlogId: blogID})
	if err != nil {
		fmt.Printf("Error happened while deleting: %v\n", err)

	}
	return deleteRes
}

func callListBlogs(ctx context.Context, c pb.BlogServiceClient) {
	stream, err := c.ListBlog(context.Background(), &pb.ListBlogRequest{})
	if err != nil {
		log.Fatalf("error while calling list by RPC: %v", err)
	}
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Something happened: %v", err)
		}
		fmt.Println(res.GetBlog())
	}
}

// start Mongo with IP bind 0.0.0.0.
// If running as a service then update the file C:\Program Files\MongoDB\Server\X.X\bin\mongod.cfg
// and then connect using
// mongodb://user:pass@IP_HOST:27017/dbname
// where you can find host IP by running the command
// cat /etc/resolv.conf
// ip addr | grep eth0
