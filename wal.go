//It implements a Write-Ahead Log (WAL) for a key-value store. The WAL records operations (like setting or deleting keys) to a file, allowing the store to recover its state after a crash or restart by replaying the logged operations.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type WALRecord struct {
	Operation  string `json:"operation"`
	Key        string `json:"key"`
	Value      string `json:"value"`
	Expiration int64  `json:"expiration"`
}

type WAL struct {
	file *os.File
}

func OpenWAL(filename string) (*WAL, error) {
	file, err := os.OpenFile(
		filename,
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		0644,
	)

	if err != nil {
		return nil, err
	}

	return &WAL{
		file: file,
	}, nil
}

func (w *WAL) Append(record WALRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	_, err = w.file.Write(data)
	if err != nil {
		return err
	}

	// Make sure the data is flushed to durable storage.
	return w.file.Sync()
}

func (w *WAL) Replay(store *Store) error {
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	reader := bufio.NewReader(w.file)

	for {
		line, err := reader.ReadBytes('\n')

		if err == io.EOF {
			// Normal end of WAL.
			break
		}

		if err != nil {
			return err
		}

		if len(line) == 0 {
			continue
		}

		var record WALRecord

		if err := json.Unmarshal(line, &record); err != nil {
			return fmt.Errorf("corrupt WAL record: %w", err)
		}

		switch record.Operation {

		case "SET":
			store.applySet(
				record.Key,
				record.Value,
				record.Expiration,
			)

		case "DEL":
			store.applyDelete(record.Key)

		default:
			return fmt.Errorf("unknown WAL operation: %s", record.Operation)
		}
	}

	// Move back to the end so new records are appended correctly.
	_, err := w.file.Seek(0, io.SeekEnd)

	return err
}

func (w *WAL) Close() error {
	return w.file.Close()
}