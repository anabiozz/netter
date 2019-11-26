package main

import (
	"fmt"
	"netter"
	"time"
)

func main() {
	c := netter.NewClient()
	c.WaitMin = 10 * time.Second
	c.WaitMax = 1 * time.Second
	c.Max = 2

	resp, err := c.Get("http://127.0.0.1:9595")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("resp", resp)

}
