package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	pb "github.com/yourusername/p3/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const chunkSize = 64 * 1024 // 64KB chunks

type Producer struct {
	id         int
	serverAddr string
	videoDir   string
}

func NewProducer(id int, serverAddr string) *Producer {
	return &Producer{
		id:         id,
		serverAddr: serverAddr,
		videoDir:   fmt.Sprintf("../videos/producer%d", id),
	}
}

func (p *Producer) uploadVideo(filename string) error {
	// Read the video file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Connect to consumer
	conn, err := grpc.Dial(p.serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewMediaUploadClient(conn)
	stream, err := client.UploadVideo(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create stream: %v", err)
	}

	// Send video in chunks
	basename := filepath.Base(filename)
	totalSize := int64(len(data))
	chunkCount := (len(data) + chunkSize - 1) / chunkSize

	log.Printf("Producer %d: Uploading %s (%d bytes, %d chunks)", p.id, basename, totalSize, chunkCount)

	for i := 0; i < chunkCount; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := &pb.VideoChunk{
			Filename:    basename,
			Data:        data[start:end],
			ChunkNumber: int64(i),
			IsLast:      i == chunkCount-1,
		}

		if i == 0 {
			chunk.TotalSize = totalSize
		}

		if err := stream.Send(chunk); err != nil {
			return fmt.Errorf("failed to send chunk %d: %v", i, err)
		}

		// Wait for ACK
		resp, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("failed to receive ACK for chunk %d: %v", i, err)
		}

		if resp.Status == pb.UploadResponse_QUEUE_FULL {
			log.Printf("Producer %d: Queue full! Video %s was dropped", p.id, basename)
			return fmt.Errorf("queue full")
		}

		if resp.Status == pb.UploadResponse_ERROR {
			log.Printf("Producer %d: Error uploading %s: %s", p.id, basename, resp.Message)
			return fmt.Errorf("upload error: %s", resp.Message)
		}

		// Progress indicator
		if i%10 == 0 || i == chunkCount-1 {
			progress := float64(i+1) / float64(chunkCount) * 100
			log.Printf("Producer %d: Progress %s: %.1f%%", p.id, basename, progress)
		}
	}

	// Close and get final response
	if err := stream.CloseSend(); err != nil {
		return fmt.Errorf("failed to close stream: %v", err)
	}

	log.Printf("Producer %d: Successfully uploaded %s", p.id, basename)
	return nil
}

func (p *Producer) run() {
	log.Printf("Producer %d starting, reading from %s", p.id, p.videoDir)

	// Check if directory exists
	if _, err := os.Stat(p.videoDir); os.IsNotExist(err) {
		log.Fatalf("Producer %d: Directory %s does not exist", p.id, p.videoDir)
	}

	// Find all video files
	files, err := filepath.Glob(filepath.Join(p.videoDir, "*"))
	if err != nil {
		log.Fatalf("Producer %d: Failed to list files: %v", p.id, err)
	}

	if len(files) == 0 {
		log.Printf("Producer %d: No video files found in %s", p.id, p.videoDir)
		return
	}

	log.Printf("Producer %d: Found %d files to upload", p.id, len(files))

	// Upload each video
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}

		// Only process video files
		ext := filepath.Ext(file)
		if ext != ".mp4" && ext != ".avi" && ext != ".mov" && ext != ".mkv" {
			log.Printf("Producer %d: Skipping non-video file %s", p.id, file)
			continue
		}

		log.Printf("Producer %d: Starting upload of %s", p.id, filepath.Base(file))
		
		if err := p.uploadVideo(file); err != nil {
			log.Printf("Producer %d: Failed to upload %s: %v", p.id, filepath.Base(file), err)
		}

		// Small delay between uploads
		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("Producer %d: Finished uploading all videos", p.id)
}

func main() {
	id := flag.Int("id", 1, "Producer ID")
	server := flag.String("server", "localhost:50051", "Consumer server address")
	flag.Parse()

	producer := NewProducer(*id, *server)
	producer.run()
}