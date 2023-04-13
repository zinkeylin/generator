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
    "strconv"
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
}
 
func validator(limitsStr, threadsStr string) (parameters, error) {
    var params parameters
    limits, err := strconv.Atoi(limitsStr)
    if err != nil {
        return params, err
    }
    if limits <= 0 {
        return params, errors.New("invalid value for limits")
    }
    threads, err := strconv.Atoi(threadsStr)
    if err != nil {
        return params, err
    }
    if threads <= 0 {
        return params, errors.New("invalid value for threads")
    }
    params.Limits = limits
    params.Threads = threads
    return params, nil
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
                fmt.Fprintln(w, "error: ", err)
            }
            // кладём параметры в params
            err = json.Unmarshal(bs, &params)
            if err != nil {
                fmt.Fprintln(w, "error: ", err)
            }
            // debug
            // fmt.Println(params)
        }
    })
	http.HandleFunc("/getNumbers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
            fmt.Fprintln(w, getNumbers(params.Limits, params.Threads))
		}
	})
    fmt.Println("Server is listening...\nlink:http://localhost:8080")
    http.ListenAndServe("localhost:8080", nil)
}