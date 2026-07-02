package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1 << 30)

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
		respondWithError(w, http.StatusBadRequest, "Couldn't find the video", err)
		return 
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorize, authentication required", err)
		return
	}
	
	file, header, err := r.FormFile("video")
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

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Incorrect media type", err)
		return 
	}

	mediaTypeAndExt := strings.Split(mediaType, "/")

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create temp file", err)
		return
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to copy onto temp file", err)
		return
	}
	
	tempfilePath, err := filepath.Abs(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get tempFile path", err)
		return
	}

	var aspectName string

	aspectRatio, err :=  getVideoAspectRatio(tempfilePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get aspect ratio", err)
		return
	}

	switch aspectRatio {
	case "9:16":
		aspectName = "portrait"
	case "16:9":
		aspectName = "landscape"
	default:
		aspectName = "other"
	}


	if _, err := tempFile.Seek(0,io.SeekStart); err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to SeekStart temp file", err)
		return 
	}

	key := make([]byte, 32)
	rand.Read(key)
	encodedKey := fmt.Sprintf("%v/", aspectName) + hex.EncodeToString(key) + fmt.Sprintf(".%s", mediaTypeAndExt[1])

	newVideoURL := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v",cfg.s3Bucket, cfg.s3Region, encodedKey)

	_, err = cfg.s3Client.PutObject(
		context.Background(),
		&s3.PutObjectInput{
			Bucket: &cfg.s3Bucket,
			Key: &encodedKey,
			Body: tempFile,
			ContentType: &mediaType,
	})

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to put object in s3Bucket", err)
		return 
	}

	video.VideoURL = &newVideoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video info", err)
		return
	}
}
