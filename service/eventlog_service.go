package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"s2qt/util"
)

type RunLogSession struct {
	RunID      string
	SourceType string
	StartedAt  time.Time
}

func NewRunID() string {
	now := time.Now()
	return fmt.Sprintf("RUN-%s", now.Format("20060102-150405"))
}

func StartEventLog(sourceType string) (*RunLogSession, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	logDir := filepath.Join(paths.Var, "log")
	logFile := filepath.Join(logDir, "event.log")

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}

	runID := NewRunID()
	now := time.Now()

	f, err := os.Create(logFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	header := buildRunStartHeader(runID, sourceType, now)
	if _, err := f.WriteString(header); err != nil {
		return nil, err
	}

	return &RunLogSession{
		RunID:      runID,
		SourceType: sourceType,
		StartedAt:  now,
	}, nil
}

func LogInfo(message string) {
	_ = appendEventLog("INFO", message)
}

func LogWarn(message string) {
	_ = appendEventLog("WARN", message)
}

func LogError(message string) {
	_ = appendEventLog("ERROR", message)
}

func EndEventLog(status string) {
	status = strings.TrimSpace(strings.ToUpper(status))
	if status == "" {
		status = "COMPLETED"
	}
	_ = appendRunEnd(status)
}

func appendEventLog(level, message string) error {
	paths, err := util.GetAppPaths()
	if err != nil {
		return err
	}

	logFile := filepath.Join(paths.Var, "log", "event.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	line := fmt.Sprintf("%s [%s] %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		strings.ToUpper(strings.TrimSpace(level)),
		strings.TrimSpace(message),
	)

	_, err = f.WriteString(line)
	return err
}

func appendRunEnd(status string) error {
	paths, err := util.GetAppPaths()
	if err != nil {
		return err
	}

	logFile := filepath.Join(paths.Var, "log", "event.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	footer := buildRunEndFooter(status, time.Now())
	_, err = f.WriteString(footer)
	return err
}

func buildRunStartHeader(runID, sourceType string, t time.Time) string {
	return fmt.Sprintf(
		"==================================================\n"+
			"RUN START\n"+
			"run_id=%s\n"+
			"source_type=%s\n"+
			"started_at=%s\n"+
			"==================================================\n",
		strings.TrimSpace(runID),
		strings.TrimSpace(sourceType),
		t.Format("2006-01-02 15:04:05"),
	)
}

func buildRunEndFooter(status string, t time.Time) string {
	return fmt.Sprintf(
		"==================================================\n"+
			"RUN END\n"+
			"status=%s\n"+
			"ended_at=%s\n"+
			"==================================================\n",
		strings.TrimSpace(status),
		t.Format("2006-01-02 15:04:05"),
	)
}
