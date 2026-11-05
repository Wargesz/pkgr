package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	pb "github.com/wargesz/pkgr/proto"

	grpc "google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedDispatcherServer
}

type Blueprint struct {
	Name     string   `json:"name"`
	Source   string   `json:"source"`
	Steps    []string `json:"steps"`
	Binaries []string `json:"binaries"`
	Needs    []string `json:"needs"`
}

var blueprintChan chan Blueprint

func (s *server) ConnectWorker(stream pb.Dispatcher_ConnectWorkerServer) error {
	log.Printf("[Server] A worker has connected!")
	for {
		res := workerLoop(stream)
		if res != nil {
			return res
		}
	}
}

func (s *server) RequestArtifact(req *pb.ArtifactRequest, stream grpc.ServerStreamingServer[pb.BuildResult]) error {
	log.Printf("[Server] A client connected!")
	return nil
}

func workerLoop(stream pb.Dispatcher_ConnectWorkerServer) error {
	bp := <-blueprintChan
	task := createBuildTask(bp, "")
	go stream.Send(&task)
	return workerListen(stream)
}

func workerListen(stream pb.Dispatcher_ConnectWorkerServer) error {
	b := strings.Builder{}
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			log.Printf("[Server] Worker disconnected.")
			return nil
		}
		if err != nil {
			log.Printf("[Server] Error receiving from worker: %v", err)
			return err
		}
		switch content := res.Content.(type) {
		case *pb.BuildResult_Artifact:
			saveArtifact(content.Artifact)
		case *pb.BuildResult_BuildLog:
			b.WriteString(content.BuildLog)
			if strings.Contains(content.BuildLog, "\n") {
				print("[WORKER] ", b.String())
			}
			b.Reset()
		default:
			fmt.Println("no content")
		}
	}
}

func saveArtifact(artifact *pb.Artifact) {
	err := os.WriteFile(artifact.Filename, artifact.Data, 777)
	must(err, "failed to save artifact")
}

func createBuildTask(b Blueprint, clientId string) pb.BuildTask {
	return pb.BuildTask{
		Name:     b.Name,
		Source:   b.Source,
		Needs:    b.Needs,
		Steps:    b.Steps,
		Binaries: b.Binaries,
	}
}

func sendBuildTask(stream pb.Dispatcher_ConnectWorkerServer, task *pb.BuildTask) {
	err := stream.Send(task)
	if err != nil {
		log.Printf("[Server] Failed to send task (%v): %v", task, err)
	}
}

func main() {
	blueprintChan = make(chan Blueprint, 5)
	lis, err := net.Listen("tcp", "192.168.0.15:50051")
	must(err, "failed to listen")

	s := grpc.NewServer()
	pb.RegisterDispatcherServer(s, &server{})
	go order()
	log.Printf("Dispatcher Server listening at %v", lis.Addr())
	err = s.Serve(lis)
	must(err, "failed to serve")
}

func order() {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println(">")
		text, _ := reader.ReadString('\n')
		text = strings.ReplaceAll(text, "\n", "")
		bp := readBlueprint(text)
		blueprintChan <- bp
	}
}

func readBlueprint(blueprintName string) Blueprint {
	var bp Blueprint
	data, err := os.ReadFile("blueprints/" + blueprintName + ".json")
	must(err, "failed to read blueprint json")
	err = json.Unmarshal(data, &bp)
	must(err, "failed to parse json")
	return bp
}

func must(err error, reason string) {
	if err != nil {
		log.Fatalf("%v: %v", reason, err)
	}
}
