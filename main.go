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
	opsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "promedemo_processed_ops_total",
		Help: "The total number of processed events"},
		[]string{"code", "method"})
	opsQueued = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "promedemo",
		Subsystem: "processed",
		Name:      "ops_queued",
		Help:      "Number of blob storage operations waiting to be processed.",
	})

	opsDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "promedemo",
		Subsystem: "processed",
		Name:      "ops_duration",
		Help:      "The duration of processed events.",
	}, []string{"code", "method"})
)

const (
	success       = 200
	unauthorized  = 301
	notFound      = 404
	internalError = 500
)

const (
	sign       = "sign"
	signVerify = "signVerify"
	encrypt    = "encrypt"
	decrypt    = "decrypt"
)

func init() {
	//prometheus.MustRegister(opsProcessed)
	prometheus.MustRegister(opsQueued)
	opsQueued.Set(0)
	prometheus.Register(opsDuration)
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
			code := getStatusCode()
			method := getMethod()
			duration := getDuration()
			opsProcessed.WithLabelValues(code, method).Inc()
			time.Sleep(time.Duration(duration) * time.Millisecond)
			opsDuration.WithLabelValues(code, method).Observe(float64(duration / 1000.0))
		}
	}()
}

func getStatusCode() string {
	rdm := rand.Intn(1000)
	var code int
	switch {
	case rdm < 800:
		code = success
	case rdm < 900:
		code = unauthorized
	case rdm < 960:
		code = notFound
	default:
		code = internalError
	}
	return fmt.Sprintf("%d", code)
}

func getDuration() int64 {
	rdm := rand.Intn(1000)
	switch {
	case rdm < 900:
		return rand.Int63n(5)
	case rdm < 950:
		return rand.Int63n(10)
	case rdm < 970:
		return rand.Int63n(50)
	case rdm < 980:
		return rand.Int63n(100)
	case rdm < 990:
		return rand.Int63n(500)
	case rdm < 995:
		return rand.Int63n(1000)
	default:
		return rand.Int63n(5000)
	}
}

func getMethod() string {
	rdm := rand.Intn(1000)
	var method string
	switch {
	case rdm < 450:
		method = sign
	case rdm < 900:
		method = signVerify
	case rdm < 950:
		method = encrypt
	default:
		method = decrypt
	}
	return method
}

func main() {
	recordMetrics()
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":2112", nil)
	if err != nil {
		fmt.Printf("start error: %v", err)
	}
}
