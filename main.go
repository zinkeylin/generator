package main
 
import (
    "encoding/json"
    "errors"
    "fmt"
    "math/rand"
    "sync"
    "time"
    "io/ioutil"
    "net/http"
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
	var numbers []int
    // запуск threads пушеров
    for i := 0; i < threads; i++ {
        curr := make(chan int)
        chans = append(chans, curr)
        go pusher(curr, limits)
    }
 
    // получаем числа из чекера
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
    Correct bool
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
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "html/index.html")
    })
    http.HandleFunc("/pushParams", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodPost {
            bs, err := ioutil.ReadAll(r.Body)
            if err != nil {
                fmt.Println(w, "error: ", err)
            }
            // кладём параметры в params
            err = json.Unmarshal(bs, &params)
            if err != nil {
                fmt.Fprintln(w, "error: ", err)
            }
            err = validator(params)
            if err != nil {
                fmt.Fprintln(w, "error: ", err)
                params.Correct = false
            } else {
                params.Correct = true
            }
        }
    })
	http.HandleFunc("/getNumbers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && params.Correct {
            fmt.Fprintln(w, getNumbers(params.Limits, params.Threads))
		}
	})
    fmt.Println("Server is listening...\nlink:http://localhost:8080")
    http.ListenAndServe("localhost:8080", nil)
}
