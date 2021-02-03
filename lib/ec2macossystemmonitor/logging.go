package ec2macossystemmonitor

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
)

// Logger contains booleans for where to log, a tag used in syslog and the syslog Writer itself.
type Logger struct {
	LogToStdout    bool
	LogToSystemLog bool
	Tag            string
	SystemLog      syslog.Writer
}

// defaultLogInterval is the number of writes before emitting a log entry 10 = once every 10 minutes
const DefaultLogInterval = 10

// StatusLogBuffer contains a message format string and a written bytes for this format string for flushing the logs
type StatusLogBuffer struct {
	Message string
	Written int64
}

// IntervalLogger is a special logger that provides a way to only log at a certain interval.
type IntervalLogger struct {
	logger      Logger
	LogInterval int
	Counter     int
	Message     string
}

// NewLogger creates a new logger.  Logger writes using the LOG_LOCAL0 facility by default if system logging is enabled.
func NewLogger(tag string, systemLog bool, stdout bool) (logger *Logger, err error) {
	// Set up system logging, if enabled
	syslogger := &syslog.Writer{}
	if systemLog {
		syslogger, err = syslog.New(syslog.LOG_LOCAL0, tag)
		if err != nil {
			return &Logger{}, fmt.Errorf("ec2macossystemmonitor: unable to create new syslog logger: %s\n", err)
		}
	}
	// Set log to use microseconds, if stdout is enabled
	if stdout {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	}

	return &Logger{LogToSystemLog: systemLog, LogToStdout: stdout, Tag: tag, SystemLog: *syslogger}, nil
}

// Info writes info to stdout and/or the system log.
func (l *Logger) Info(v ...interface{}) {
	if l.LogToStdout {
		log.Print(v...)
	}
	if l.LogToSystemLog {
		l.SystemLog.Info(fmt.Sprint(v...))
	}
}

// Infof writes formatted info to stdout and/or the system log.
func (l *Logger) Infof(format string, v ...interface{}) {
	if l.LogToStdout {
		log.Printf(format, v...)
	}
	if l.LogToSystemLog {
		l.SystemLog.Info(fmt.Sprintf(format, v...))
	}
}

// Warn writes a warning to stdout and/or the system log.
func (l *Logger) Warn(v ...interface{}) {
	if l.LogToStdout {
		log.Print(v...)
	}
	if l.LogToSystemLog {
		l.SystemLog.Warning(fmt.Sprint(v...))
	}
}

// Warnf writes a formatted warning to stdout and/or the system log.
func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.LogToStdout {
		log.Printf(format, v...)
	}
	if l.LogToSystemLog {
		l.SystemLog.Warning(fmt.Sprintf(format, v...))
	}
}

// Error writes an error to stdout and/or the system log.
func (l *Logger) Error(v ...interface{}) {
	if l.LogToStdout {
		log.Print(v...)
	}
	if l.LogToSystemLog {
		l.SystemLog.Err(fmt.Sprint(v...))
	}
}

// Errorf writes a formatted error to stdout and/or the system log.
func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.LogToStdout {
		log.Printf(format, v...)
	}
	if l.LogToSystemLog {
		l.SystemLog.Err(fmt.Sprintf(format, v...))
	}
}

// Fatal writes an error to stdout and/or the system log then exits 1.
func (l *Logger) Fatal(v ...interface{}) {
	l.Error(v...)
	os.Exit(1)
}

// Fatalf writes a formatted error to stdout and/or the system log then exits 1.
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Errorf(format, v...)
	os.Exit(1)
}

// PushToInterval adds to the counter and sets the Message, care should be taken to retrieve the Message before setting since
// its overwritten
func (t *IntervalLogger) PushToInterval(i int, message string) (flushed bool) {
	t.Counter = +i
	t.Message = message
	if t.Counter > t.LogInterval {
		t.logger.Info(message)
		t.Counter = 0
		return true
	}
	return false
}
