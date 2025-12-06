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
    - [Windows](https://www.gyan.dev/ffmpeg/builds/)

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
2. Create the necessary directories for the project:

**macOS**
```
mkdir -p proto
mkdir -p consumer/proto
mkdir -p consumer/web
mkdir -p producer/proto
mkdir -p test_videos/folder1
mkdir -p test_videos/folder2
```

**Windows**
```
mkdir proto
mkdir consumer\proto
mkdir consumer\web
mkdir producer\proto
mkdir test_videos\folder1
mkdir test_videos\folder2
```

3. Copy the `video_upload.proto` file to the required locations:

**macOS**
```
cp video_upload.proto proto/
cp video_upload.proto consumer/proto/
cp video_upload.proto producer/proto/
```

**Windows**
```
copy video_upload.proto proto\
copy video_upload.proto consumer\proto\
copy video_upload.proto producer\proto\
```

4. Navigate to the consumer and producer proto directories and generate their respective Go code:

**macOS**
```
cd consumer/proto
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    video_upload.proto
cd ../..
```
```
cd producer/proto
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    video_upload.proto
cd ../..
```

**Windows**
```
cd consumer\proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative video_upload.proto
cd ..\..
```
```
cd producer\proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative video_upload.proto
cd ..\..
```

5. Set up the Go modules:
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

6. Build the consumer and producer executables:

**macOS**
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

**Windows**
```
cd consumer
go build -o consumer-bin.exe
cd ..
```
```
cd producer
go build -o producer-bin.exe
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

