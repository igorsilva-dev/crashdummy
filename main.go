package main

import (
	"fmt"
	"log"
	"github.com/igorsilva-dev/crashdummy/app/handlers"
	"net/http"
)

func main() {

	handlers.Initiate()

	fmt.Println("")
	fmt.Println("listening at localhost:10000...")
	log.Fatal(http.ListenAndServe(":10000", nil))
}
