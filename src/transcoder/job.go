package transcoder

import (
	"bufio"
	"context"
	"fmt"
	"net/textproto"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	"github.com/AdmiralBulldogTv/VodTranscoder/src/global"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/structures"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/svc/mongo"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
)

type Job struct {
	structures.VodTranscodeJob
}

func (j *Job) Process(gCtx global.Context) (ret bool) {
	ret = true

	localLog := logrus.WithField("vod_id", j.VodID.Hex())
	localLog.Info("new job: ", j.Variant)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	filePath := path.Join(gCtx.Config().Transcode.ReadPath, j.VodID.Hex()+".flv")
	outFolder := path.Join(gCtx.Config().Transcode.WritePath, j.VodID.Hex(), j.Variant.Name)
	defer func() {
		if ret {
			if err := os.RemoveAll(outFolder); err != nil {
				localLog.Errorf("failed to cleanup %s directory: %s", outFolder, err.Error())
			}
		} else {
			if _, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameVods).UpdateOne(context.Background(), bson.M{
				"_id":           j.VodID,
				"variants.name": j.Variant.Name,
			}, bson.M{
				"$set": bson.M{
					"variants.$.ready": true,
				},
			}); err != nil {
				localLog.Errorf("failed to update mongo: %s", err.Error())
			}
		}

		respErr := ""
		if ret {
			respErr = "failed to transcode " + j.Variant.Name
		}

		body, _ := json.Marshal(structures.ApiTranscodeUpdate{
			VodID:   j.VodID,
			Variant: j.Variant,
			Error:   respErr,
		})

		if err := gCtx.Inst().RMQ.Publish(gCtx.Config().RMQ.ApiTaskQueue, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		}); err != nil {
			localLog.Errorf("failed to publish rmq: %s", err.Error())
		}
	}()

	if err := os.MkdirAll(outFolder, 0600); err != nil {
		localLog.Errorf("failed to create directory %s: %s", outFolder, err)
		return
	}

	var ffmpegCmd *exec.Cmd

	if j.Variant.Name == "source" {
		// we never transcode source
		ffmpegCmd = exec.CommandContext(ctx, "ffmpeg",
			"-i", filePath,
			"-c", "copy",
			"-hls_time", "5",
			"-hls_playlist_type", "vod",
			"-hls_allow_cache", "1", // allow cache of hls segments
			"-hls_segment_filename", path.Join(outFolder, "%04d.ts"),
			"-g", fmt.Sprint(j.Variant.FPS*5), // 5 second hls segments so each gop must be 5 x fps
			path.Join(outFolder, "playlist.m3u8"),
		)
	} else {
		ffmpegCmd = exec.CommandContext(ctx, "ffmpeg",
			"-i", filePath,
			"-c:v", "libx264",
			"-c:a", "aac",
			"-hls_time", "5",
			"-hls_playlist_type", "vod",
			"-hls_allow_cache", "1", // allow cache of hls segments
			"-hls_segment_filename", path.Join(outFolder, "%04d.ts"),
			"-g", fmt.Sprint(j.Variant.FPS*5), // 5 second hls segments so each gop must be 5 x fps
			"-r", fmt.Sprint(j.Variant.FPS),
			"-vf", fmt.Sprintf("scale=%d:%d", j.Variant.Width, j.Variant.Height),
			"-b:v", fmt.Sprint(j.Variant.Bitrate),
			path.Join(outFolder, "playlist.m3u8"),
		)
	}

	ffmpegCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}

	chErr := make(chan error)
	stdErr, _ := ffmpegCmd.StderrPipe()
	defer stdErr.Close()
	reader := textproto.NewReader(bufio.NewReader(stdErr))
	go func() {
		for {
			line, err := reader.ReadLine()
			localLog.Debug("ffmpeg output: ", line)
			if err != nil {
				return
			}
		}
	}()
	go func() {
		chErr <- ffmpegCmd.Run()
	}()

	select {
	case err := <-chErr:
		ret = err != nil
	case <-gCtx.Done():
		_ = ffmpegCmd.Process.Signal(syscall.SIGINT)
		select {
		case <-chErr:
		case <-time.After(time.Second * 15):
			cancel()
			<-chErr
		}
		return
	}

	return
}
