package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	presignRequest, err := presignClient.PresignGetObject(context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
		func(po *s3.PresignOptions) {
			po.Expires = expireTime
		})
	if err != nil {
		return "", fmt.Errorf("Couldn't get a presigned request to get %s:%s: %w", bucket, key, err)
	}
	return presignRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	videoURLSplit := strings.Split(*video.VideoURL, ",")
	if len(videoURLSplit) != 2 {
		return database.Video{}, fmt.Errorf("Incorrect VideoURL format. Expected 'bucket':'key'")
	}
	bucket, key := videoURLSplit[0], videoURLSplit[1]
	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*5)
	if err != nil {
		return database.Video{}, fmt.Errorf("Could not generate presigned URL: %w", err)
	}
	// Should I just create a new video object here? Will leave it like this for now
	video.VideoURL = &presignedURL
	return video, nil
}
