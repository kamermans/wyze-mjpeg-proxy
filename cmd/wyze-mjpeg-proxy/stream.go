package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	boundary = "ffmpeg"
)

type Streamer struct {
	Name       string
	Source     string
	Quality    string
	Resolution string
	Framerate  string
	FrameChan  chan []byte
	LastFrame  []byte

	numClients int
	mu         sync.RWMutex
	clientMu   sync.Mutex
	frameDelay time.Duration
}

func NewStreamer(stream StreamConfig) *Streamer {
	return &Streamer{
		Name:       stream.Name,
		Source:     stream.Source,
		Quality:    qualityToFFmpeg(stream.Quality),
		Resolution: strings.Replace(stream.Resolution, "x", ":", 1),
		Framerate:  fmt.Sprintf("%d", stream.Framerate),
		FrameChan:  make(chan []byte, 1),
		frameDelay: time.Second / time.Duration(stream.Framerate),
	}
}

const restartInterval = 10 * time.Second

func (s *Streamer) Start() {
	go func() {
		for {
			s.runStreamer()
			fmt.Printf("Restarting %s stream in %v seconds\n", s.Name, restartInterval)
			time.Sleep(restartInterval)
		}
	}()
}

func (s *Streamer) runStreamer() {

	defer fmt.Printf("ffmpeg exited, stopping %s stream\n", s.Name)

	// Read from source
	// ffmpeg -i "rtsp://192.168.4.66:8554/mercury-1-camera" -c:v mjpeg -q:v 1 -f mpjpeg -an -
	cmd := exec.Command(
		"ffmpeg",
		"-rtsp_transport", "tcp",
		"-i", s.Source,
		"-c:v", "mjpeg",
		"-q:v", s.Quality,
		"-vf", "scale="+s.Resolution,
		"-r", s.Framerate,
		"-f", "mpjpeg",
		"-an",
		"-",
	)
	// cmd := exec.Command("ffmpeg", "-rtsp_transport", "tcp", "-i", s.Source, "-r", "15", "-f", "mjpeg", "-")

	if verbose {
		fmt.Printf("Running command: %v\n", cmd.Args)
	}

	// Read from stdout and write to StreamBuffer
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Failed to get stdout pipe for %s: %v\n", s.Name, err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start ffmpeg for %s: %v\n", s.Name, err)
		return
	}

	fmt.Printf("Started %s stream\n", s.Name)

	// Create a new multipart reader to parse the MJPEG stream.
	multipartReader := multipart.NewReader(stdout, boundary)

	rateMu := sync.Mutex{}
	bytesRead := 0

	if verbose {
		rateReportTimer := time.NewTicker(5 * time.Second)
		defer rateReportTimer.Stop()

		go func() {
			lastTime := time.Now()

			for range rateReportTimer.C {
				elapsed := time.Since(lastTime)
				lastTime = time.Now()

				rateMu.Lock()
				// Calculate the stream rate in bytes per second.
				rate := float64(bytesRead) / elapsed.Seconds() / 1024
				bytesRead = 0
				rateMu.Unlock()

				fmt.Printf("Stream rate for %s: %.2f KB/s\n", s.Name, rate)
			}
		}()
	}

	// Loop over each part (frame) in the MJPEG stream.
	for {
		part, err := multipartReader.NextPart()
		if err == io.EOF {
			// End of the stream.
			break
		}
		if err != nil {
			fmt.Printf("Failed to read next part: %v\n", err)
			break
		}

		// Read the frame data from the current part.
		frameData, err := io.ReadAll(part)
		if err != nil {
			fmt.Printf("Failed to read frame data: %v\n", err)
			break
		}

		rateMu.Lock()
		bytesRead += len(frameData)
		rateMu.Unlock()

		// if verbose {
		// 	fmt.Printf("Read frame data: %v B\n", len(frameData))
		// }

		s.mu.Lock()
		s.LastFrame = frameData
		s.mu.Unlock()

		s.clientMu.Lock()
		clients := s.numClients
		s.clientMu.Unlock()

		if clients > 0 {
			// Send the frame data to the frameChan channel without blocking.
			select {
			case s.FrameChan <- frameData:
			default:
				// Drop the frame if the channel is full.
				if verbose {
					fmt.Printf("Dropped frame for %s: channel full\n", s.Name)
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		fmt.Printf("Failed to wait for ffmpeg for %s: %v\n", s.Name, err)
		return
	}
}

func (s *Streamer) StreamToClient(w http.ResponseWriter, r *http.Request) {
	s.clientMu.Lock()
	s.numClients++
	s.clientMu.Unlock()

	defer func() {
		s.clientMu.Lock()
		s.numClients--
		s.clientMu.Unlock()
	}()

	ctx := r.Context()

	writer := multipart.NewWriter(w)
	writer.SetBoundary(boundary)

	defer writer.Close()
	header := textproto.MIMEHeader{
		"Content-Type": []string{"image/jpeg"},
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Client %s disconnected from %s\n", r.RemoteAddr, s.Name)
			return
		case frame := <-s.FrameChan:
			if veryVerbose {
				fmt.Printf("Sending frame to %s client: %v B\n", s.Name, len(frame))
			}

			header.Set("Content-Length", strconv.Itoa(len(frame)))
			part, err := writer.CreatePart(header)

			if err != nil {
				fmt.Printf("Failed to create part: %v\n", err)
				return
			}

			if _, err := part.Write(frame); err != nil {
				fmt.Printf("Failed to write frame to client: %v\n", err)
				return
			}

			// Flush the buffer to the client.
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		}

	}
}

func (s *Streamer) ImageToClient(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()

	if len(s.LastFrame) == 0 {
		http.Error(w, "No frame available", http.StatusServiceUnavailable)
		s.mu.RUnlock()
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(s.LastFrame)))
	w.Write(s.LastFrame)
	s.mu.RUnlock()

	// Rate limit clients so they can't request images faster than the desired framerate.
	time.Sleep(s.frameDelay)
}

func (s *Streamer) StreamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace;boundary=ffmpeg")
	w.Header().Set("Cache-Control", "no-cache")
	s.StreamToClient(w, r)
}

func (s *Streamer) ImageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-cache")
	s.ImageToClient(w, r)
}

// qualityToFFmpeg converts the quality value (percentage from 1 to 100) to an ffmpeg compression string
func qualityToFFmpeg(quality int) string {
	if quality < 1 {
		quality = 1
	} else if quality > 100 {
		quality = 100
	}

	// Convert the quality percentage 1-100 to the inverted compression value 1-31
	// 1 is the highest quality (least compression), 31 is the lowest quality (most compression)
	// The formula is: quality = 100 - (compression * 3.225806451612903)
	compression := 100 - (float64(quality) * 3.225806451612903)

	return fmt.Sprintf("%.0f", compression)
}
