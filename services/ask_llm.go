package services
import (
	"bytes"
    "encoding/json"
    "net/http"
	"new/models"
)


type AskLLMService struct{}

func (s *AskLLMService) AskLLMQuestion(question models.Question) (string, error) {
	// Конвертируем вопрос в json
    jsonData, err := json.Marshal(question)
    if err != nil {
        return "", err
    }

    // Отправка POST запроса на FastAPI сервер открытый по порту 8000
    resp, err := http.Post("http://localhost:5555/ask/", "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // Проверка, что есть ответ
    if resp.StatusCode != http.StatusOK {
        return "", err
    }

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    
    // Возвращает ответ от FastAPI сервера
    return result["message"].(string), nil
}