package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := []byte("TsunamiRental2025")
	hash, err := bcrypt.GenerateFromPassword(password, 14)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(hash))
}
