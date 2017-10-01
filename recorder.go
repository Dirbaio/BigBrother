package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/Dirbaio/BigBrother/mpd"
	_ "github.com/lib/pq"
	fsnotify "gopkg.in/fsnotify.v1"
)

type Recorder struct {
	camera           *Camera
	periodId         int
	lastSegmentCount int
	ffmpeg           *exec.Cmd
	startTime        time.Time
	done             chan struct{}
}

func NewRecorder(camera *Camera) *Recorder {
	return &Recorder{
		camera: camera,
		done:   make(chan struct{}),
	}
}

func (this *Recorder) getPath() string {
	return fmt.Sprintf("%s/%d/%d", storagePath, this.camera.ID, this.periodId)
}

func (this *Recorder) getMpdPath() string {
	return this.getPath() + "/stream.mpd"
}

func (this *Recorder) readMpd() {
	m, err := mpd.ReadFromFile(this.getMpdPath())
	if err != nil {
		log.Printf("%v: Error reading MPD: %v", this.camera, err)
		return
	}

	rep := m.Periods[0].AdaptationSets[0].Representations[0]

	// TODO we probably only need to do this once
	_, err = db.Exec(`
		UPDATE period SET
		 	codecs=$2,
			width=$3,
			height=$4,
			frame_rate=$5,
			timescale=$6
		WHERE id=$1
		`,
		this.periodId, rep.Codecs, rep.Width, rep.Height, rep.FrameRate, rep.SegmentTemplate.Timescale,
	)

	if err != nil {
		log.Fatal(err)
	}

	offset := uint64(0)
	for i, segment := range rep.SegmentTemplate.SegmentTimeline.Segments {
		if i >= this.lastSegmentCount {
			log.Printf("%v: New segment: %d\n", this.camera, i)
			segmentTime := this.startTime.Add(time.Duration(int64(offset) * int64(1000000000) / int64(*rep.SegmentTemplate.Timescale)))
			_, err = db.Exec(`
				INSERT INTO segment (period_id, camera_id, off, len, time, index)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				this.periodId, this.camera.ID, offset, segment.Duration, segmentTime, i+1)
			if err != nil {
				log.Fatal(err)
			}
		}
		offset += segment.Duration
	}
	this.lastSegmentCount = len(rep.SegmentTemplate.SegmentTimeline.Segments)

}

func findCrlf(data []byte) int {
	for i, c := range data {
		if c == '\r' || c == '\n' {
			return i
		}
	}
	return -1
}
func crlfSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := findCrlf(data); i >= 0 {
		if i == 0 {
			return 1, nil, nil
		}
		// We have a full newline-terminated line.
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func relayLogs(pipe io.ReadCloser, prefix string) {
	in := bufio.NewScanner(pipe)
	in.Split(crlfSplit)

	for in.Scan() {
		log.Printf("%s: %s", prefix, in.Text())
	}
	if err := in.Err(); err != nil {
		log.Printf("%s: error %s", prefix, err)
	}
}

//ffmpeg
func (this *Recorder) startFfmpeg() {
	this.ffmpeg = exec.Command(
		"ffmpeg",
		"-nostats",
		"-rtsp_transport",
		"tcp",
		"-i",
		this.camera.Source,
		"-codec",
		"copy",
		"-f",
		"dash",
		this.getMpdPath(),
	)
	stdout, err := this.ffmpeg.StdoutPipe()
	if err != nil {
		log.Printf("%v: error setting up ffmpeg stdout: %v", this.camera, err)
	} else {
		go relayLogs(stdout, fmt.Sprintf("%v ffmpeg-out", this.camera))
	}
	stderr, err := this.ffmpeg.StderrPipe()
	if err != nil {
		log.Printf("%v: error setting up ffmpeg stderr: %v", this.camera, err)
	} else {
		go relayLogs(stderr, fmt.Sprintf("%v ffmpeg-err", this.camera))
	}

	err = this.ffmpeg.Start()
	if err != nil {
		log.Printf("%v: error starting ffmpeg: %v", this.camera, err)
		return
	}
	log.Printf("%v: Started ffmpeg", this.camera)
}

func (this *Recorder) stopFfmpeg() {
	this.ffmpeg.Process.Signal(os.Interrupt)
	log.Printf("%v: Waiting for ffmpeg to exit", this.camera)
	this.ffmpeg.Wait()
	log.Printf("%v: ffmpeg exited", this.camera)
}

func (this *Recorder) Stop() {
	close(this.done)
}

func (this *Recorder) Run() {
	log.Printf("%v: Starting", this.camera)

	this.startTime = time.Now().UTC()

	err := db.Get(&this.periodId, "INSERT INTO period (camera_id, time) VALUES($1, $2) RETURNING id", this.camera.ID, this.startTime)
	if err != nil {
		log.Fatal(err)
	}

	path := this.getPath()
	mpdPath := this.getMpdPath()
	os.MkdirAll(path, 0770)

	this.startFfmpeg()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("%v: file watch create error:", err)
	}
	defer watcher.Close()

	err = watcher.Add(path)
	if err != nil {
		log.Printf("%v: file watch add error:", err)
	}

loop:
	for {
		select {
		case event := <-watcher.Events:
			//log.Println("event:", event)
			if event.Op&fsnotify.Create == fsnotify.Create && event.Name == mpdPath {
				log.Printf("%v: mpd modified", this.camera)
				this.readMpd()
			}
		case err := <-watcher.Errors:
			log.Printf("%v: file watch error:", err)
		case <-this.done:
			log.Printf("%v: Stopping", this.camera)
			break loop
		}
	}

	this.stopFfmpeg()
	log.Printf("%v: Reading MPD for one last time", this.camera)
	this.readMpd()
	log.Printf("%v: Bye bye", this.camera)
}
