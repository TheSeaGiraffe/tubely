package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		errMsg := "The uploaded file is too big. Please select a file that is 1 MB or less"
		respondWithError(w, http.StatusRequestEntityTooLarge, errMsg, err)
		return
	}

	thumbnailFileUpload, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer thumbnailFileUpload.Close()

	fileMediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing Content-Type header value", err)
		return
	}
	if !(fileMediaType == "image/jpeg" || fileMediaType == "image/png") {
		respondWithError(w, http.StatusBadRequest, "Uploaded image file does not have the correct media type", err)
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

	imgFileExt, err := mime.ExtensionsByType(fileMediaType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Not a valid MIME type", err)
		return
	}
	if len(imgFileExt) == 0 {
		respondWithError(w, http.StatusInternalServerError, "Could not find extension for given MIME type", err)
		return
	}

	randomBytes := make([]byte, 32)
	_, err = rand.Read(randomBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create random bytes", err)
		return
	}
	thumbnailID := base64.RawURLEncoding.EncodeToString(randomBytes)

	thumbnailFilename := fmt.Sprintf("%s%s", thumbnailID, imgFileExt[0])
	thumbnailFile, err := os.Create(filepath.Join(cfg.assetsRoot, thumbnailFilename))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create thumbnail file on internal filesystem", err)
		return
	}
	defer thumbnailFile.Close()

	_, err = io.Copy(thumbnailFile, thumbnailFileUpload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not copy uploaded thumbnail to internal filesystem", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, thumbnailFilename)
	video.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video info", err)
		return
	}

	// Keeping this for debugging purposes. Will need to remember to set this to DEBUG
	// level in the future.
	log.Println("uploaded thumbnail for video", videoID, "by user", userID)

	respondWithJSON(w, http.StatusOK, video)
}
