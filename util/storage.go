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

var allowedMilestoneFileTypes = map[string]bool{
	"image/jpeg":         true,
	"image/png":          true,
	"image/webp":         true,
	"application/pdf":    true,
	"text/plain":         true,
	"text/csv":           true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"application/zip":              true,
	"application/x-zip-compressed": true,
}

var allowedMilestoneExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".pdf":  true,
	".txt":  true,
	".csv":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".zip":  true,
}

const maxMilestoneFileSize = 5 * 1024 * 1024 // 5MB

func normalizeContentType(ct string) string {
	return strings.TrimSpace(strings.Split(ct, ";")[0])
}

func ValidatePhoto(header *multipart.FileHeader) error {
	if header.Size > maxMilestoneFileSize {
		return fmt.Errorf("file too large: max 5MB")
	}

	ct := normalizeContentType(header.Header.Get("Content-Type"))
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedMilestoneFileTypes[ct] && !allowedMilestoneExtensions[ext] {
		return fmt.Errorf("invalid file type: allowed JPG, PNG, WebP, PDF, TXT, CSV, DOC, DOCX, XLS, XLSX, ZIP")
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
	return allowedImageTypes[normalizeContentType(ct)]
}
