# Networked Producer and Consumer

## Prerequisites
- **Golang**
    - macOS: `brew install go`
    - [Windows](https://go.dev/doc/install)
- **Protocol Buffers (protoc)**
    - macOS: `brew install protobuf`
    - [Windows](https://github.com/protocolbuffers/protobuf/releases)
- **FFMPEG**
    - macOS: `brew install  ffmpeg`
    - [Windows](https://www.ffmpeg.org/download.html)

## Installation
1. Install the following Go plugins required to run the program: 

**protobuf**
```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

**grpc**
```
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```
2. Run the following commands to initialize the project: 

**Project Structure**
```
mkdir -p proto
mkdir -p consumer/proto
mkdir -p consumer/web
mkdir -p producer/proto
mkdir -p test_videos/folder1
mkdir -p test_videos/folder2
```

**Proto File**
```
cp video_upload.proto proto/
cp video_upload.proto consumer/proto/
cp video_upload.proto producer/proto/
```

**gRPC Code**
```
cd consumer/proto
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       video_upload.proto
cd ../..
```

**Go Modules**
```
cd producer/proto
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       video_upload.proto
cd ../..
```

```
cd consumer
go mod init consumer
go mod tidy
cd ..
```

```
cd producer
go mod init producer
go mod tidy
cd ..
```

3. Build  the consumer and producer:

```
cd consumer
go build -o consumer-bin
cd ..
```

```
cd producer
go build -o producer-bin
cd ..
```

## Usage
1. Freely modify the test video folders and their content inside the `test_videos` directory. 
2. Run the following commands **in order** in **two separate terminals**:
### Consumer
**macOS**
```
cd consumer && ./consumer-bin
```
**Windows**
```
cd consumer ^&^& consumer-bin.exe
```

### Producer
**macOS**
```
cd producer && ./producer-bin -threads 2 -server localhost:9090 -folders ../test_videos/folder1,../test_videos/folder2
```
**Windows**
```
cd producer ^&^& producer-bin.exe -threads 2 -server localhost:9090 -folders ..\test_videos\folder1,..\test_videos\folder2
```
3. The GUI can be accessed at: https://localhost:8080
