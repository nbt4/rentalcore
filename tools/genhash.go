package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <password>\n", os.Args[0])
		os.Exit(2)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(os.Args[1]), 14)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(hash))
}
