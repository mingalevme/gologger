package gologger

import (
	"net/http"
)

type Fields map[string]interface{}

type Clonable interface {
	Clone() interface{}
}

func (f Fields) Clone() Fields {
	if f == nil {
		return nil
	}
	clone := Fields{}
	for key, value := range f {
		if clonable, ok := value.(Clonable); ok {
			clone[key] = clonable.Clone()
		} else {
			clone[key] = value
		}
	}
	return clone
}

type Logger interface {
	WithField(key string, value interface{}) Logger
	WithFields(fields Fields) Logger
	WithError(err error) Logger
	WithRequest(req *http.Request) Logger

	Log(level Level, args ...interface{})

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})

	Debug(args ...interface{})
	Info(args ...interface{})
	Warning(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})

	Close()
}
