package main

import (
	"fmt"
	"os"

	"s2qt/service"
)

func main() {
	result, err := service.CheckRuntimeForPNG(true)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}

	fmt.Printf("OK: %v\n", result.OK)
	fmt.Printf("Checked: %v\n", result.Checked)
	fmt.Printf("Installed: %v\n", result.Installed)
	fmt.Printf("Missing: %v\n", result.Missing)
	fmt.Printf("Message: %s\n", result.Message)
}
