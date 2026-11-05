package main

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	pb "github.com/wargesz/pkgr/proto"
)

var GLOBALS map[string]string

func build(task *pb.BuildTask) {
	envSetup(task.Name)
	cleanDirectories()
	installDependencies(task)
	downloadSource(task)
	extractArchive(task)
	runSteps(task)
}

func installDependencies(task *pb.BuildTask) {
	if len(task.Needs) == 0 {
		return
	}
	writer := &logWriter{stream: stream}
	args := []string{"-y"}
	args = append(args, task.Needs...)
	cmd := exec.Command("xbps-install", args...)
	cmd.Dir = "build/"
	cmd.Stdout = writer
	cmd.Stderr = writer
	err := cmd.Run()
	must(err)
}

func envSetup(name string) {
	GLOBALS = make(map[string]string)
	GLOBALS["$PREFIX"] = os.Getenv("PREFIX") + "/" + name
	GLOBALS["$NPROC"] = strconv.Itoa(runtime.NumCPU())
}

func runSteps(task *pb.BuildTask) {
	writer := &logWriter{stream: stream}
	files, err := os.ReadDir("build/")
	must(err)
	for _, step := range task.Steps {
		parts := strings.Split(evaluateVariable(step), " ")
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = "build/" + files[0].Name()
		cmd.Stdout = writer
		cmd.Stderr = writer
		err := cmd.Run()
		must(err)
	}
}

func evaluateVariable(in string) string {
	for k, v := range GLOBALS {
		if strings.Contains(in, k) {
			in = strings.Replace(in, k, v, 1)
		}
	}
	return in
}

func downloadSource(task *pb.BuildTask) {
	writer := &logWriter{stream: stream}
	cmd := exec.Command("wget", task.Source)
	cmd.Dir = "build/"
	cmd.Stdout = writer
	cmd.Stderr = writer
	err := cmd.Run()
	must(err)
}

func cleanDirectories() {
	os.RemoveAll("build/")
	os.Mkdir("build/", 0777)
	os.RemoveAll("out/")
	os.Mkdir("out/", 0777)
}

func createArtifact(task *pb.BuildTask) {
	binaries := strings.Join(task.Binaries, ",")
	os.WriteFile(os.Getenv("PREFIX")+"/"+task.Name+"/"+"binaries.txt", []byte(binaries), 777)
	writer := &logWriter{stream: stream}
	cmd := exec.Command("zip", "-r", task.Name+".zip", os.Getenv("PREFIX")+task.Name)
	cmd.Dir = os.Getenv("PREFIX")
	cmd.Stdout = writer
	cmd.Stderr = writer
	err := cmd.Run()
	must(err)
}

func extractArchive(task *pb.BuildTask) {
	if strings.Index(task.Source, ".zip") != -1 {
		extractArchiveZip(task)
	}
	if strings.Index(task.Source, ".tar.") != -1 {
		extractArchiveTar(task)
	}
}

func extractArchiveZip(task *pb.BuildTask) {
	writer := &logWriter{stream: stream}
	files, err := os.ReadDir("build/")
	must(err)
	cmd := exec.Command("unzip", "-q", files[0].Name())
	cmd.Dir = "build/"
	cmd.Stdout = writer
	cmd.Stderr = writer
	err = cmd.Run()
	must(err)
	os.Remove("build/" + files[0].Name())
}

func extractArchiveTar(task *pb.BuildTask) {
	writer := &logWriter{stream: stream}
	files, err := os.ReadDir("build/")
	must(err)
	cmd := exec.Command("tar", "xf", files[0].Name())
	cmd.Dir = "build/"
	cmd.Stdout = writer
	cmd.Stderr = writer
	err = cmd.Run()
	must(err)
	os.Remove("build/" + files[0].Name())
}
