package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	const maxMemory = 10 << 20 // aparentemente esto ("<<") es bitshifting  10 << 20 == 10 * 1024 * 1024 == 10MB

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form", err)
		return 
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form", err)
		return 
	}

	defer file.Close()

	mediaTypeData := header.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(mediaTypeData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse MIME", err)
		return
	}

	if mediaType != "image/png" && mediaType != "image/jpeg" {
		respondWithError(w, http.StatusBadRequest, "Incorrect media type", err)
		return
	}

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get video metadata", err)
		return
	}

	if userID != videoData.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized, authentication required", err)
		return
	}

	mediaTypeAndExt := strings.Split(mediaTypeData, "/")

	thumbnailFilePath := filepath.Join(cfg.assetsRoot, videoIDString) + fmt.Sprintf(".%s", mediaTypeAndExt[1])

	newFile, err := os.Create(thumbnailFilePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create file", err)
		return
	}

	if _, err := io.Copy(newFile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to write contents to file", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%v/%v/%v", cfg.port, cfg.assetsRoot, videoIDString) + fmt.Sprintf(".%s", mediaTypeAndExt[1])

	videoData.ThumbnailURL = &thumbnailURL

	if err := cfg.db.UpdateVideo(videoData); err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoData)
}
