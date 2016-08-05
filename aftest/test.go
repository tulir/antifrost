package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	go func() {
		for {
			bytes := make([]byte, 128)
			n, err := os.Stdin.Read(bytes)
			fmt.Println(n, err, string(bytes))
		}
	}()
	for i := 1; i < 5; i++ {
		time.Sleep(time.Duration(i) * time.Second)
		fmt.Println("hi", i)
	}
}
