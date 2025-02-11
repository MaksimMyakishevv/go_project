package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"new/dto"
	"new/models"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"gorm.io/gorm"
	// smithyendpoints "github.com/aws/smithy-go/endpoints"
)

type AudioService struct {
	DB *gorm.DB
}


func (s *AudioService) SaveAudio(input dto.AudioDTO) (*models.Audio, error) {

	file_path := &models.Audio{
		Path:    input.Path,
	}

	// Сохраняем в базе данных
	if err := s.DB.Create(file_path).Error; err != nil {
		return nil, err
	}

	return file_path, nil
}

func (s *AudioService) GetAllAudio() ([]models.Audio, error) {
	var audio []models.Audio

	if err := s.DB.Find(&audio).Error; err != nil {
		return nil, err
	}

	return audio, nil
}



func (s *AudioService) GetFiles() error {

	bucketName := os.Getenv("BUCKET_NAME")

    // Подгружаем конфигурацию из ~/.aws/*
    cfg, err := config.LoadDefaultConfig(context.TODO())
    if err != nil {
        log.Fatal(err)
		return err
    }

    // Создаем клиента для доступа к хранилищу S3
    client := s3.NewFromConfig(cfg)

    // Запрашиваем список всех файлов в бакете
    result, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
        Bucket: aws.String(bucketName),
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, object := range result.Contents {
        log.Printf("object=%s size=%d Bytes last modified=%s", aws.ToString(object.Key), aws.ToInt64(object.Size), object.LastModified.Local().Format("2006-01-02 15:04:05 Monday"))
    }

	return nil
}

func (s *AudioService) LoadFile(file models.UploadedFile, ctx context.Context) error {

	bucketName := os.Getenv("BUCKET_NAME")
	keyPrefix := os.Getenv("KEY_PREFIX2") //если нужно сохранять файл в папке бакета - то тут путь будет

    // Подгружаем конфигурацию из ~/.aws/*
    cfg, err := config.LoadDefaultConfig(context.TODO())
    if err != nil {
        log.Fatal(err)
		return err
    }

    // Создаем клиента для доступа к хранилищу S3
    client := s3.NewFromConfig(cfg)

	objectKey := keyPrefix + file.Filename
	fileName := file.Filename

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   file.File,
			})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "EntityTooLarge" {
			log.Printf("Error while uploading object to %s. The object is too large.\n"+
				"To upload objects larger than 5GB, use the S3 console (160GB max)\n"+
				"or the multipart upload API (5TB max).", bucketName)
		} else {
			log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n",
				fileName, bucketName, objectKey, err)
		}
	err = s3.NewObjectExistsWaiter(client).Wait(
		ctx, &s3.HeadObjectInput{Bucket: aws.String(bucketName), Key: aws.String(objectKey)}, time.Minute)
	if err != nil {
		log.Printf("Failed attempt to wait for object %s to exist.\n", objectKey)
	}

	return err
	}

	return err
}

// На данный момент эта функция нигде не используется, и я не проверяла работает ли вообще
func (s *AudioService) GenerateAudio(req models.TTSRequest) ([]byte, error) {
	// Convert the request body to JSON
	jsonBody := new(bytes.Buffer)

	err := json.NewEncoder(jsonBody).Encode(req)
	if err != nil {
		return nil, fmt.Errorf("error encoding JSON: %w", err)
	}

	fastapiUrl := "http://localhost:8001/generate_audio"
	// Forward the request to FastAPI
	resp, err := http.Post(fastapiUrl, "application/json", jsonBody)
	if err != nil {
		return nil, fmt.Errorf("error forwarding request to FastAPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FastAPI error: %s", string(body))
	}

	// Read and return the audio data from the response body
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading audio data: %w", err)
	}

	return audioData, nil
}