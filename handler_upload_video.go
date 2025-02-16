package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const hexStringLength = 32

func generateHexFileName(strLen int, aspectRatio string) (string, error) {
	randomBytes := make([]byte, strLen)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	fileName := hex.EncodeToString(randomBytes)

	switch aspectRatio {
	case "16:9":
		fileName = fmt.Sprintf("landscape/%s", fileName)
	case "9:16":
		fileName = fmt.Sprintf("portrait/%s", fileName)
	default:
		fileName = fmt.Sprintf("other/%s", fileName)
	}

	return fileName + ".mp4", nil
}

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving video info", err)
		return
	}
	if video.UserID != userID {
		videoErr := fmt.Errorf("Attempt to access video belonging to user '%v' by user '%v'", video.UserID, userID)
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to the current user", videoErr)
		return
	}

	const maxMemory = 10 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		errMsg := "The uploaded file is too big. Please select a file that is 1 GB or less"
		respondWithError(w, http.StatusRequestEntityTooLarge, errMsg, err)
		return
	}

	videoFileUpload, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer videoFileUpload.Close()

	fileMediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing Content-Type header value", err)
		return
	}
	if fileMediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Uploaded video file does not have the correct media type", err)
		return
	}

	tmpVideoFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create temp video file", err)
		return
	}
	defer func() {
		err = os.Remove(tmpVideoFile.Name())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not remove temp video file", err)
			return
		}
	}()
	defer tmpVideoFile.Close()

	_, err = io.Copy(tmpVideoFile, videoFileUpload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not copy uploaded video to internal filesystem", err)
		return
	}

	processedVideoFilePath, err := processVideoForFastStart(tmpVideoFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not modify uploaded video for faststarting", err)
		return
	}

	fileAspectRatio, err := getVideoAspectRatio(processedVideoFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not get video aspect ratio", err)
		return
	}

	fileName, err := generateHexFileName(hexStringLength, fileAspectRatio)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create hex string for video file name", err)
		return
	}

	processedVideoFile, err := os.Open(processedVideoFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not open processed video file", err)
		return
	}
	defer func() {
		err = os.Remove(processedVideoFilePath)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not remove processed video file", err)
			return
		}
	}()
	defer processedVideoFile.Close()

	s3ObjectInput := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileName,
		ContentType: &fileMediaType,
		Body:        processedVideoFile,
	}

	_, err = cfg.s3Client.PutObject(context.TODO(), &s3ObjectInput)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Problem uploading video file to S3 bucket", err)
		return
	}

	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, fileName)
	video.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video info", err)
		return
	}

	// Keeping this for debugging purposes. Will need to remember to set this to DEBUG
	// level in the future.
	log.Println("uploaded video with ID", videoID, "by user", userID)

	respondWithJSON(w, http.StatusOK, video)
}
