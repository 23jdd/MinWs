package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		client, err := Upgrade(w, r)
		if err != nil {
			fmt.Println(err)
			return
		}
		// register message
		client.OnMessage = func(data []byte) {
			client.SendText(string(data)) //
			fmt.Println(string(data))
		}
		client.Listen()
	})
	http.ListenAndServe(":8080", nil)
}
