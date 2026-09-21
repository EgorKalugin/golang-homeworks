package main

import (
	"fmt"
	"os"
	"runtime"
)

func userName() string {
	for _, key := range []string{"USER", "LOGNAME", "USERNAME"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return "неизвестно"
}

func main() {
	fmt.Println("Пользователь:", userName())

	args := os.Args[1:]
	fmt.Printf("Аргументы (%d):\n", len(args))
	if len(args) == 0 {
		fmt.Println("  <аргументов нет>")
	}
	for i, arg := range args {
		fmt.Printf("  %d: %s\n", i+1, arg)
	}

	fmt.Println("Версия Go:", runtime.Version())
}
