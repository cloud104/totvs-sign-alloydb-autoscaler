package log

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/rs/zerolog"
)

var (
	logger      zerolog.Logger
	loggerLevel = map[string]zerolog.Level{
		zerolog.TraceLevel.String(): zerolog.TraceLevel,
		zerolog.DebugLevel.String(): zerolog.DebugLevel,
		zerolog.InfoLevel.String():  zerolog.InfoLevel,
		zerolog.ErrorLevel.String(): zerolog.ErrorLevel,
	}

	// Ordem prioritária dos campos
	fieldOrder = []string{
		"timestamp",
		"level",
		"component",
		"action",
		"cycle",
		"instance",
		"cluster",
		"cpuUsage",
		"memoryUsage",
		"currentReplicas",
		"cpuThreshold",
		"memoryThreshold",
		"duration",
		"minReplicas",
		"maxReplicas",
		"nextCheckTime",
		"intervalSeconds",
		"evaluationPeriod",
		"scaleUpVotes",
		"scaleDownVotes",
		"message",
	}
)

type orderedJSONWriter struct {
	w io.Writer
}

func (w *orderedJSONWriter) Write(p []byte) (n int, err error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(p, &raw); err != nil {
		return w.w.Write(p)
	}

	buf := new(bytes.Buffer)
	buf.WriteString("{")

	// Escreve campos na ordem prioritária
	for i, key := range fieldOrder {
		if value, exists := raw[key]; exists {
			if i > 0 {
				buf.WriteString(",")
			}
			writeField(buf, key, value)
			delete(raw, key)
		}
	}

	// Escreve campos restantes em ordem alfabética
	remainingKeys := make([]string, 0, len(raw))
	for k := range raw {
		remainingKeys = append(remainingKeys, k)
	}
	sort.Strings(remainingKeys)

	for _, key := range remainingKeys {
		buf.WriteString(",")
		writeField(buf, key, raw[key])
	}

	buf.WriteString("}\n")
	return w.w.Write(buf.Bytes())
}

func writeField(buf *bytes.Buffer, key string, value interface{}) {
	keyEscaped, _ := json.Marshal(key)
	valueEscaped, _ := json.Marshal(value)
	buf.Write(keyEscaped)
	buf.WriteString(":")
	buf.Write(valueEscaped)
}

func Initialize() {
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05" // Formato sem fuso horário
	zerolog.TimestampFieldName = "timestamp"

	envLogLevel := os.Getenv("LOG_LEVEL")

	if envLogLevel == "" {
		loggingLevel := flag.String("logging_level", zerolog.InfoLevel.String(), "logging level")
		loggingFormat := flag.String("logging_format", "json", "logging format")
		flag.Parse()
		setLoggingFormat(*loggingFormat)
		setLoggingLevel(*loggingLevel)
	} else {
		setLoggingFormat("json")
		setLoggingLevel(envLogLevel)
	}
}

func setLoggingFormat(format string) {
	switch format {
	case "json":
		logger = zerolog.New(&orderedJSONWriter{w: os.Stderr}).
			With().
			Timestamp().
			Logger()
	default:
		logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006-01-02 15:04:05", // Formato para console
			PartsOrder: []string{"timestamp", "level", "message"},
		}).
			With().
			Timestamp().
			Logger()
	}
}

// Restante do código mantido igual...

func setLoggingLevel(level string) {
	logLevel := strings.ToLower(level)
	if lvl, ok := loggerLevel[logLevel]; ok {
		zerolog.SetGlobalLevel(lvl)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	logger.Info().
		Str("currentLevel", zerolog.GlobalLevel().String()).
		Msg("logging level set")

	switch zerolog.GlobalLevel() {
	case zerolog.TraceLevel:
		logger = logger.With().Caller().Stack().Logger()
		logger.Info().Msg("caller and stack tracing enabled")
	case zerolog.DebugLevel:
		logger = logger.With().Stack().Logger()
		logger.Info().Msg("stack tracing enabled")
	}
}

func Info() *zerolog.Event  { return logger.Info() }
func Warn() *zerolog.Event  { return logger.Warn() }
func Debug() *zerolog.Event { return logger.Debug() }
func Trace() *zerolog.Event { return logger.Trace() }
func Fatal() *zerolog.Event { return logger.Fatal() }

func Error(err error) *zerolog.Event {
	return logger.Error().Err(err)
}

func ErrorMessage(msg string) *zerolog.Event {
	return logger.Error().Str("error_message", msg)
}
