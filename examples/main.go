package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan int, 10)

	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(100*time.Millisecond)
			fmt.Println("  > writing:", i)
			ch <- i
			fmt.Println("  > written:", i)
		}
		close(ch)
	}()

	for {
		select {
		case v, ok := <-ch:
			if !ok {
				fmt.Println("channel closed. exit")
				goto END
			}
			fmt.Println("received:", v)
			time.Sleep(2*time.Second)
		}
	}

	END:
		fmt.Println("end")
}
