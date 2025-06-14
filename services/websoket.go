package services

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"new/dto"
	"new/utils"

	"github.com/gorilla/websocket"
)

// WebSocketHandler для обработки WebSocket-соединений
type WebSocketHandler struct {
	PlaceService *PlaceService
	Clients      map[*websocket.Conn]bool
	mu           sync.Mutex
}

// NewWebSocketHandler создаёт новый обработчик WebSocket
func NewWebSocketHandler(placeService *PlaceService) *WebSocketHandler {
	log.Printf("Инициализация нового WebSocketHandler")
	return &WebSocketHandler{
		PlaceService: placeService,
		Clients:      make(map[*websocket.Conn]bool),
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // В продакшене настрой проверку происхождения
	},
}

// HandleWebSocket обрабатывает WebSocket-соединения
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("Попытка нового WebSocket-соединения с %s", r.RemoteAddr)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка при переходе на WebSocket: %v, удалённый адрес: %s", err, r.RemoteAddr)
		http.Error(w, "Не удалось выполнить переход на WebSocket", http.StatusInternalServerError)
		return
	}
	log.Printf("WebSocket-соединение установлено для %s, время: %v", r.RemoteAddr, time.Since(startTime))

	// Добавляем клиента в список
	h.mu.Lock()
	h.Clients[conn] = true
	clientCount := len(h.Clients)
	h.mu.Unlock()
	log.Printf("Клиент добавлен, общее количество клиентов: %d", clientCount)

	defer func() {
		h.mu.Lock()
		delete(h.Clients, conn)
		clientCount = len(h.Clients)
		h.mu.Unlock()
		log.Printf("Клиент отключён, осталось клиентов: %d", clientCount)
		conn.Close()
	}()

	// Извлекаем токен из параметров URL
	query := r.URL.Query()
	token := query.Get("token")
	if token == "" {
		log.Printf("Отсутствует токен в параметрах URL для соединения с %s", r.RemoteAddr)
		if err := conn.WriteJSON(map[string]string{"error": "Токен отсутствует в параметрах URL"}); err != nil {
			log.Printf("Не удалось отправить ошибку отсутствия токена клиенту: %v", err)
		}
		return
	}
	log.Printf("Получен токен: %s", token)

	// Извлекаем userID из токена
	userID, err := utils.ExtractUserIDFromToken(token)
	if err != nil {
		log.Printf("Ошибка валидации токена для соединения с %s: %v", r.RemoteAddr, err)
		if err := conn.WriteJSON(map[string]string{"error": "Недействительный или истёкший токен"}); err != nil {
			log.Printf("Не удалось отправить ошибку недействительного токена клиенту: %v", err)
		}
		return
	}
	log.Printf("Пользователь аутентифицирован, userID: %s", userID)

	// Цикл обработки сообщений
	for {
		var osmObjects []dto.OSMObject
		log.Printf("Ожидание сообщения от userID: %s", userID)
		err := conn.ReadJSON(&osmObjects)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Неожиданное отключение клиента для userID: %s, ошибка: %v", userID, err)
			} else {
				log.Printf("Ошибка чтения JSON от userID: %s, ошибка: %v", userID, err)
			}
			break
		}
		log.Printf("Получено %d OSM-объектов от userID: %s", len(osmObjects), userID)

		// Канал для получения результатов обработки
		resultChan := make(chan map[string]interface{})
		log.Printf("Создан канал результатов для обработки OSM-объектов для userID: %s", userID)

		// Запускаем обработку в отдельной горутине
		go func() {
			defer close(resultChan)
			log.Printf("Запуск StreamProcessJSON для userID: %s с %d OSM-объектами", userID, len(osmObjects))
			startProcess := time.Now()
			h.PlaceService.StreamProcessJSON(userID, osmObjects, resultChan)
			log.Printf("StreamProcessJSON завершён для userID: %s, время: %v", userID, time.Since(startProcess))
		}()

		// Отправляем результаты по мере их поступления
		for result := range resultChan {
			// Сериализуем результат в JSON для подсчёта размера
			resultBytes, err := json.Marshal(result)
			if err != nil {
				log.Printf("Ошибка сериализации результата для userID: %s, ошибка: %v", userID, err)
				continue
			}
			log.Printf("Отправка результата userID: %s, размер результата: %d байт", userID, len(resultBytes))
			if err := conn.WriteJSON(result); err != nil {
				log.Printf("Ошибка отправки результата userID: %s, ошибка: %v", userID, err)
				break
			}
			log.Printf("Результат успешно отправлен userID: %s", userID)
		}
	}
	log.Printf("WebSocket-соединение закрыто для userID: %s, удалённый адрес: %s", userID, r.RemoteAddr)
}
