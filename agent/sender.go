package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

func (agent *Agent) sendBatch(lines []LogLine) error {
	hostName, _ := os.Hostname()

	grouped := make(map[string][]string)
	for _, l := range lines {
		grouped[l.Source] = append(grouped[l.Source], l.Log)
	}

	for source, texts := range grouped {
		batch := LogBatch{
			Lines: texts,
			Metadata: map[string]string{
				"hostname": hostName,
				"version":  "0.1.0",
				"source":   source,
			},
		}

		data, err := msgpack.Marshal(&batch)
		if err != nil {
			log.Printf("failed to marshal batch: %v", err)
			return err
		}

		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write(data)
		if err := gz.Close(); err != nil {
			return err
		}

		payload := buf.Bytes()
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/logs", agent.serverUrl), bytes.NewReader(payload))

		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/msgpack")
		req.Header.Set("Content-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			log.Printf("failed to send batch: %v", err)
			return err
		}
		log.Printf("Sending batch for source %s", source)

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusServiceUnavailable {
			return fmt.Errorf("Server Overloaded: %d", resp.StatusCode)
		}

		if resp.StatusCode >= 500 {
			return fmt.Errorf("Server Error: %d", resp.StatusCode)
		}

		if resp.StatusCode >= 400 {
			return nil
		}
	}
	return nil
}

func (agent *Agent) sendWithRetry(lines []LogLine) {
	backoff := time.Second
	maxBackoff := 30 * backoff

	for {
		err := agent.sendBatch(lines)

		if err == nil {
			return
		}

		log.Printf("batch failed, retrying in %s: %v", backoff, err)
		time.Sleep(jitter(backoff))

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func jitter(backoff time.Duration) time.Duration {
	return time.Duration(rand.Int64N(int64(backoff)))
}
