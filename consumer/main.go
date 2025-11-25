package main

import (
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

	pb "consumer/proto"

	"google.golang.org/grpc"
)

type VideoJob struct {
	Filename string
	Data     []byte
}

type Server struct {
	pb.UnimplementedVideoUploadServiceServer
	queue      chan VideoJob
	uploadDir  string
	previewDir string
}

var (
	threads   = flag.Int("threads", 3, "Number of consumer threads")
	queueSize = flag.Int("queue", 10, "Maximum queue size")
	grpcPort  = flag.Int("grpc-port", 9090, "gRPC port")
	httpPort  = flag.Int("http-port", 8080, "HTTP port")
)

func main() {
	flag.Parse()

	// Create directories
	uploadDir := "uploads"
	previewDir := "previews"
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(previewDir, 0755)

	// Create bounded queue
	queue := make(chan VideoJob, *queueSize)

	// Start consumer workers
	var wg sync.WaitGroup
	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go consumerWorker(i, queue, uploadDir, previewDir, &wg)
	}

	// Start gRPC server
	server := &Server{
		queue:      queue,
		uploadDir:  uploadDir,
		previewDir: previewDir,
	}

	go startGRPCServer(server)

	// Start HTTP server for GUI
	startHTTPServer(uploadDir, previewDir)

	wg.Wait()
}

func consumerWorker(id int, queue chan VideoJob, uploadDir, previewDir string, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("Consumer worker %d started\n", id)

	for job := range queue {
		log.Printf("Worker %d processing: %s\n", id, job.Filename)

		// Save video file
		videoPath := filepath.Join(uploadDir, job.Filename)
		err := os.WriteFile(videoPath, job.Data, 0644)
		if err != nil {
			log.Printf("Worker %d error saving file: %v\n", id, err)
			continue
		}

		// Generate 10-second preview
		previewPath := filepath.Join(previewDir, "preview_"+job.Filename)
		err = generatePreview(videoPath, previewPath)
		if err != nil {
			log.Printf("Worker %d error generating preview: %v\n", id, err)
			continue
		}

		log.Printf("Worker %d completed: %s\n", id, job.Filename)
	}
}

func generatePreview(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-t", "10", "-c", "copy", "-y", outputPath)
	return cmd.Run()
}

func (s *Server) UploadVideo(stream pb.VideoUploadService_UploadVideoServer) error {
	var filename string
	var videoData []byte

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if filename == "" {
			filename = chunk.Filename
		}

		videoData = append(videoData, chunk.Data...)
	}

	// Try to add to queue (non-blocking)
	job := VideoJob{
		Filename: filename,
		Data:     videoData,
	}

	select {
	case s.queue <- job:
		log.Printf("Video queued: %s (size: %d bytes)\n", filename, len(videoData))
		return stream.SendAndClose(&pb.UploadResponse{
			Success: true,
			Message: "Video uploaded successfully",
		})
	default:
		log.Printf("Queue full! Dropped: %s\n", filename)
		return stream.SendAndClose(&pb.UploadResponse{
			Success: false,
			Message: "Queue is full, video dropped",
		})
	}
}

func startGRPCServer(server *Server) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterVideoUploadServiceServer(grpcServer, server)

	log.Printf("gRPC server listening on port %d\n", *grpcPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func startHTTPServer(uploadDir, previewDir string) {
	// Serve static files
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/", fs)

	// API endpoint to list videos
	http.HandleFunc("/api/videos", func(w http.ResponseWriter, r *http.Request) {
		files, err := os.ReadDir(uploadDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("["))
		first := true
		for _, file := range files {
			if !file.IsDir() {
				if !first {
					w.Write([]byte(","))
				}
				w.Write([]byte(fmt.Sprintf(`"%s"`, file.Name())))
				first = false
			}
		}
		w.Write([]byte("]"))
	})

	// Serve video files
	http.Handle("/video/", http.StripPrefix("/video/", http.FileServer(http.Dir(uploadDir))))
	http.Handle("/preview/", http.StripPrefix("/preview/", http.FileServer(http.Dir(previewDir))))

	log.Printf("HTTP server listening on port %d\n", *httpPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), nil))
}
