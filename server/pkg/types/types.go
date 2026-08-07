package types

import "time"

type LogEntry struct {
	Timestamp time.Time
	Host      string
	Source    string
	Message   string
}

type LogRecord struct {
	Timestamp time.Time `parquet:"timestamp,timestamp(microsecond)"`
	Host      string    `parquet:"host,dict"`
	Source    string    `parquet:"source,dict"`
	Message   string    `parquet:"message,zstd"`
}

func ToLogRecord(log LogEntry) LogRecord {
	return LogRecord{
		Timestamp: log.Timestamp,
		Host:      log.Host,
		Source:    log.Source,
		Message:   log.Message,
	}
}
