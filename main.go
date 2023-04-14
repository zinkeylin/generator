// websockets.go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// Срез каналов для прослушки чекером
var chans = []chan int{}

// Флаг-статус генерации, для остановки пушеров
var neededPush = false

// пушер, генерирующий рандомное число в свой канал
func pusher(c chan<- int, limits int) {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	for {
		if !neededPush {
			break
		}
		c <- r.Intn(limits + 1)
	}
}

// проверяет уникальность чисел, кладёт уникальные в канал
func getNumber(out chan int) {
	uniqueChecker := make(map[int]bool)
	for {
		if len(chans) == 0 {
			break
		}
		// прослушка каждого канала
		for _, c := range chans {
			num, ok := <-c
			if ok {
				if !uniqueChecker[num] {
					uniqueChecker[num] = true
					out <- num
				}
			}
		}
	}
}

// посылка с параметрами
type parameters struct {
	Limits, Threads int
	Correct         bool
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func validator(params parameters) error {
	if params.Limits <= 0 {
		return errors.New("invalid value for limits")
	}
	if params.Threads <= 0 {
		return errors.New("invalid value for threads")
	}
	return nil
}

func main() {
	var params parameters
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

		for {
			// обрабатываем сообщение от клиента
			_, bs, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("error: ", err)
				err = conn.WriteMessage(websocket.TextMessage, []byte(string(err.Error())))
				if err != nil {
					fmt.Println("Connection error: ", err)
				}
			}
			// обрабатываем параметры
			err = json.Unmarshal(bs, &params)
			if err != nil {
				fmt.Println("error: ", err)
				err = conn.WriteMessage(websocket.TextMessage, []byte(string(err.Error())))
				if err != nil {
					fmt.Println("Connection error: ", err)
				}
				params.Correct = false

			}
			// проверка корректности
			err = validator(params)
			if err != nil {
				fmt.Println("error: ", err)
				err = conn.WriteMessage(websocket.TextMessage, []byte(string(err.Error())))
				if err != nil {
					fmt.Println("Connection error: ", err)
				}
				params.Correct = false
			} else {
				params.Correct = true
			}

			if params.Correct {
				// разрешаем пушить числа
				neededPush = true
				// запуск threads пушеров
				for i := 0; i < params.Threads; i++ {
					curr := make(chan int)
					chans = append(chans, curr)
					go pusher(curr, params.Limits)
				}
				// Канал для приёма чисел
				out := make(chan int)
				// запуск геттера (чекера уникальности)
				go getNumber(out)
				for i := 0; i < params.Limits; i++ {
					num := <-out
					err = conn.WriteMessage(websocket.TextMessage, []byte(strconv.Itoa(num)))
					if err != nil {
						fmt.Println("error: ", err)
						err = conn.WriteMessage(websocket.TextMessage, []byte(string(err.Error())))
						if err != nil {
							fmt.Println("Connection error: ", err)
						}
					}
				}
				// очищаем срез каналов
				chans = []chan int{}
				// запрещаем пушить числа
				neededPush = false
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "html/index.html")
	})

	fmt.Println("Server is listening...\nlink:http://localhost:8080")
	http.ListenAndServe("localhost:8080", nil)
}
