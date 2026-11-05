package main

import (
	context "context"
	"fmt"
	"io"
	log "log"
	"os"
	"sync"

	pb "github.com/wargesz/pkgr/proto"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type logWriter struct {
	mu     sync.Mutex
	stream pb.Dispatcher_ConnectWorkerClient
}

var stream grpc.BidiStreamingClient[pb.BuildResult, pb.BuildTask]

func main() {
	conn, err := grpc.NewClient(os.Getenv("HOST")+":50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	must(err)
	defer conn.Close()
	client := pb.NewDispatcherClient(conn)
	stream, err = client.ConnectWorker(context.Background())
	must(err)
	for {
		task, err := stream.Recv()
		if err == io.EOF {
			break
		}
		must(err)
		build(task)
		fmt.Println("Finished build of:", task.Name)
		fmt.Println("Creating artifact...")
		createArtifact(task)
		stream.Send(&pb.BuildResult{
			Content: &pb.BuildResult_Artifact{
				Artifact: &pb.Artifact{
					Filename: task.Name + ".zip",
					Data:     getArtifactData(task),
				},
			},
		})
	}
}

func getArtifactData(task *pb.BuildTask) []byte {
	data, err := os.ReadFile(os.Getenv("PREFIX") + task.Name + ".zip")
	must(err)
	return data
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	err := w.stream.Send(&pb.BuildResult{
		Content: &pb.BuildResult_BuildLog{
			BuildLog: string(p),
		},
	})
	fmt.Print(string(p))
	must(err)
	return len(p), nil
}

func must(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
