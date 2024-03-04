package util

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

func AskConfirmation(prompt string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}

		input = strings.ToLower(strings.TrimSpace(input))
		switch input {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
	}
}

func RandomDigits(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(48 + rand.Intn(10))
	}
	return string(b)
}
