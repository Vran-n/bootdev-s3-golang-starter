package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 1 << 30
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

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

	fmt.Println("uploading video", videoID, "by user", userID)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "User is not the video owner", err)
		return
	}

	videofile, header, err := r.FormFile("video")
	if err != nil {
		if mb_err, ismaxbytes := errors.AsType[*http.MaxBytesError](err); ismaxbytes {
			respondWithError(w, http.StatusRequestEntityTooLarge, "Video is larger than 1 GB", mb_err)
		} else {
			respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		}
		return
	}
	defer videofile.Close()

	mediatype, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid content type params", err)
		return
	}

	types := strings.Split(mediatype, "/")
	fileExtension := types[1]

	if strings.Compare(fileExtension, "mp4") != 0 {
		respondWithError(w, http.StatusBadRequest, "File is not an mp4", nil)
		return
	}

	newvideofile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't create temp video", err)
		return
	}

	defer os.Remove(fmt.Sprintf("%s%s", os.TempDir(), "/tubely-upload.mp4"))
	defer newvideofile.Close()

	if _, err := io.Copy(newvideofile, videofile); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy the old image file to the new image file", err)
		return
	}

	if _, err := newvideofile.Seek(0, io.SeekStart); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't set the \"offset\" of the file", err)
		return
	}

	key := make([]byte, 32)
	rand.Read(key)
	filename := base64.RawURLEncoding.EncodeToString(key) + "." + fileExtension

	bucketname := "tubely-12026"

	if _, err := cfg.s3Client.PutObject(
		r.Context(),
		&s3.PutObjectInput{
			Bucket:      &bucketname,
			Key:         &filename,
			Body:        newvideofile,
			ContentType: &mediatype,
		},
	); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't put video in s3 bucket", err)
		return
	}

	vidurl := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketname, cfg.s3Region, filename)
	video.VideoURL = &vidurl

	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video thumbnail url", err)
		return
	}
}
