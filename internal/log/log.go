package log

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var (
	logger      zerolog.Logger
	loggerLevel = map[string]zerolog.Level{
		zerolog.TraceLevel.String(): zerolog.TraceLevel,
		zerolog.DebugLevel.String(): zerolog.DebugLevel,
		zerolog.InfoLevel.String():  zerolog.InfoLevel,
		zerolog.ErrorLevel.String(): zerolog.ErrorLevel,
	}
)

func Initialize() {
	// Obter o nível de log da variável de ambiente
	envLogLevel := os.Getenv("LOG_LEVEL")
	
	// Se não estiver definido, usar as flags de linha de comando
	if envLogLevel == "" {
		loggingLevel := flag.String("logging_level", zerolog.InfoLevel.String(), "logging level [-logging_level=trace|debug|info|error] (default is info)")
		loggingFormat := flag.String("logging_format", "json", "logging format [-logging_format=standard|json] (default is json)")
		flag.Parse()
		
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		
		setLoggingFormat(*loggingFormat)
		setLoggingLevel(*loggingLevel)
	} else {
		// Usar o nível de log da variável de ambiente
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		
		// Formato padrão JSON se não for especificado de outra forma
		setLoggingFormat("json")
		setLoggingLevel(envLogLevel)
	}
}

func setLoggingFormat(loggingFormat string) {
	switch loggingFormat {
	case "json":
		logger = zerolog.New(os.Stderr).With().Logger()
		logger.Info().Str("format", "json").Msg("setting logging format")
	default:
		logger = zerolog.New(
			zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: time.RFC3339,
			},
		).With().Timestamp().Logger()
		logger.Info().Str("format", "standard").Msg("setting logging format")
	}
}

func setLoggingLevel(loggingLevel string) {
	// Converter para minúsculas para garantir compatibilidade
	loggingLevelLower := strings.ToLower(loggingLevel)
	
	level, ok := loggerLevel[loggingLevelLower]
	if ok {
		zerolog.SetGlobalLevel(level)
	} else {
		logger.Info().Str("loggingLevelName", loggingLevel).Msg("unknown logging level, using default")
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	logger.Info().Str("loggingLevel", zerolog.GlobalLevel().String()).Msg("logging level configured")

	if zerolog.GlobalLevel() == zerolog.DebugLevel {
		logger.Info().Msg("stack enabled for logging")
		logger = logger.With().Stack().Logger()
	}

	if zerolog.GlobalLevel() == zerolog.TraceLevel {
		logger.Info().Msg("caller enabled for logging")
		logger = logger.With().Stack().Caller().Logger()
	}
}

func Info() *zerolog.Event {
	return logger.Info()
}

func Warn() *zerolog.Event {
	return logger.Warn()
}

func Debug() *zerolog.Event {
	return logger.Debug()
}

func Trace() *zerolog.Event {
	return logger.Trace()
}

func Fatal() *zerolog.Event {
	return logger.Fatal()
}

func ErrorMessage(message string) *zerolog.Event {
	wrappedError := errors.New(message)
	return logger.Error().Err(wrappedError)
}

func Error(err error) *zerolog.Event {
	wrappedError := errors.Wrap(err, err.Error())
	return logger.Error().Err(wrappedError)
}
