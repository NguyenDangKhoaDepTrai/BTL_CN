# Distributed File Transfer System

A Go-based distributed file transfer system that splits large files into smaller pieces and transfers them concurrently over multiple TCP connections.

## Description

This project provides a robust solution for transferring large files by:
- Splitting large files into smaller, manageable pieces (512KB each)
- Distributing these pieces across multiple TCP connections
- Concurrent transfer of file pieces for improved performance
- Reassembling the pieces back into the original file

### Key Features
- **File Splitting**: Breaks down large files into 512KB chunks
- **Concurrent Transfer**: Uses multiple TCP connections (ports 8080+) for parallel transfer
- **Distributed Hosting**: Each piece can be served independently
- **Automatic Reassembly**: Pieces are automatically merged back into the original file
- **Error Handling**: Robust error handling for network and file operations

## Getting Started

### Prerequisites

You need to have installed:
```bash
Go 1.15 or higher
```

### Installation

1. Clone the repository
```bash
git clone https://github.com/NguyenDangKhoaDepTrai/BTL_MMT_HK241.git
```

2. Navigate to the project directory
```bash
cd distributed-file-transfer
```

### Running the Project

1. First, split your file:
```bash
go run split/split.go
# Enter the filename when prompted
# Example: large_file.mp4
# The file will be split into pieces (piece_0.dat, piece_1.dat, etc.)
# Cut all the pieces to the hosting folder
```



2. Start the hosting servers:
```bash
go run hosting/hosting.go
# Enter the number of pieces to serve
```

3. Receive and merge the files:
```bash
go run output/main.go
# Enter the number of pieces to receive
# Enter the output filename
# Example: large_file.mp4
```

## Usage Example

1. To split a file named "large_file.mp4":
   - Run the split program
   - Enter "large_file.mp4" when prompted
   - The file will be split into pieces (piece_0.dat, piece_1.dat, etc.)
   - Cut all the pieces to the hosting folder

2. To host the pieces:
   - Run the hosting program
   - Enter the number of pieces
   - The program will start serving each piece on a different port

3. To receive and reassemble the file:
   - Run the output program
   - Enter the number of pieces to receive
   - Enter the desired output filename
   - The program will download all pieces concurrently and merge them
