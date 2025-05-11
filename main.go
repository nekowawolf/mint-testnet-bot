package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nekowawolf/mint-testnet-bot/dapps"
)

func main() {
	fmt.Println("\nSelect Dapps:")
	fmt.Println("1. Cap")
	fmt.Print("\nEnter your choice: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		dapps.Cap()
	default:
		fmt.Println("Invalid choice. Please select a valid option.")
		os.Exit(1)
	}
}
