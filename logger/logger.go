package logger

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jbenet/go-base58"
	"github.com/republicprotocol/go-do"
)

var defaultLoggerMu = new(sync.RWMutex)

// DefaultLogger is used when no Logger object is used to log events.
var DefaultLogger = func() *Logger {
	logger, err := NewLogger(Options{
		Plugins: []PluginOptions{
			PluginOptions{File: &FilePluginOptions{Path: "stdout"}, WebSocket: nil},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("cannot init default logger: %v", err))
	}
	logger.Start()
	return logger
}()

// SetDefaultLogger to a specific Logger object. The previous DefaultLogger
// will be stopped, and the new DefaultLogger will be started.
func SetDefaultLogger(logger *Logger) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Lock()
	DefaultLogger.Stop()
	DefaultLogger = logger
}

// LogInfo logs an info Log using a GenericEvent using the DefaultLogger.
func LogInfo(message string) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	DefaultLogger.Info(message)
}

// LogWarn logs a warn Log using a GenericEvent using the DefaultLogger.
func LogWarn(message string) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	DefaultLogger.Warn(message)
}

// LogError logs an error Log using a GenericEvent using the DefaultLogger.
func LogError(message string) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	DefaultLogger.Error(message)
}

// LogUsage logs an info Log using a UsageEvent using the DefaultLogger.
func LogUsage(cpu, memory float64, network uint64) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	DefaultLogger.Usage(cpu, memory, network)
}

// LogOrderConfirmed logs an OrderConfirmedEvent using the DefaultLogger.
func LogOrderConfirmed(ty Level, orderID string) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	DefaultLogger.OrderConfirmed(ty, orderID)
}

// LogOrderMatch logs an OrderMatchEvent using the DefaultLogger.
func LogOrderMatch(ty Level, id, buyID, sellID string) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	DefaultLogger.OrderMatch(ty, id, buyID, sellID)
}

// LogBuyOrderReceived logs an OrderReceivedEvent using the DefaultLogger.
func LogBuyOrderReceived(ty Level, id, fragmentID string) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	DefaultLogger.BuyOrderReceived(ty, id, fragmentID)
}

// LogSellOrderReceived logs an OrderReceivedEvent using the DefaultLogger.
func LogSellOrderReceived(ty Level, id, fragmentID string) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	DefaultLogger.SellOrderReceived(ty, id, fragmentID)
}

// LogNetwork logs a NetworkEvent using the DefaultLogger.
func LogNetwork(ty Level, message string) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	DefaultLogger.Network(ty, message)
}

// LogCompute logs a ComputeEvent using the DefaultLogger.
func LogCompute(ty Level, message string) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	DefaultLogger.Compute(ty, message)
}

// Logger handles distributing logs to plugins registered with it
type Logger struct {
	do.GuardedObject
	Plugins []Plugin
	Tags    map[string]string
}

// Options are used to Unmarshal a Logger from JSON.
type Options struct {
	Plugins []PluginOptions   `json:"plugins"`
	Tags    map[string]string `json:"tags"`
}

// The Plugin interface describes a worker that consumes logs
type Plugin interface {
	Start() error
	Stop() error
	Log(log Log) error
}

// PluginOptions are used to Unmarshal plugins from JSON.
type PluginOptions struct {
	File      *FilePluginOptions      `json:"file,omitempty"`
	WebSocket *WebSocketPluginOptions `json:"websocket,omitempty"`
}

// NewLogger returns a new Logger that will start and stop a set of plugins.
func NewLogger(options Options) (*Logger, error) {
	logger := &Logger{
		GuardedObject: do.NewGuardedObject(),
		Plugins:       make([]Plugin, 0, len(options.Plugins)),
		Tags:          options.Tags,
	}
	for i := range options.Plugins {
		if options.Plugins[i].File != nil {
			plugin, err := NewFilePlugin(*options.Plugins[i].File)
			if err != nil {
				return nil, err
			}
			logger.Plugins = append(logger.Plugins, plugin)
		}
		if options.Plugins[i].WebSocket != nil {
			plugin := NewWebSocketPlugin(logger, *options.Plugins[i].WebSocket)
			logger.Plugins = append(logger.Plugins, plugin)
		}
	}
	return logger, nil
}

// Start starts all the plugins of the logger
func (logger *Logger) Start() {
	for _, plugin := range logger.Plugins {
		if err := plugin.Start(); err != nil {
			log.Println(err)
		}
	}
}

// Stop stops all the plugins of the logger
func (logger Logger) Stop() {
	for _, plugin := range logger.Plugins {
		if err := plugin.Stop(); err != nil {
			log.Println(err)
		}
	}
}

// Log an Event.
func (logger *Logger) Log(l Log) {
	l.Tags = logger.Tags
	for _, plugin := range logger.Plugins {
		if err := plugin.Log(l); err != nil {
			log.Println(err)
		}
	}
}

// Info logs an info Log using a GenericEvent.
func (logger *Logger) Info(message string) {
	logger.Log(Log{
		Timestamp: time.Now(),
		Type:      LevelInfo,
		EventType: Generic,
		Event: GenericEvent{
			Message: message,
		},
	})
}

// Warn logs a warn Log using a GenericEvent.
func (logger *Logger) Warn(message string) {
	logger.Log(Log{
		Timestamp: time.Now(),
		Type:      Warn,
		EventType: Generic,
		Event: GenericEvent{
			Message: message,
		},
	})
}

// Error logs an error Log using a GenericEvent.
func (logger *Logger) Error(message string) {
	logger.Log(Log{
		Timestamp: time.Now(),
		Type:      Error,
		EventType: Generic,
		Event: GenericEvent{
			Message: message,
		},
	})
}

