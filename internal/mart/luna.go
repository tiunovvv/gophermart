package mart

import "strconv"

func (m *Mart) CheckLunaAlgorithm(cardNumber string) bool {
	const (
		digitsCount = 9
		shift       = 2
	)

	digits := make([]int, len(cardNumber))
	for i, char := range cardNumber {
		digit, err := strconv.Atoi(string(char))
		if err != nil {
			return false
		}
		digits[i] = digit
	}

	for i := len(digits) - shift; i >= 0; i -= shift {
		digits[i] *= shift
		if digits[i] > digitsCount {
			digits[i] -= digitsCount
		}
	}

	sum := 0
	for _, digit := range digits {
		sum += digit
	}

	return sum%10 == 0
}
