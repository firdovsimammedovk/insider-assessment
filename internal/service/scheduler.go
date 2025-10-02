package service

import (
	"log"
	"time"
)

type Scheduler struct {
	service   *MessageService
	interval  time.Duration
	ticker    *time.Ticker
	stopChan  chan struct{}
	isRunning bool
}

func NewScheduler(service *MessageService, interval time.Duration) *Scheduler {
	return &Scheduler{
		service:  service,
		interval: interval,
	}
}

func (sch *Scheduler) Start() error {
	if sch.isRunning {
		log.Println("Scheduler is already running.")
		return nil
	}
	sch.ticker = time.NewTicker(sch.interval)
	sch.stopChan = make(chan struct{})
	sch.isRunning = true
	go func() {
		log.Println("Message scheduler started.")
		for {
			select {
			case <-sch.stopChan:
				log.Println("Message scheduler stopping...")
				sch.ticker.Stop()
				sch.isRunning = false
				log.Println("Message scheduler stopped.")
				return
			case <-sch.ticker.C:
				err := sch.service.ProcessPendingMessages(2)
				if err != nil {
					log.Printf("Error processing pending messages: %v", err)
				}
			}
		}
	}()
	return nil
}

func (sch *Scheduler) Stop() error {
	if !sch.isRunning {
		log.Println("Scheduler is not running.")
		return nil
	}
	close(sch.stopChan)
	return nil
}

func (sch *Scheduler) IsRunning() bool {
	return sch.isRunning
}
