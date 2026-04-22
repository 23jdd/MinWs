package main

import (
	"net/http"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := Upgrade(w, r)
		if err != nil {
			//fmt.Println(err)
		}
	})
	http.ListenAndServe(":8080", nil)
}
