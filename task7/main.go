package main

import (
	"fmt"
	"sync"
	"time"
)

func or(channels ...<-chan interface{}) <-chan interface{} {
	//если каналов нет возвращаем nil
	if len(channels) == 0 {
		return nil
	}
	done := make(chan interface{})
	var once sync.Once //гарантирует, что close(done), выполниться только 1 раз
	//для каждого канала запускам горутину
	for _, ch := range channels {
		go func(c <-chan interface{}) {
			//ждем закрытия или значения
			<-c
			//закрываем общий канал
			once.Do(func() {
				close(done)
			})
		}(ch)
	}
	return done
}

func main() {

	sig := func(after time.Duration) <-chan interface{} {
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(after)
		}()
		return c
	}

	start := time.Now()
	<-or(
		sig(2*time.Hour),
		sig(5*time.Minute),
		sig(1*time.Second),
		sig(1*time.Hour),
		sig(1*time.Minute),
	)
	fmt.Printf("done after %v", time.Since(start))

}
