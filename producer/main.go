package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	pb "producer/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	threads    = flag.Int("threads", 2, "Number of producer threads")
	serverAddr = flag.String("server", "localhost:9090", "Consumer server address")
	folders    = flag.String("folders", "", "Comma-separated list of folders")
)

const chunkSize = 64 * 1024 // 64KB chunks

func main() {
	flag.Parse()

	if *folders == "" {
		log.Fatal("Please specify folders with -folders flag")
	}

	// Parse folder list
	folderList := parseFolders(*folders)
	if len(folderList) < *threads {
		log.Printf("Warning: %d threads but only %d folders\n", *threads, len(folderList))
		*threads = len(folderList)
	}

	// Connect to consumer
	conn, err := grpc.NewClient(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewVideoUploadServiceClient(conn)

	// Start producer threads
	var wg sync.WaitGroup
	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go producerWorker(i, folderList[i], client, &wg)
	}

	wg.Wait()
	log.Println("All producers finished")
}

func parseFolders(folderStr string) []string {
	var folders []string
	current := ""
	for _, ch := range folderStr {
		if ch == ',' {
			if current != "" {
				folders = append(folders, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		folders = append(folders, current)
	}
	return folders
}

func producerWorker(id int, folder string, client pb.VideoUploadServiceClient, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("Producer %d scanning folder: %s\n", id, folder)

	files, err := os.ReadDir(folder)
	if err != nil {
		log.Printf("Producer %d error reading folder: %v\n", id, err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Check if it's a video file
		ext := filepath.Ext(file.Name())
		if ext != ".mp4" && ext != ".avi" && ext != ".mov" && ext != ".mkv" {
			continue
		}

		videoPath := filepath.Join(folder, file.Name())
		log.Printf("Producer %d uploading: %s\n", id, file.Name())

		err := uploadVideo(client, videoPath, file.Name())
		if err != nil {
			log.Printf("Producer %d error uploading %s: %v\n", id, file.Name(), err)
		} else {
			log.Printf("Producer %d completed: %s\n", id, file.Name())
		}

		time.Sleep(100 * time.Millisecond) // Small delay between uploads
	}
}

func uploadVideo(client pb.VideoUploadServiceClient, filePath, filename string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := client.UploadVideo(ctx)
	if err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	buffer := make([]byte, chunkSize)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		chunk := &pb.VideoChunk{
			Filename: filename,
			Data:     buffer[:n],
			IsLast:   false,
		}

		if err := stream.Send(chunk); err != nil {
			return err
		}
	}

	// Send final chunk
	finalChunk := &pb.VideoChunk{
		Filename: filename,
		Data:     []byte{},
		IsLast:   true,
	}
	stream.Send(finalChunk)

	// Close and receive response
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("upload failed: %s", resp.Message)
	}

	return nil
}
