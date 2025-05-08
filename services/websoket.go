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

	// Получаем токен из заголовка Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("Отсутствует заголовок Authorization")
		conn.WriteJSON(map[string]string{"error": "Authorization header missing"})
		return
	}

	// Извлекаем userID из токена с помощью utils
	userID, err := utils.ExtractUserIDFromToken(authHeader)
	if err != nil {
		log.Printf("Ошибка авторизации: %v", err)
		conn.WriteJSON(map[string]string{"error": "Invalid or expired token"})
		return
	}

	//Цикл обработки сообщений
	for {
		var osmObjects []dto.OSMObject
		err := conn.ReadJSON(&osmObjects)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Клиент отключился: %v", err)
			}
			break
		}

		// Обрабатываем данные через PlaceService
		results, err := h.PlaceService.ProcessJSON(userID, osmObjects)
		if err != nil {
			log.Printf("Ошибка обработки: %v", err)
			conn.WriteJSON(map[string]string{"error": err.Error()})
			continue
		}

		// Отправляем результат клиенту
		if err := conn.WriteJSON(results); err != nil {
			log.Printf("Ошибка отправки: %v", err)
			break
		}
	}
}
