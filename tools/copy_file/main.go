package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/bazel-contrib/bazel-lib/tools/common"
	pb "github.com/bazel-contrib/bazel-lib/tools/worker"
	"google.golang.org/protobuf/proto"
)

var logger = log.New(os.Stderr, "", 0)

func parseFlags(args []string) (string, string, bool) {
	fs := flag.NewFlagSet("copy_file", flag.ExitOnError)

	src := fs.String("src", "", "source file path")
	dest := fs.String("dest", "", "destination file path")
	persistentWorker := fs.Bool("persistent_worker", false, "enable persistent worker")

	fs.Parse(args)

	if *src == "" && *dest == "" && !*persistentWorker {
		fmt.Fprintf(os.Stderr, "No arguments provided. Please provide at least one argument: %v\n", args)
	}

	// fmt.Fprintln(os.Stderr, "Source file:", *src)
	// fmt.Fprintln(os.Stderr, "Destination file:", *dest)
	// fmt.Fprintln(os.Stderr, "Persistent worker enabled:", *persistentWorker)
	return *src, *dest, *persistentWorker
}

func main() {
	pb.RunWorker()
	src, dst, worker_mode := parseFlags(os.Args[1:])
	// fmt.Fprintf(os.Stderr, "Copying %v -> %v in worker mode: %v from %v\n", src, dst, worker_mode, os.Args)
	if !worker_mode {
		copySingleFile(src, dst)
	} else {
		fmt.Fprintf(os.Stderr, "Starting worker mode\n")
		copyWorker()
	}
}

func copySingleFile(src string, dst string) {
	err := common.CopyFile(src, dst)
	if err != nil {
		log.Fatal(err)
	}
}

func copyWorker() {

	for {
		consumeWorkRequest()
	}
}

func consumeWorkRequest() {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	msgLen, err := binary.ReadUvarint(reader)
	if err != nil {
		log.Fatal(err)
	}
	messageBuf := make([]byte, msgLen)
	readStdin(messageBuf, reader)

	workRequest := &pb.WorkRequest{}
	err = proto.Unmarshal(messageBuf, workRequest)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Fprintf(os.Stderr, "Received work request: %v\n", workRequest)
	src, dst, _ := parseFlags(workRequest.Arguments)
	copySingleFile(src, dst)
	response := &pb.WorkResponse{
		ExitCode:     0,
		RequestId:    workRequest.RequestId,
		WasCancelled: false,
	}
	workResponse, err := proto.Marshal(response)
	if err != nil {
		log.Fatal(err)
	}
	writeStdout(workResponse, writer)
	// log.Fatalf("Copied %v -> %v", src, dst)
}

func writeStdout(b []byte, writer *bufio.Writer) {
	// fmt.Fprintf(os.Stderr, "Writing %v bytes\n", len(b))
	uvariantBuffer := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(uvariantBuffer, uint64(len(b)))
	n, err := writer.Write(uvariantBuffer[:n])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing message length: %v\n", err)
	}

	n, err = writer.Write(b)
	if err != nil {
		log.Fatal(err)
	}

	err = writer.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error flushing writer: %v\n", err)
	}

	if n != len(b) {
		hexdump := hex.EncodeToString(b)
		fmt.Fprintf(os.Stderr, "Expected to write %v bytes indicating message length, but found: %v\n%v", len(b), n, hexdump)
	}
}

func readStdin(b []byte, reader io.Reader) {
	// fmt.Fprintf(os.Stderr, "Reading %v bytes\n", len(b))

	n, err := reader.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	if n != len(b) {
		hexdump := hex.EncodeToString(b)
		fmt.Fprintf(os.Stderr, "Expected to read %v bytes indicating message length, but found: %v\n%v", len(b), n, hexdump)
	}
}
