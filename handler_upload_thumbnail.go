package main

import (
	"fmt"
	"io"
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

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
	thumbnailFilename := fmt.Sprintf("%s%s", videoID, imgFileExt[0])
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

	respondWithJSON(w, http.StatusOK, video)
}
