package main

import (
	"fmt"
	"github.com/dingweihua/grpool"
	"time"
)

func main() {
	workerNum, jobQueueLen := 10, 13

	startTime := time.Now()
	pool := grpool.NewWorkerPool(workerNum, jobQueueLen)
	defer pool.Release()

	jobNum := 100
	pool.AddCount(jobNum)
	for i := 0; i < jobNum; i++ {
		count := i
		pool.JobQueue <- func() {
			fmt.Printf("hello, number %d\n", count)
			time.Sleep(time.Second * 1)
			pool.JobDone()
		}
	}
	pool.WaitAll()
	fmt.Printf("Consuming %f seconds\n", time.Now().Sub(startTime).Seconds())
}
