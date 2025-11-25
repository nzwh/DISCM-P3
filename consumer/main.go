package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	pb "github.com/yourusername/p3/proto"
	"google.golang.org/grpc"
)

type VideoJob struct {
	Filename string
	Data     []byte
	Hash     string
}

type VideoMetadata struct {
	Filename    string    `json:"filename"`
	FullPath    string    `json:"full_path"`
	PreviewPath string    `json:"preview_path"`
	UploadTime  time.Time `json:"upload_time"`
	Size        int64     `json:"size"`
}

type Consumer struct {
	pb.UnimplementedMediaUploadServer
	queue          chan VideoJob
	videos         []VideoMetadata
	videosMutex    sync.RWMutex
	uploadDir      string
	previewDir     string
	duplicates     map[string]bool
	duplicatesMutex sync.RWMutex
}

func NewConsumer(queueSize int, consumerCount int, uploadDir, previewDir string) *Consumer {
	c := &Consumer{
		queue:      make(chan VideoJob, queueSize),
		videos:     make([]VideoMetadata, 0),
		uploadDir:  uploadDir,
		previewDir: previewDir,
		duplicates: make(map[string]bool),
	}

	// Start consumer workers
	for i := 0; i < consumerCount; i++ {
		go c.worker(i)
	}

	return c
}

// gRPC UploadVideo handler
func (c *Consumer) UploadVideo(stream pb.MediaUpload_UploadVideoServer) error {
	var filename string
	var buffer []byte

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if chunk.ChunkNumber == 0 {
			filename = chunk.Filename
			buffer = make([]byte, 0, chunk.TotalSize)
		}

		buffer = append(buffer, chunk.Data...)

		// Send ACK
		if err := stream.Send(&pb.UploadResponse{
			Status:      pb.UploadResponse_ACK,
			ChunkNumber: chunk.ChunkNumber,
		}); err != nil {
			return err
		}

		if chunk.IsLast {
			// Calculate hash for duplicate detection
			hash := hashData(buffer)

			// Check for duplicates
			c.duplicatesMutex.RLock()
			isDuplicate := c.duplicates[hash]
			c.duplicatesMutex.RUnlock()

			if isDuplicate {
				log.Printf("Duplicate video detected: %s", filename)
				return stream.Send(&pb.UploadResponse{
					Status:  pb.UploadResponse_ERROR,
					Message: "Duplicate video",
				})
			}

			// Try to add to queue (non-blocking)
			job := VideoJob{
				Filename: filename,
				Data:     buffer,
				Hash:     hash,
			}

			select {
			case c.queue <- job:
				log.Printf("Video queued: %s (size: %d bytes)", filename, len(buffer))
				return stream.Send(&pb.UploadResponse{
					Status:  pb.UploadResponse_SUCCESS,
					Message: "Video uploaded successfully",
				})
			default:
				log.Printf("Queue full! Dropping video: %s", filename)
				return stream.Send(&pb.UploadResponse{
					Status:  pb.UploadResponse_QUEUE_FULL,
					Message: "Queue is full, video dropped",
				})
			}
		}
	}

	return nil
}

// Worker processes videos from the queue
func (c *Consumer) worker(id int) {
	log.Printf("Consumer worker %d started", id)

	for job := range c.queue {
		log.Printf("Worker %d processing: %s", id, job.Filename)

		// Save full video
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("%s_%s", timestamp, job.Filename)
		fullPath := filepath.Join(c.uploadDir, filename)

		if err := os.WriteFile(fullPath, job.Data, 0644); err != nil {
			log.Printf("Worker %d error saving video: %v", id, err)
			continue
		}

		// Generate preview (first 10 seconds)
		previewFilename := fmt.Sprintf("preview_%s", filename)
		previewPath := filepath.Join(c.previewDir, previewFilename)

		if err := generatePreview(fullPath, previewPath); err != nil {
			log.Printf("Worker %d error generating preview: %v", id, err)
			// Continue even if preview fails
		}

		// Mark as not duplicate
		c.duplicatesMutex.Lock()
		c.duplicates[job.Hash] = true
		c.duplicatesMutex.Unlock()

		// Add to video list
		metadata := VideoMetadata{
			Filename:    job.Filename,
			FullPath:    "/videos/" + filename,
			PreviewPath: "/previews/" + previewFilename,
			UploadTime:  time.Now(),
			Size:        int64(len(job.Data)),
		}

		c.videosMutex.Lock()
		c.videos = append(c.videos, metadata)
		c.videosMutex.Unlock()

		log.Printf("Worker %d completed: %s", id, job.Filename)
	}
}

func hashData(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func generatePreview(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-t", "10",
		"-c", "copy",
		"-y",
		outputPath,
	)
	return cmd.Run()
}

// HTTP handlers
func (c *Consumer) handleVideos(w http.ResponseWriter, r *http.Request) {
	c.videosMutex.RLock()
	defer c.videosMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c.videos)
}

func main() {
	consumerCount := flag.Int("c", 3, "Number of consumer workers")
	queueSize := flag.Int("q", 10, "Queue size")
	grpcPort := flag.String("grpc-port", "50051", "gRPC port")
	httpPort := flag.String("http-port", "8080", "HTTP port")
	flag.Parse()

	// Create directories
	uploadDir := "uploads/full"
	previewDir := "uploads/previews"
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(previewDir, 0755)

	// Create consumer
	consumer := NewConsumer(*queueSize, *consumerCount, uploadDir, previewDir)

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", ":"+*grpcPort)
		if err != nil {
			log.Fatalf("Failed to listen: %v", err)
		}

		grpcServer := grpc.NewServer()
		pb.RegisterMediaUploadServer(grpcServer, consumer)

		log.Printf("gRPC server listening on port %s", *grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Start HTTP server for GUI
	http.HandleFunc("/api/videos", consumer.handleVideos)
	http.Handle("/videos/", http.StripPrefix("/videos/", http.FileServer(http.Dir(uploadDir))))
	http.Handle("/previews/", http.StripPrefix("/previews/", http.FileServer(http.Dir(previewDir))))
	http.Handle("/", http.FileServer(http.Dir("static")))

	log.Printf("HTTP server listening on port %s", *httpPort)
	log.Printf("Consumer started with %d workers and queue size %d", *consumerCount, *queueSize)
	log.Fatal(http.ListenAndServe(":"+*httpPort, nil))
}