package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type FFProbeOutput struct {
	Streams []Streams `json:"streams"`
}

type Disposition struct {
	Default         int `json:"default"`
	Dub             int `json:"dub"`
	Original        int `json:"original"`
	Comment         int `json:"comment"`
	Lyrics          int `json:"lyrics"`
	Karaoke         int `json:"karaoke"`
	Forced          int `json:"forced"`
	HearingImpaired int `json:"hearing_impaired"`
	VisualImpaired  int `json:"visual_impaired"`
	CleanEffects    int `json:"clean_effects"`
	AttachedPic     int `json:"attached_pic"`
	TimedThumbnails int `json:"timed_thumbnails"`
	NonDiegetic     int `json:"non_diegetic"`
	Captions        int `json:"captions"`
	Descriptions    int `json:"descriptions"`
	Metadata        int `json:"metadata"`
	Dependent       int `json:"dependent"`
	StillImage      int `json:"still_image"`
	Multilayer      int `json:"multilayer"`
}

type Tags struct {
	Language    string `json:"language"`
	HandlerName string `json:"handler_name"`
	VendorID    string `json:"vendor_id"`
	Encoder     string `json:"encoder"`
	Timecode    string `json:"timecode"`
}

type Streams struct {
	Index              int         `json:"index"`
	CodecName          string      `json:"codec_name"`
	CodecLongName      string      `json:"codec_long_name"`
	Profile            string      `json:"profile"`
	CodecType          string      `json:"codec_type"`
	CodecTagString     string      `json:"codec_tag_string"`
	CodecTag           string      `json:"codec_tag"`
	Width              int         `json:"width"`
	Height             int         `json:"height"`
	CodedWidth         int         `json:"coded_width"`
	CodedHeight        int         `json:"coded_height"`
	ClosedCaptions     int         `json:"closed_captions"`
	FilmGrain          int         `json:"film_grain"`
	HasBFrames         int         `json:"has_b_frames"`
	SampleAspectRatio  string      `json:"sample_aspect_ratio"`
	DisplayAspectRatio string      `json:"display_aspect_ratio"`
	PixFmt             string      `json:"pix_fmt"`
	Level              int         `json:"level"`
	ColorRange         string      `json:"color_range"`
	ColorSpace         string      `json:"color_space"`
	ColorTransfer      string      `json:"color_transfer"`
	ColorPrimaries     string      `json:"color_primaries"`
	ChromaLocation     string      `json:"chroma_location"`
	FieldOrder         string      `json:"field_order"`
	Refs               int         `json:"refs"`
	IsAvc              string      `json:"is_avc"`
	NalLengthSize      string      `json:"nal_length_size"`
	ID                 string      `json:"id"`
	RFrameRate         string      `json:"r_frame_rate"`
	AvgFrameRate       string      `json:"avg_frame_rate"`
	TimeBase           string      `json:"time_base"`
	StartPts           int         `json:"start_pts"`
	StartTime          string      `json:"start_time"`
	DurationTs         int         `json:"duration_ts"`
	Duration           string      `json:"duration"`
	BitRate            string      `json:"bit_rate"`
	BitsPerRawSample   string      `json:"bits_per_raw_sample"`
	NbFrames           string      `json:"nb_frames"`
	ExtradataSize      int         `json:"extradata_size"`
	Disposition        Disposition `json:"disposition"`
	Tags               Tags        `json:"tags"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	args := []string{
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "v:0",
		filePath,
	}
	cmd := exec.Command("ffprobe", args...)
	cmdBuf := new(bytes.Buffer)
	cmd.Stdout = cmdBuf
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error running command: %w", err)
	}

	var ffprobeOut FFProbeOutput
	err = json.Unmarshal(cmdBuf.Bytes(), &ffprobeOut)
	if err != nil {
		return "", fmt.Errorf("Error unmarshaling command output: %w", err)
	}

	width := ffprobeOut.Streams[0].Width
	height := ffprobeOut.Streams[0].Height

	aspectRatio := float64(width) / float64(height)
	aspectRatioLandscape := 16.0 / 9.0
	tolerance := 0.5
	toleranceLandscape := aspectRatioLandscape * tolerance
	aspectRatioPortrait := 9.0 / 16.0
	tolerancePortrait := aspectRatioPortrait * tolerance

	var aspectRatioStr string
	if ((aspectRatioLandscape - toleranceLandscape) <= aspectRatio) && (aspectRatio <= (aspectRatioLandscape + tolerance)) {
		aspectRatioStr = "16:9"
	} else if ((aspectRatioPortrait - tolerancePortrait) <= aspectRatio) && (aspectRatio <= (aspectRatioPortrait + tolerance)) {
		aspectRatioStr = "9:16"
	} else {
		aspectRatioStr = "other"
	}

	return aspectRatioStr, nil
}
