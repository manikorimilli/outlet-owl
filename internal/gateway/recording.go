package gateway

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// recording is one saved answer. Its key is the SHA-256 of the exact request
// body, which holds the model, the prompt text (so its version) and the user
// message (HLD section 5).
type recording struct {
	Key     string          `json:"key"`
	Purpose Purpose         `json:"purpose"`
	Prompt  recordingPrompt `json:"prompt"`
	// Request and Response are the raw bodies, kept as JSON.
	Request  json.RawMessage `json:"request"`
	Response json.RawMessage `json:"response"`
}

type recordingPrompt struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
}

func recordingKey(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (g *Gateway) recordingPath(p Purpose, key string) string {
	return filepath.Join(g.cfg.RecordingsDir, string(p), key+".json")
}

// replay answers from the recording of this exact request and writes no
// budget row (AC-US-02-003-1); a missing recording is an error, never a live
// call (tenet 5).
func (g *Gateway) replay(r Request, body []byte) (Response, error) {
	key := recordingKey(body)
	path := g.recordingPath(r.Purpose, key)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Response{}, &RecordingMissingError{Purpose: r.Purpose, Key: key, Path: path}
	}
	if err != nil {
		return Response{}, fmt.Errorf("gateway: read recording %s: %w", path, err)
	}
	var rec recording
	if err := json.Unmarshal(data, &rec); err != nil {
		return Response{}, fmt.Errorf("gateway: unreadable recording %s: %w", path, err)
	}
	// The file is indented for people to read; compare the compact bytes.
	var stored bytes.Buffer
	if err := json.Compact(&stored, rec.Request); err != nil || !bytes.Equal(stored.Bytes(), body) {
		return Response{}, fmt.Errorf("%w: %s", ErrRecordingMismatch, path)
	}
	p, err := parseChat(rec.Response)
	if err != nil {
		return Response{}, fmt.Errorf("gateway: recording %s: %w", path, err)
	}
	p.resp.Mode = Replay
	g.logCall(r, 0, "replayed", 200, "0.00000000", "0.00000000", p.resp, 0)
	return p.resp, nil
}

// save writes the recording of a paid answer: a temporary file, then a
// rename, so a crash never leaves half a recording.
func (g *Gateway) save(r Request, body, raw []byte) error {
	key := recordingKey(body)
	path := g.recordingPath(r.Purpose, key)
	data, err := json.MarshalIndent(recording{
		Key: key, Purpose: r.Purpose,
		Prompt:  recordingPrompt{Name: r.Prompt.Name, Version: r.Prompt.Number},
		Request: body, Response: raw,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrRecordingNotSaved, path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrRecordingNotSaved, path, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".recording-*")
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrRecordingNotSaved, path, err)
	}
	_, werr := tmp.Write(append(data, '\n'))
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(tmp.Name()) // best effort; the error below is what matters
		return fmt.Errorf("%w: %s: %v", ErrRecordingNotSaved, path, errors.Join(werr, cerr))
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("%w: %s: %v", ErrRecordingNotSaved, path, err)
	}
	return nil
}
