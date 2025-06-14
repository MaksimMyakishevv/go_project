package services

import (
	"log"
	"net/http"
	"sync"

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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка подключения WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Добавляем клиента в список
	h.mu.Lock()
	h.Clients[conn] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.Clients, conn)
		h.mu.Unlock()
	}()

	// Извлекаем токен из параметров URL
	query := r.URL.Query()
	token := query.Get("token")
	if token == "" {
		log.Printf("Отсутствует токен в параметрах URL")
		conn.WriteJSON(map[string]string{"error": "Token missing in URL parameters"})
		return
	}

	// Извлекаем userID из токена с помощью utils
	userID, err := utils.ExtractUserIDFromToken(token)
	if err != nil {
		log.Printf("Ошибка авторизации: %v", err)
		conn.WriteJSON(map[string]string{"error": "Invalid or expired token"})
		return
	}

	// Цикл обработки сообщений
	for {
		var osmObjects []dto.OSMObject
		err := conn.ReadJSON(&osmObjects)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Клиент отключился: %v", err)
			}
			break
		}

		// Канал для получения результатов обработки
		resultChan := make(chan map[string]interface{})

		// Запускаем обработку в отдельной горутине
		go func() {
			defer close(resultChan)
			h.PlaceService.StreamProcessJSON(userID, osmObjects, resultChan)
		}()

		// Отправляем результаты по мере их поступления
		for result := range resultChan {
			if err := conn.WriteJSON(result); err != nil {
				log.Printf("Ошибка отправки результата: %v", err)
				break
			}
		}
	}
}
