package gologger

import (
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"github.com/rollbar/rollbar-go"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

type Creator struct {
	Prefix string
	Env    map[string]string
}

func (c Creator) GetEnv(key string, def string) string {
	if c.Env == nil {
		return def
	}
	key = c.Prefix + key
	if val, ok := c.Env[key]; ok {
		return val
	} else {
		return def
	}
}

func (c Creator) Create() (Logger, error) {
	return c.NewLogChannel(c.GetEnv("LOG_CHANNEL", "stdout"))
}

func (c Creator) NewLogChannel(channel string) (Logger, error) {
	switch channel {
	case "stack":
		return c.NewStackLogger()
	case "stdout":
		return c.NewStdoutLogger()
	case "stderr":
		return c.NewStderrLogger()
	case "sentry":
		if h, err := c.NewSentryHub(); err != nil {
			return nil, err
		} else {
			return c.NewSentryLogger(h)
		}
	case "rollbar":
		if r, err := c.NewRollbar(); err != nil {
			return nil, err
		} else {
			return c.NewRollbarLogger(r)
		}
	case "array":
		return c.NewArrayLogger()
	case "null":
		return c.NewNullLogger(), nil
	default:
		panic(errors.Errorf("unsupported log channel: %s", channel))
	}
}

func (c Creator) NewStackLogger() (Logger, error) {
	logger := NewStackLogger()
	channels := strings.Split(c.GetEnv("LOG_STACK_CHANNELS", "stdout"), ",")
	for _, channel := range channels {
		channel = strings.TrimSpace(channel)
		if channel == "" {
			continue
		}
		if channel == "stack" {
			return nil, errors.Errorf("stack channel recursion")
		}
		if l, err := c.NewLogChannel(channel); err != nil {
			logger.Add(l)
		} else {
			return nil, errors.Wrap(err, "error while initialing logger: "+channel)
		}

	}
	return logger, nil
}

func (c Creator) NewStdoutLogger() (Logger, error) {
	if level, err := ParseLevel(c.GetEnv("LOG_STDOUT_LEVEL", "debug")); err != nil {
		return nil, errors.Wrap(err, "parsing stdout logging level")
	} else {
		return NewStdoutLogger(level), nil
	}
}

func (c Creator) NewStderrLogger() (Logger, error) {
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(os.Stderr)
	if level, err := logrus.ParseLevel(c.GetEnv("LOG_STDERR_LEVEL", "error")); err != nil {
		return nil, errors.Wrap(err, "parsing stderr logging level")
	} else {
		logrusLogger.SetLevel(level)
	}
	return NewLogrusLogger(logrusLogger), nil
}

func (c Creator) NewNullLogger() Logger {
	return NewNullLogger()
}

func (c Creator) NewArrayLogger() (Logger, error) {
	if lvl, err := ParseLevel(c.GetEnv("LOG_ARRAY_LEVEL", "debug")); err != nil {
		return nil, err
	} else {
		return NewArrayLogger(lvl), nil
	}
}

func (c Creator) NewSentryLogger(s *sentry.Hub) (Logger, error) {
	level, err := ParseLevel(c.GetEnv("LOG_SENTRY_LEVEL", LevelWarning.String()))
	if err != nil {
		return nil, errors.Wrap(err, "parsing sentry log level (LOG_SENTRY_LEVEL)")
	}
	return NewSentryLogger(s, level), nil
}

func (c Creator) NewSentryHub() (*sentry.Hub, error) {
	dsn := c.GetEnv("LOG_SENTRY_DSN", c.GetEnv("SENTRY_DSN", ""))
	if dsn == "" {
		return nil, errors.New("Missing LOG_SENTRY_DSN / SENTRY_DSN env var")
	}
	debug := strings.ToLower(strings.TrimSpace(c.GetEnv("LOG_SENTRY_DEBUG", "")))
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:         dsn,
		Debug:       debug == "" || debug == "0" || debug == "false",
		Environment: c.GetEnv("LOG_SENTRY_ENV", c.GetEnv("SENTRY_ENV", "production")),
	})
	if err != nil {
		return nil, err
	}
	s := sentry.NewHub(client, sentry.NewScope())
	s.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetFingerprint([]string{"{{ default }}", "{{ message }}", "{{ error.type }}", "{{ error.value }}"})
	})
	return s, nil
}

func (c Creator) NewRollbarLogger(r *rollbar.Client) (Logger, error) {
	level, err := ParseLevel(c.GetEnv("LOG_ROLLBAR_LEVEL", LevelWarning.String()))
	if err != nil {
		return nil, errors.Wrap(err, "parsing rollbar log level (LOG_ROLLBAR_LEVEL)")
	}
	return NewRollbarLogger(r, level), nil
}

func (c Creator) NewRollbar() (*rollbar.Client, error) {
	token := c.GetEnv("LOG_ROLLBAR_TOKEN", c.GetEnv("ROLLBAR_TOKEN", ""))
	if token == "" {
		return nil, errors.New("env var LOG_ROLLBAR_TOKEN / ROLLBAR_TOKEN is empty")
	}
	environmentID := c.GetEnv("LOG_ROLLBAR_ENV", c.GetEnv("ROLLBAR_ENV", "production"))
	r := rollbar.New(token, environmentID, "", "", "")
	r.SetFingerprint(true)
	return r, nil
}

func GetOSEnvMap() map[string]string {
	m := map[string]string{}
	for _, element := range os.Environ() {
		variable := strings.Split(element, "=")
		m[variable[0]] = variable[1]
	}
	return m
}
