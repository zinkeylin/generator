// websockets.go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"sync"
	"time"
	"strings"
)

// Срез каналов для прослушки чекером
var chans = []chan int{}

// пушер, генерирующий рандомное число в свой канал
func pusher(c chan<- int, limits int) {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	for {
		c <- r.Intn(limits + 1)
	}
}

// чекер, проверяющий уникальность чисел, кладёт уникальные в numbers
func checker(limits int, numbers *[]int) {
	uniqueChecker := make(map[int]bool)
	for {
		// прослушка каждого канала
		for _, c := range chans {
			select {
			case num := <-c:
				{
					if !uniqueChecker[num] {
						uniqueChecker[num] = true
						*numbers = append(*numbers, num)
						limits--
						if limits == 0 {
							return
						}
					}
				}
			}
		}
	}
}

func getNumbers(limits, threads int) []int {
	// запуск threads сендеров
	for i := 0; i < threads; i++ {
		curr := make(chan int)
		chans = append(chans, curr)
		go pusher(curr, limits)
	}

	// получаем числа из чекера
	var numbers []int
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		checker(limits, &numbers)
	}()
	wg.Wait()
	return numbers
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

func arrayToString(a []int, delim string) string {
    return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
}

func main() {
	var params parameters
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

		for {
			// Read message from browser
			msgType, bs, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("error: ", err)
				err = conn.WriteMessage(msgType, []byte(string(err.Error())))
				if err != nil {
					fmt.Println("Connection error: ", err)
				}
			}
			err = json.Unmarshal(bs, &params)
			if err != nil {
				fmt.Println("error: ", err)
				err = conn.WriteMessage(msgType, []byte(string(err.Error())))
				if err != nil {
					fmt.Println("Connection error: ", err)
				}
				params.Correct = false

			}
			err = validator(params)
			if err != nil {
				fmt.Println("error: ", err)
				err = conn.WriteMessage(msgType, []byte(string(err.Error())))
				if err != nil {
					fmt.Println("Connection error: ", err)
				}
				params.Correct = false
			} else {
				params.Correct = true
			}
			
			if params.Correct {
				err = conn.WriteMessage(msgType, []byte("\n" + arrayToString(getNumbers(params.Limits, params.Threads), "\n")))
				if err != nil {
					fmt.Println("error: ", err)
					err = conn.WriteMessage(msgType, []byte(string(err.Error())))
					if err != nil {
						fmt.Println("Connection error: ", err)
					}
				}
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	http.ListenAndServe(":8080", nil)
}
