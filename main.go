package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	queue        = make(chan string, 10)
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "promedemo_processed_ops_total",
		Help: "The total number of processed events",
	})
	opsQueued = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "promedemo",
		Subsystem: "processed",
		Name:      "ops_queued",
		Help:      "Number of blob storage operations waiting to be processed.",
	})
)

func init() {
	//prometheus.MustRegister(opsProcessed)
	prometheus.MustRegister(opsQueued)
	opsQueued.Set(0)
}

func produce() {
	go func() {
		for i := 0; ; i++ {
			var ok bool
			item := fmt.Sprintf("item_%d", i)
			select {
			case queue <- item:
				ok = true
			default:
				ok = false
			}

			if ok {
				opsQueued.Inc()
				fmt.Printf("produce: %s\n", item)
				rdm := rand.Intn(1000)
				time.Sleep((1000 + time.Duration(rdm)) * time.Millisecond)
			} else {
				rdm := rand.Intn(5000)
				time.Sleep((5000 + time.Duration(rdm)) * time.Millisecond)
			}
		}
	}()
}

func consume() {
	go func() {
		for {
			var ok bool
			var item string
			select {
			case item = <-queue:
				ok = true
			default:
				ok = false
			}

			if ok {
				opsQueued.Dec()
				fmt.Printf("consume: %s\n", item)
				rdm := rand.Intn(1000)
				time.Sleep((1000 + time.Duration(rdm)) * time.Millisecond)
			} else {
				rdm := rand.Intn(5000)
				time.Sleep((5000 + time.Duration(rdm)) * time.Millisecond)
			}
		}
	}()
}

func recordMetrics() {
	produce()
	consume()

	go func() {
		for {
			opsProcessed.Inc()
			base := 0
			rdm := rand.Intn(10)
			time.Sleep(time.Duration(base+rdm) * time.Millisecond)
		}
	}()
}

func main() {
	recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":2112", nil)
	if err != nil {
		fmt.Printf("start error: %v", err)
	}
}
