package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"os"

	"github.com/bazel-contrib/bazel-lib/tools/common"
	pb "github.com/bazel-contrib/bazel-lib/tools/proto"
	"google.golang.org/protobuf/proto"
)

var logger = log.New(os.Stderr, "", 0)

func parseFlags(args []string) (string, string, bool) {
	flagSet := flag.NewFlagSet("copy_file", flag.ExitOnError)
	src := flagSet.String("src", "", "Input file to be copied")
	dst := flagSet.String("dst", "", "Output location")
	worker_mode := flagSet.Bool("persistent_worker", false, "Start in bazel worker mode")
	err := flagSet.Parse(args)
	if err != nil {
		log.Fatal(err)
	}
	return *src, *dst, *worker_mode
}

func main() {
	src, dst, worker_mode := parseFlags(os.Args)

	if !worker_mode {
		copySingleFile(src, dst)
	} else {
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

	src, dst, _ := parseFlags(workRequest.Arguments)
	copySingleFile(src, dst)
	log.Fatalf("Copied %v -> %v", src, dst)
}

func readStdin(b []byte, reader io.Reader) {
	logger.Printf("Reading %v bytes", len(b))

	n, err := reader.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	if n != len(b) {
		hexdump := hex.EncodeToString(b)
		log.Fatalf("Expected to read %v bytes indicating message length, but found: %v\n%v", len(b), n, hexdump)
	}
}
