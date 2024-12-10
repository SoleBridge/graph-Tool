# Graph Tool Documentation

## Overview

The **Graph Tool** is a graph drawing program written in Go. It allows users to:
- Place and label vertices.
- Recolor vertices.
- Dynamically add/remove vertices and edges,
- Drag and reposition vertices.
- Output graph informmation, such as degrees of vertices and the adjacency matrix.

## Prerequisites

Before compiling the program, ensure you have [Go](https://golang.org/) installed.

Clone the repository:
```bash
git clone https://github.com/SoleBridge/graph-Tool.git
```

## Compilation

Navigate to the project directory and run:
```bash
go build -o graph-tool main.go
```
to generate the executable file `graph-tool`.

Alternatively, you can run:
```bash
go run .
```
to compile and run the code.

## Features
- **Vertex Placement**: Add vertices and label them dynamically.
- **Edge Drawing**: Add edges using straight lines or Bezier curves.
- **Interactive Manipulation**: Drag and reposition vertices, add or remove components dynamically.
- **Curves**: Parallel edges and lops are drawn with Brezier curves.
