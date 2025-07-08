# rq - simple and powerful HTTP Client for Go

A modern HTTP client library for Go that makes HTTP requests simple and intuitive.

## Installation

```bash
go get github.com/k64z/rq
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/k64z/rq"
)

func main() {
    ctx := context.Background()
    
    // Simple GET request
    resp := rq.Get("https://api.github.com/users/k64z").Do(ctx)
    if resp.IsOK() {
        fmt.Println(resp.String())
    }
    
    // POST with JSON
    type User struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }
    
    user := User{Name: "John Doe", Email: "john@example.com"}
    resp = rq.
        Post("https://api.example.com/users").
        BodyJSON(user).
        Do(ctx)
    
    // Parse JSON response
    var created User
    if err := resp.JSON(&created); err == nil {
        fmt.Printf("Created user: %+v\n", created)
    }
}
```
