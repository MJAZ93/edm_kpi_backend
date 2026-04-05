package util

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

const maxPhotoSize = 5 * 1024 * 1024 // 5MB

func ValidatePhoto(header *multipart.FileHeader) error {
	if header.Size > maxPhotoSize {
		return fmt.Errorf("file too large: max 5MB")
	}

	ct := header.Header.Get("Content-Type")
	if !allowedImageTypes[ct] {
		return fmt.Errorf("invalid file type: only JPEG, PNG, WebP allowed")
	}
	return nil
}

func UploadPhoto(file multipart.File, header *multipart.FileHeader) (string, error) {
	backend := os.Getenv("STORAGE_BACKEND")
	if backend == "s3" {
		return uploadToS3(file, header)
	}
	return uploadToLocal(file, header)
}

func uploadToLocal(file multipart.File, header *multipart.FileHeader) (string, error) {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	dir := filepath.Join(uploadDir, "milestones")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	ext := filepath.Ext(header.Filename)
	filename := uuid.New().String() + ext
	path := filepath.Join(dir, filename)

	out, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}

	return "/uploads/milestones/" + filename, nil
}

func uploadToS3(file multipart.File, header *multipart.FileHeader) (string, error) {
	// S3 upload placeholder — implement with aws-sdk-go-v2 when needed
	ext := filepath.Ext(header.Filename)
	key := fmt.Sprintf("milestones/%s%s", uuid.New().String(), ext)
	bucket := os.Getenv("AWS_BUCKET")
	region := os.Getenv("AWS_REGION")

	_ = file // TODO: actual S3 PutObject

	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, key)
	return url, nil
}

func IsValidImageContentType(ct string) bool {
	ct = strings.Split(ct, ";")[0]
	return allowedImageTypes[strings.TrimSpace(ct)]
}
