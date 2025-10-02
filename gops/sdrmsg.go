package main

import (
    "fmt"
)

// Assume QUEUE_NAME, MAX_SIZE, sdrstat, etc. are defined elsewhere

// In Go, we use channels for message queues instead of POSIX mq
var messageQueue = make(chan string, 10)

func serverthread(arg interface{}) interface{} {
    for sdrstat.StopFlag == 0 {
        select {
        case msg := <-messageQueue:
            fmt.Print(msg)
        default:
            // No message, continue
        }
    }
    fmt.Println("Received stopflag, stopping server thread")
    // Cleanup: nothing needed for Go channels
    fmt.Println("At server return")
    return nil
}
