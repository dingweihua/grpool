package grpool

import (
	"sync"
)

type Job func()

type worker struct {
	workerChan chan *worker
	jobChan    chan Job
	stop       chan struct{}
}

func (w *worker) start() {
	go func() {
		var job Job
		for {
			// worker free, add it to worker channel
			w.workerChan <- w
			select {
			// execute job
			case job = <-w.jobChan:
				job()
			// wait for stop
			case <-w.stop:
				w.stop <- struct{}{}
				return
			}
		}
	}()
}

func newWorker(workerChan chan *worker) *worker {
	return &worker{
		workerChan: workerChan,
		jobChan:    make(chan Job),
		stop:       make(chan struct{}),
	}
}

type WorkerPool struct {
	JobQueue   chan Job
	workerChan chan *worker
	dispatcher *dispatcher
	wg         sync.WaitGroup
}

func (pool *WorkerPool) Release() {
	pool.dispatcher.stop <- struct{}{}
	<-pool.dispatcher.stop
}

func (pool *WorkerPool) AddCount(count int) {
	pool.wg.Add(count)
}

func (pool *WorkerPool) JobDone() {
	pool.wg.Done()
}

func (pool *WorkerPool) WaitAll() {
	pool.wg.Wait()
}

func NewWorkerPool(workerNum int, jobQueueLen int) *WorkerPool {
	jobQueue := make(chan Job, jobQueueLen)
	workerChan := make(chan *worker, workerNum)

	return &WorkerPool{
		JobQueue:   jobQueue,
		workerChan: workerChan,
		dispatcher: newDispatcher(jobQueue, workerChan),
	}
}

type dispatcher struct {
	JobQueue   chan Job
	workerChan chan *worker
	stop       chan struct{}
}

func (d *dispatcher) dispatch() {
	for {
		select {
		case job := <-d.JobQueue:
			worker := <-d.workerChan
			worker.jobChan <- job
		case <-d.stop:
			for i := 0; i < cap(d.workerChan); i++ {
				// 如果stop的时候正好还有在执行的任务，导致竞争？
				// 不会！因为select中的case是互斥的，进入当前case之后上面的case就不会继续分发job了
				worker := <-d.workerChan
				// TODO 这里的stop信号为什么要request + ack?
				worker.stop <- struct{}{}
				<-worker.stop
			}
			d.stop <- struct{}{}
			return
		}
	}
}

func newDispatcher(jobQueue chan Job, workerChan chan *worker) *dispatcher {
	dispatcher := &dispatcher{
		JobQueue:   jobQueue,
		workerChan: workerChan,
		stop:       make(chan struct{}),
	}

	// 启动worker
	for i := 0; i < cap(dispatcher.workerChan); i++ {
		worker := newWorker(dispatcher.workerChan)
		worker.start()
	}

	// dispatch job to each worker
	go dispatcher.dispatch()
	return dispatcher
}
