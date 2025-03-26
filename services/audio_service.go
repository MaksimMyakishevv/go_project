package services

import (
	"context"
	"errors"
	"fmt"
	"log"
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

func (s *AudioService) GetAllAudio() ([]models.Audio, error) {
	var audio []models.Audio

	if err := s.DB.Find(&audio).Error; err != nil {
		return nil, err
	}

	return audio, nil
}

func (s *AudioService) GetFiles() error {

	// Подгружаем конфигурацию из ~/.aws/*
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	// Создаем клиента для доступа к хранилищу S3
	client := s3.NewFromConfig(cfg)

	// Запрашиваем список бакетов
	result, err := client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		log.Fatal(err)
	}

	for _, bucket := range result.Buckets {
		log.Printf("bucket=%s creation time=%s", aws.ToString(bucket.Name), bucket.CreationDate.Local().Format("2006-01-02 15:04:05 Monday"))
	}
	return nil
}

func (s *AudioService) LoadFile(file models.UploadedFile, ctx context.Context) (string, error) {
	bucketName := os.Getenv("BUCKET_NAME")
	keyPrefix := os.Getenv("KEY_PREFIX2") // Если нужно сохранять файл в папке бакета - то тут путь будет
	region := os.Getenv("AWS_REGION")     // Регион AWS

	// Подгружаем конфигурацию из ~/.aws/*
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	// Создаем клиента для доступа к хранилищу S3
	client := s3.NewFromConfig(cfg)

	// Формируем полный ключ объекта
	objectKey := keyPrefix + file.Filename
	fileName := file.Filename

	// Загружаем файл в S3
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
		return "", err
	}

	// Проверяем существование объекта
	err = s3.NewObjectExistsWaiter(client).Wait(
		ctx, &s3.HeadObjectInput{Bucket: aws.String(bucketName), Key: aws.String(objectKey)}, time.Minute)
	if err != nil {
		log.Printf("Failed attempt to wait for object %s to exist.\n", objectKey)
		return "", err
	}

	// Генерируем публичную ссылку на файл
	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, region, objectKey)

	// Возвращаем ссылку на файл
	return fileURL, nil
}

