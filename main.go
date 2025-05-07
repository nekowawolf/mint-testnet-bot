package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nekowawolf/mint-testnet-bot/dapps"
)

func main() {
	fmt.Println("\nSelect dApp:")
	fmt.Println("1. CAP")
	fmt.Print("Enter your choice: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		fmt.Println("\nDApp selected: CAP")
		
		fmt.Print("\nEnter number of mints: ")
		numInput, _ := reader.ReadString('\n')
		numInput = strings.TrimSpace(numInput)

		numMints, err := strconv.Atoi(numInput)
		if err != nil || numMints < 1 {
			fmt.Println("Invalid number. Please enter a positive integer.")
			os.Exit(1)
		}

		fmt.Printf("\nDApp: CAP\n")
		fmt.Printf("Mints: %d\n\n", numMints)

		dapps.CUSD(numMints)
	default:
		fmt.Println("Invalid choice")
		os.Exit(1)
	}
}