// Usage logs an info Log using a UsageEvent.
func (logger *Logger) Usage(cpu, memory float64, network uint64) {
	logger.Log(Log{
		Timestamp: time.Now(),
		Type:      LevelInfo,
		EventType: Usage,
		Event: UsageEvent{
			CPU:     cpu,
			Memory:  memory,
			Network: network,
		},
	})
}

// OrderConfirmed logs an OrderConfirmedEvent.
func (logger *Logger) OrderConfirmed(ty Level, orderID string) {
	logger.Log(Log{
		Timestamp: time.Now(),
		Type:      ty,
		EventType: OrderConfirmed,
		Event: OrderConfirmedEvent{
			OrderID: orderID,
		},
	})
}

// OrderMatch logs an OrderMatchEvent.
func (logger *Logger) OrderMatch(ty Level, id, buyID, sellID string) {
	logger.Log(Log{
		Timestamp: time.Now(),
		Type:      ty,
		EventType: OrderMatch,
		Event: OrderMatchEvent{
			ID:     id,
			BuyID:  buyID,
			SellID: sellID,
		},
	})
}

// BuyOrderReceived logs an OrderReceivedEvent.
func (logger *Logger) BuyOrderReceived(ty Level, id, fragmentID string) {
	logger.Log(Log{
		Timestamp: time.Now(),
		Type:      ty,
		EventType: OrderReceived,
		Event: OrderReceivedEvent{
			BuyID:      &id,
			FragmentID: fragmentID,
		},
	})
}

// SellOrderReceived logs an OrderReceivedEvent.
func (logger *Logger) SellOrderReceived(ty Level, id, fragmentID string) {
	logger.Log(Log{
		Timestamp: time.Now(),
		Type:      ty,
		EventType: OrderReceived,
		Event: OrderReceivedEvent{
			SellID:     &id,
			FragmentID: fragmentID,
		},
	})
}

// Network logs a NetworkEvent.
func (logger *Logger) Network(ty Level, message string) {
	logger.Log(Log{
		Timestamp: time.Now(),
		Type:      ty,
		EventType: Network,
		Event: NetworkEvent{
			Message: message,
		},
	})
}

// Compute logs a ComputeEvent.
func (logger *Logger) Compute(ty Level, message string) {
	logger.Log(Log{
		Timestamp: time.Now(),
		Type:      ty,
		EventType: Compute,
		Event: ComputeEvent{
			Message: message,
		},
	})
}

// Level defines the different levels of Log messages that can be sent.
type Level string

// Values for the LogType.
const (
	LevelInfo = Level("info")
	Warn      = Level("warn")
	Error     = Level("error")
)

// EventType defines the different types of Event messages that can be sent in a
// Log.
type EventType string

// Values for the EventType.
const (
	Generic        = EventType("generic")
	Epoch          = EventType("epoch")
	Usage          = EventType("usage")
	Ethereum       = EventType("ethereum")
	OrderConfirmed = EventType("orderConfirmed")
	OrderMatch     = EventType("orderMatch")
	OrderReceived  = EventType("orderReceived")
	Network        = EventType("network")
	Compute        = EventType("compute")
)

// A Log is logged by the Logger using all available Plugins.
type Log struct {
	Timestamp time.Time         `json:"timestamp"`
	Type      Level             `json:"type"`
	EventType EventType         `json:"eventType"`
	Event     Event             `json:"event"`
	Tags      map[string]string `json:"tags"`
}

// The Event interface describes a log event
type Event interface {
	String() string
}

// A GenericEvent logs a string
type GenericEvent struct {
	Message string `json:"message"`
}

func (event GenericEvent) String() string {
	return event.Message
}

// An EpochEvent logs that an epoch transition has been observed
type EpochEvent struct {
	Hash []byte `json:"hash"`
}

func (event EpochEvent) String() string {
	return base58.Encode(event.Hash)
}

// UsageEvent logs CPU, Memory and Network usage
type UsageEvent struct {
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Network uint64  `json:"network"`
}

func (event UsageEvent) String() string {
	return fmt.Sprintf("cpu = %v; memory = %v; network = %v", event.CPU, event.Memory, event.Network)
}

// OrderConfirmedEvent logs two confirmed orders
type OrderConfirmedEvent struct {
	OrderID string `json:"orderId"`
}

func (event OrderConfirmedEvent) String() string {
	return fmt.Sprintf("confirmation = %s", event.OrderID)
}

// OrderMatchEvent logs two matched orders
type OrderMatchEvent struct {
	ID     string `json:"id"`
	BuyID  string `json:"buyId"`
	SellID string `json:"sellId"`
}

func (event OrderMatchEvent) String() string {
	return fmt.Sprintf("buy = %s; sell = %s", event.BuyID, event.SellID)
}

// OrderReceivedEvent logs a newly received buy or sell fragment
type OrderReceivedEvent struct {
	BuyID      *string `json:"buyId,omitempty"`
	SellID     *string `json:"sellId,omitempty"`
	FragmentID string  `json:"fragmentId"`
}

func (event OrderReceivedEvent) String() string {
	if event.BuyID != nil {
		return "buy = " + *event.BuyID
	}
	if event.SellID != nil {
		return "sell = " + *event.SellID
	}
	return ""
}

// NetworkEvent logs a generic network-related message
type NetworkEvent struct {
	Message string `json:"message"`
}

func (event NetworkEvent) String() string {
	return event.Message
}

// ComputeEvent logs a generic compute-related message
type ComputeEvent struct {
	Message string `json:"message"`
}

func (event ComputeEvent) String() string {
	return event.Message
}
