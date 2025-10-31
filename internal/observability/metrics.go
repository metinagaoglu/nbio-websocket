package observability

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Metrics holds server performance metrics
type Metrics struct {
	startTime        time.Time
	connectedClients atomic.Int64
	totalConnections atomic.Uint64
	totalDisconnects atomic.Uint64
	messagesSent     atomic.Uint64
	messagesReceived atomic.Uint64
	broadcastsSent   atomic.Uint64
	errorsTotal      atomic.Uint64
	parseErrors      atomic.Uint64
	handlerErrors    atomic.Uint64
	handlerPanics    atomic.Uint64
}

var serverMetrics *Metrics

// InitMetrics initializes the global metrics instance
func InitMetrics() {
	serverMetrics = &Metrics{
		startTime: time.Now(),
	}
	Info("Metrics initialized")
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	if serverMetrics == nil {
		InitMetrics()
	}
	return serverMetrics
}

// Snapshot returns current metrics values
type MetricsSnapshot struct {
	Uptime            string  `json:"uptime"`
	UptimeSeconds     int64   `json:"uptime_seconds"`
	ConnectedClients  int64   `json:"connected_clients"`
	TotalConnections  uint64  `json:"total_connections"`
	TotalDisconnects  uint64  `json:"total_disconnects"`
	MessagesSent      uint64  `json:"messages_sent"`
	MessagesReceived  uint64  `json:"messages_received"`
	BroadcastsSent    uint64  `json:"broadcasts_sent"`
	ErrorsTotal       uint64  `json:"errors_total"`
	ParseErrors       uint64  `json:"parse_errors"`
	HandlerErrors     uint64  `json:"handler_errors"`
	HandlerPanics     uint64  `json:"handler_panics"`
	MessagesPerSecond float64 `json:"messages_per_second"`
}

// Snapshot returns a point-in-time snapshot of metrics
func (m *Metrics) Snapshot() MetricsSnapshot {
	uptimeSeconds := int64(time.Since(m.startTime).Seconds())
	messagesReceived := m.messagesReceived.Load()

	var messagesPerSecond float64
	if uptimeSeconds > 0 {
		messagesPerSecond = float64(messagesReceived) / float64(uptimeSeconds)
	}

	return MetricsSnapshot{
		Uptime:            time.Since(m.startTime).String(),
		UptimeSeconds:     uptimeSeconds,
		ConnectedClients:  m.connectedClients.Load(),
		TotalConnections:  m.totalConnections.Load(),
		TotalDisconnects:  m.totalDisconnects.Load(),
		MessagesSent:      m.messagesSent.Load(),
		MessagesReceived:  messagesReceived,
		BroadcastsSent:    m.broadcastsSent.Load(),
		ErrorsTotal:       m.errorsTotal.Load(),
		ParseErrors:       m.parseErrors.Load(),
		HandlerErrors:     m.handlerErrors.Load(),
		HandlerPanics:     m.handlerPanics.Load(),
		MessagesPerSecond: messagesPerSecond,
	}
}

func (m *Metrics) IncrementConnections() {
	m.totalConnections.Add(1)
	m.connectedClients.Add(1)
}

func (m *Metrics) IncrementDisconnects() {
	m.totalDisconnects.Add(1)
	m.connectedClients.Add(-1)
}

func (m *Metrics) IncrementMessagesSent() {
	m.messagesSent.Add(1)
}

func (m *Metrics) IncrementMessagesReceived() {
	m.messagesReceived.Add(1)
}

func (m *Metrics) IncrementBroadcasts() {
	m.broadcastsSent.Add(1)
}

func (m *Metrics) IncrementErrors() {
	m.errorsTotal.Add(1)
}

func (m *Metrics) IncrementParseErrors() {
	m.parseErrors.Add(1)
	m.errorsTotal.Add(1)
}

func (m *Metrics) IncrementHandlerErrors() {
	m.handlerErrors.Add(1)
	m.errorsTotal.Add(1)
}

func (m *Metrics) IncrementHandlerPanics() {
	m.handlerPanics.Add(1)
	m.errorsTotal.Add(1)
}

// StartMetricsServer starts an HTTP server for metrics endpoint
func StartMetricsServer(addr string) error {
	http.HandleFunc("/metrics", MetricsHandler)
	http.HandleFunc("/health", HealthHandler)

	Info("Metrics server starting", zap.String("address", addr))
	return http.ListenAndServe(addr, nil)
}

// MetricsHandler exposes metrics as JSON
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := GetMetrics()
	snapshot := metrics.Snapshot()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snapshot); err != nil {
		Error("Failed to encode metrics", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HealthHandler provides basic health check endpoint
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := GetMetrics()
	snapshot := metrics.Snapshot()

	health := map[string]interface{}{
		"status":            "healthy",
		"uptime":            snapshot.Uptime,
		"connected_clients": snapshot.ConnectedClients,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		Error("Failed to encode health", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
